package catalog

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"

	"aihub/internal/providers"
)

const (
	pricingCurrencyUSD       = "USD"
	pricingUnitPer1MTokens   = "per_1m_tokens"
	pricingSourceProviderCat = "provider_catalog"
	tokensPerMillion         = 1_000_000.0
)

// ProviderReportedPricing is a read-time projection of list pricing from
// provider_metadata. It is not execution cost and not Hub billing.
type ProviderReportedPricing struct {
	Currency   string
	Unit       string
	Input      *float64
	Output     *float64
	CacheRead  *float64
	CacheWrite *float64
	Source     string
	Complete   bool
}

// ExtractProviderReportedPricing parses provider catalog metadata into USD per 1M
// tokens when the upstream publishes recognizable fields.
func ExtractProviderReportedPricing(providerID string, metadata json.RawMessage) (ProviderReportedPricing, bool) {
	if len(metadata) == 0 || string(metadata) == "null" {
		return ProviderReportedPricing{}, false
	}

	switch strings.TrimSpace(providerID) {
	case providers.OpenCodeGoID:
		return extractOpenCodeGoPricing(metadata)
	case providers.OpenRouterID:
		return extractOpenRouterPricing(metadata)
	default:
		if p, ok := extractOpenRouterPricing(metadata); ok {
			return p, true
		}
		if p, ok := extractOpenCodeGoPricing(metadata); ok {
			return p, true
		}
		return ProviderReportedPricing{}, false
	}
}

func extractOpenCodeGoPricing(metadata json.RawMessage) (ProviderReportedPricing, bool) {
	var payload struct {
		Cost *struct {
			Text struct {
				Input struct {
					Price float64 `json:"price"`
					Cache struct {
						Read  *float64 `json:"read"`
						Write *float64 `json:"write"`
					} `json:"cache"`
				} `json:"input"`
				Output struct {
					Price float64 `json:"price"`
				} `json:"output"`
			} `json:"text"`
		} `json:"cost"`
	}
	if err := json.Unmarshal(metadata, &payload); err != nil || payload.Cost == nil {
		return ProviderReportedPricing{}, false
	}

	text := payload.Cost.Text
	input := text.Input.Price
	output := text.Output.Price
	pricing := ProviderReportedPricing{
		Currency: pricingCurrencyUSD,
		Unit:     pricingUnitPer1MTokens,
		Input:    &input,
		Output:   &output,
		Source:   pricingSourceProviderCat,
		Complete: true,
	}
	if text.Input.Cache.Read != nil {
		pricing.CacheRead = text.Input.Cache.Read
	}
	if text.Input.Cache.Write != nil {
		pricing.CacheWrite = text.Input.Cache.Write
	}
	return pricing, true
}

func extractOpenRouterPricing(metadata json.RawMessage) (ProviderReportedPricing, bool) {
	var payload struct {
		Pricing map[string]json.RawMessage `json:"pricing"`
	}
	if err := json.Unmarshal(metadata, &payload); err != nil || len(payload.Pricing) == 0 {
		return ProviderReportedPricing{}, false
	}

	inputPerToken, inputOK := parsePricingTokenRate(payload.Pricing, "prompt")
	outputPerToken, outputOK := parsePricingTokenRate(payload.Pricing, "completion")
	if !inputOK || !outputOK {
		return ProviderReportedPricing{}, false
	}

	input := scalePerTokenToPerMillion(inputPerToken)
	output := scalePerTokenToPerMillion(outputPerToken)
	pricing := ProviderReportedPricing{
		Currency: pricingCurrencyUSD,
		Unit:     pricingUnitPer1MTokens,
		Input:    &input,
		Output:   &output,
		Source:   pricingSourceProviderCat,
		Complete: true,
	}

	if cacheRead, ok := parsePricingTokenRate(payload.Pricing, "input_cache_read"); ok {
		scaled := scalePerTokenToPerMillion(cacheRead)
		pricing.CacheRead = &scaled
	}
	if cacheWrite, ok := parsePricingTokenRate(payload.Pricing, "input_cache_write"); ok {
		scaled := scalePerTokenToPerMillion(cacheWrite)
		pricing.CacheWrite = &scaled
	}

	return pricing, true
}

func parsePricingTokenRate(pricing map[string]json.RawMessage, key string) (float64, bool) {
	raw, ok := pricing[key]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}

	var asNumber float64
	if err := json.Unmarshal(raw, &asNumber); err == nil {
		if math.IsNaN(asNumber) || math.IsInf(asNumber, 0) {
			return 0, false
		}
		return asNumber, true
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err != nil {
		return 0, false
	}
	asString = strings.TrimSpace(asString)
	if asString == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(asString, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	return value, true
}

func scalePerTokenToPerMillion(perToken float64) float64 {
	return perToken * tokensPerMillion
}
