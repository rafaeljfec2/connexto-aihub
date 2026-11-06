package metering

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"aihub/internal/metering/capture"
	"aihub/internal/providers"
)

// NormalizeExecutionUsage maps provider-specific usage bytes to the Hub contract.
func NormalizeExecutionUsage(providerID string, capture capture.Result) Record {
	source := meteringSource(providerID)
	if len(capture.FinalEventData) == 0 && len(capture.ResponseBodyTail) == 0 {
		return unavailableRecord(source)
	}
	rawUsage, ok := extractUsageJSON(capture)
	if !ok {
		return unavailableRecord(source)
	}
	switch providerID {
	case providers.OpenCodeGoID:
		return normalizeOpenCodeUsage(rawUsage, source)
	case providers.OpenRouterID:
		return normalizeOpenRouterUsage(rawUsage, source)
	case providers.CursorID, providers.LegacyTransportServiceID:
		return normalizeConnextoProxyUsage(rawUsage, source)
	case providers.AntigravityID:
		return normalizeConnextoProxyUsage(rawUsage, source)
	default:
		return unavailableRecord(source)
	}
}

func meteringSource(providerID string) Source {
	switch providerID {
	case providers.OpenCodeGoID:
		return SourceOpenCodeGo
	case providers.OpenRouterID:
		return SourceOpenRouter
	case providers.CursorID, providers.LegacyTransportServiceID:
		return SourceCursor
	case providers.AntigravityID:
		return SourceAntigravity
	default:
		return SourceUnknown
	}
}

func unavailableRecord(source Source) Record {
	return Record{Status: StatusUnavailable, Source: source}
}

func extractUsageJSON(capture capture.Result) (json.RawMessage, bool) {
	if len(capture.FinalEventData) > 0 {
		return usageFromEventJSON(capture.FinalEventData)
	}
	if len(capture.ResponseBodyTail) > 0 {
		return usageFromResponseJSON(capture.ResponseBodyTail)
	}
	return nil, false
}

func usageFromEventJSON(data []byte) (json.RawMessage, bool) {
	var envelope struct {
		Usage    json.RawMessage `json:"usage"`
		Response struct {
			Usage json.RawMessage `json:"usage"`
		} `json:"response"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, false
	}
	if len(envelope.Response.Usage) > 0 && string(envelope.Response.Usage) != "null" {
		return envelope.Response.Usage, true
	}
	if len(envelope.Usage) > 0 && string(envelope.Usage) != "null" {
		return envelope.Usage, true
	}
	return nil, false
}

func usageFromResponseJSON(body []byte) (json.RawMessage, bool) {
	var response struct {
		Usage json.RawMessage `json:"usage"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, false
	}
	if len(response.Usage) == 0 || string(response.Usage) == "null" {
		return nil, false
	}
	return response.Usage, true
}

func normalizeOpenCodeUsage(raw json.RawMessage, source Source) Record {
	record := Record{Source: source}
	usage, err := decodeOpenCodeUsage(raw)
	if err != nil {
		record.Status = StatusInvalid
		return record
	}
	record.InputTokens = int64PtrFromUint(usage.inputTokens, usage.inputTokensSet)
	record.OutputTokens = int64PtrFromUint(usage.outputTokens, usage.outputTokensSet)
	record.TotalTokens = int64PtrFromUint(usage.totalTokens, usage.totalTokensSet)
	record.CachedInputTokens = int64PtrFromUint(usage.cacheRead, usage.cacheReadSet)
	if usage.costSet {
		cost, currency, err := parseReportedCost(usage.costRaw, "USD")
		if err != nil {
			record.Status = StatusInvalid
			return record
		}
		record.ProviderReportedCost = cost
		record.Currency = currency
	}
	if usage.listCostSet {
		listCost, _, err := parseReportedCost(usage.listCostRaw, "USD")
		if err != nil {
			record.Status = StatusInvalid
			return record
		}
		record.ProviderListCost = listCost
		if record.Currency == "" {
			record.Currency = "USD"
		}
	}
	record.Status = classifyTokenStatus(record)
	return record
}

func decodeOpenCodeUsage(raw json.RawMessage) (openCodeUsageDecoded, error) {
	var payload struct {
		InputTokens  *uint64         `json:"input_tokens"`
		OutputTokens *uint64         `json:"output_tokens"`
		TotalTokens  *uint64         `json:"total_tokens"`
		Cost         json.RawMessage `json:"cost"`
		OpenCode     *struct {
			ListCost json.RawMessage `json:"list_cost"`
			Tokens   *struct {
				CacheRead *uint64 `json:"cache_read"`
			} `json:"tokens"`
		} `json:"opencode"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return openCodeUsageDecoded{}, err
	}
	out := openCodeUsageDecoded{}
	if payload.InputTokens != nil {
		out.inputTokens = *payload.InputTokens
		out.inputTokensSet = true
	}
	if payload.OutputTokens != nil {
		out.outputTokens = *payload.OutputTokens
		out.outputTokensSet = true
	}
	if payload.TotalTokens != nil {
		out.totalTokens = *payload.TotalTokens
		out.totalTokensSet = true
	}
	if len(payload.Cost) > 0 && string(payload.Cost) != "null" {
		out.costRaw = payload.Cost
		out.costSet = true
	}
	if payload.OpenCode != nil && payload.OpenCode.Tokens != nil && payload.OpenCode.Tokens.CacheRead != nil {
		out.cacheRead = *payload.OpenCode.Tokens.CacheRead
		out.cacheReadSet = true
	}
	if payload.OpenCode != nil && len(payload.OpenCode.ListCost) > 0 && string(payload.OpenCode.ListCost) != "null" {
		out.listCostRaw = payload.OpenCode.ListCost
		out.listCostSet = true
	}
	return out, nil
}

type openCodeUsageDecoded struct {
	inputTokens, outputTokens, totalTokens, cacheRead uint64
	inputTokensSet, outputTokensSet, totalTokensSet, cacheReadSet, costSet, listCostSet bool
	costRaw, listCostRaw json.RawMessage
}

func normalizeOpenRouterUsage(raw json.RawMessage, source Source) Record {
	record := Record{Source: source}
	usage, err := decodeOpenRouterUsage(raw)
	if err != nil {
		record.Status = StatusInvalid
		return record
	}
	record.InputTokens = int64PtrFromUint(usage.inputTokens, usage.inputTokensSet)
	record.OutputTokens = int64PtrFromUint(usage.outputTokens, usage.outputTokensSet)
	record.TotalTokens = int64PtrFromUint(usage.totalTokens, usage.totalTokensSet)
	record.CachedInputTokens = int64PtrFromUint(usage.cachedInputTokens, usage.cachedInputTokensSet)
	if usage.costSet {
		cost, currency, err := parseReportedCost(usage.costRaw, usage.currency)
		if err != nil {
			record.Status = StatusInvalid
			return record
		}
		record.ProviderReportedCost = cost
		record.Currency = currency
	}
	record.Status = classifyTokenStatus(record)
	return record
}

func normalizeConnextoProxyUsage(raw json.RawMessage, source Source) Record {
	record := Record{Source: source}
	usage, err := decodeOpenRouterUsage(raw)
	if err != nil {
		record.Status = StatusInvalid
		return record
	}
	record.InputTokens = int64PtrFromUint(usage.inputTokens, usage.inputTokensSet)
	record.OutputTokens = int64PtrFromUint(usage.outputTokens, usage.outputTokensSet)
	record.TotalTokens = int64PtrFromUint(usage.totalTokens, usage.totalTokensSet)
	record.CachedInputTokens = int64PtrFromUint(usage.cachedInputTokens, usage.cachedInputTokensSet)
	if usage.costSet {
		cost, currency, err := parseReportedCost(usage.costRaw, usage.currency)
		if err != nil {
			record.Status = StatusInvalid
			return record
		}
		record.ProviderReportedCost = cost
		record.Currency = currency
	}
	record.Status = classifyTokenStatus(record)
	return record
}

func decodeOpenRouterUsage(raw json.RawMessage) (openRouterUsageDecoded, error) {
	var payload struct {
		InputTokens       *uint64         `json:"input_tokens"`
		OutputTokens      *uint64         `json:"output_tokens"`
		TotalTokens       *uint64         `json:"total_tokens"`
		CachedInputTokens *uint64         `json:"cached_input_tokens"`
		Cost              json.RawMessage `json:"cost"`
		Currency          *string         `json:"currency"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return openRouterUsageDecoded{}, err
	}
	out := openRouterUsageDecoded{}
	if payload.InputTokens != nil {
		out.inputTokens = *payload.InputTokens
		out.inputTokensSet = true
	}
	if payload.OutputTokens != nil {
		out.outputTokens = *payload.OutputTokens
		out.outputTokensSet = true
	}
	if payload.TotalTokens != nil {
		out.totalTokens = *payload.TotalTokens
		out.totalTokensSet = true
	}
	if payload.CachedInputTokens != nil {
		out.cachedInputTokens = *payload.CachedInputTokens
		out.cachedInputTokensSet = true
	}
	if len(payload.Cost) > 0 && string(payload.Cost) != "null" {
		out.costRaw = payload.Cost
		out.costSet = true
	}
	if payload.Currency != nil {
		out.currency = strings.TrimSpace(*payload.Currency)
	}
	return out, nil
}

type openRouterUsageDecoded struct {
	inputTokens, outputTokens, totalTokens, cachedInputTokens uint64
	inputTokensSet, outputTokensSet, totalTokensSet, cachedInputTokensSet, costSet bool
	costRaw  json.RawMessage
	currency string
}

func int64PtrFromUint(value uint64, set bool) *int64 {
	if !set {
		return nil
	}
	if value > math.MaxInt64 {
		return nil
	}
	out := int64(value)
	return &out
}

func classifyTokenStatus(record Record) Status {
	present := 0
	if record.InputTokens != nil {
		present++
	}
	if record.OutputTokens != nil {
		present++
	}
	if record.TotalTokens != nil {
		present++
	}
	if present == 0 {
		return StatusUnavailable
	}
	if record.TotalTokens != nil && record.InputTokens != nil && record.OutputTokens != nil {
		sum := *record.InputTokens + *record.OutputTokens
		if *record.TotalTokens != sum {
			return StatusInvalid
		}
	}
	if present == 3 || (record.InputTokens != nil && record.OutputTokens != nil) {
		return StatusReported
	}
	return StatusPartial
}

func parseReportedCost(raw json.RawMessage, defaultCurrency string) (pgtype.Numeric, string, error) {
	var numeric pgtype.Numeric
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return numeric, "", fmt.Errorf("missing cost")
	}
	var asNumber json.Number
	if err := json.Unmarshal(raw, &asNumber); err != nil {
		return numeric, "", err
	}
	if err := numeric.Scan(asNumber.String()); err != nil {
		return numeric, "", err
	}
	currency := strings.TrimSpace(defaultCurrency)
	if currency == "" {
		currency = "USD"
	}
	return numeric, currency, nil
}
