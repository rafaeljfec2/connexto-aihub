package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	cursorUsageQuotaSourceName = "cursor_proxy_usage_collect"
	cursorUsageCollectRoute    = "/providers/cursor/usage/collect"
	defaultCursorUsageTimeout  = 120 * time.Second
	cursorAccountQueryTimeout  = 130 * time.Second
)

// CursorUsageQuotaAdapter queries Cursor usage via Connexto Proxy browser collect.
type CursorUsageQuotaAdapter struct {
	httpClient   *http.Client
	allowedHosts map[string]struct{}
	accountID    string
}

// NewCursorUsageQuotaAdapter creates an adapter bound to a hub provider account id.
func NewCursorUsageQuotaAdapter(httpClient *http.Client, allowedHosts []string, accountID string) *CursorUsageQuotaAdapter {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultCursorUsageTimeout}
	}
	allowedMap := make(map[string]struct{}, len(allowedHosts))
	for _, host := range allowedHosts {
		if host = strings.ToLower(strings.TrimSpace(host)); host != "" {
			allowedMap[host] = struct{}{}
		}
	}
	return &CursorUsageQuotaAdapter{
		httpClient:   httpClient,
		allowedHosts: allowedMap,
		accountID:    strings.TrimSpace(accountID),
	}
}

type cursorCollectRequest struct {
	AccountID string `json:"accountId"`
}

type cursorCollectResponse struct {
	Status    string                 `json:"status"`
	Usage     map[string]interface{} `json:"usage"`
	AccountID string                 `json:"accountId"`
}

func newCursorUsageSnapshot(scope DataScope) ProviderQuotaSnapshot {
	now := time.Now().UTC().Format(time.RFC3339)
	return ProviderQuotaSnapshot{
		ProviderID:   TypeCursor,
		ProviderName: "Cursor",
		Scope:        scope,
		Status:       QuotaStatusError,
		Quotas:       []ProviderQuotaMetric{},
		Source: ProviderQuotaSource{
			Type:      "provider_api",
			Name:      cursorUsageQuotaSourceName,
			Scope:     scope,
			FetchedAt: now,
		},
	}
}

// FetchQuota posts to the proxy collect endpoint using the gateway management key.
func (a *CursorUsageQuotaAdapter) FetchQuota(ctx context.Context, endpoint, credential string) (ProviderQuotaSnapshot, error) {
	snapshot := newCursorUsageSnapshot(ScopeAccount)

	if strings.TrimSpace(credential) == "" {
		snapshot.Status = QuotaStatusUnconfiguredCredential
		snapshot.Error = &ProviderQuotaError{
			Code:    "unconfigured_credential",
			Message: "Gateway API Key não configurada para comunicação com o proxy",
		}
		return snapshot, nil
	}
	if a.accountID == "" {
		snapshot.Error = &ProviderQuotaError{Code: "invalid_account", Message: "Conta inválida para coleta Cursor"}
		return snapshot, nil
	}

	target, err := a.target(endpoint, cursorUsageCollectRoute)
	if err != nil {
		snapshot.Error = &ProviderQuotaError{Code: "invalid_endpoint", Message: err.Error()}
		return snapshot, nil
	}

	payload, err := json.Marshal(cursorCollectRequest{AccountID: a.accountID})
	if err != nil {
		return snapshot, fmt.Errorf("marshal collect body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(payload))
	if err != nil {
		return snapshot, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gateway-API-Key", credential)
	req.Header.Set("X-Hub-Account-Id", a.accountID)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		if os.IsTimeout(err) {
			snapshot.Status = QuotaStatusError
			snapshot.Error = &ProviderQuotaError{Code: "timeout", Message: "Tempo esgotado ao coletar uso Cursor"}
			return snapshot, nil
		}
		return snapshot, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	if err != nil {
		return snapshot, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		snapshot.Status = QuotaStatusUnauthorized
		snapshot.Error = &ProviderQuotaError{
			Code:    "authentication_required",
			Message: "Sessão Cursor no proxy requer autenticação assistida",
		}
		return snapshot, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snapshot.Error = &ProviderQuotaError{
			Code:    "upstream_error",
			Message: fmt.Sprintf("Proxy retornou HTTP %d", resp.StatusCode),
		}
		return snapshot, nil
	}

	var parsed cursorCollectResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return snapshot, fmt.Errorf("decode collect response: %w", err)
	}

	switch parsed.Status {
	case "available":
		snapshot.Status = QuotaStatusAvailable
		snapshot.Quotas = mapCursorUsageMetrics(parsed.Usage)
		return snapshot, nil
	case "authentication_required", "session_expired":
		snapshot.Status = QuotaStatusUnauthorized
		snapshot.Error = &ProviderQuotaError{
			Code:    parsed.Status,
			Message: "Autentique a conta Cursor no proxy para coletar uso",
		}
		return snapshot, nil
	default:
		snapshot.Error = &ProviderQuotaError{
			Code:    parsed.Status,
			Message: "Coleta Cursor não disponível",
		}
		return snapshot, nil
	}
}

func (a *CursorUsageQuotaAdapter) target(endpoint, path string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if base == "" {
		base = "http://127.0.0.1:9090/v1"
	}
	parsed, err := url.Parse(base + path)
	if err != nil {
		return "", err
	}
	if len(a.allowedHosts) > 0 {
		host := strings.ToLower(parsed.Hostname())
		if _, ok := a.allowedHosts[host]; !ok {
			return "", fmt.Errorf("host %q not allowed for cursor usage collect", host)
		}
	}
	return parsed.String(), nil
}

func mapCursorUsageMetrics(usage map[string]interface{}) []ProviderQuotaMetric {
	if usage == nil {
		return []ProviderQuotaMetric{}
	}
	metrics := make([]ProviderQuotaMetric, 0, 8)
	unit := "USD"

	var weeklyUsedPercent *float64
	if pct, ok := cursorUsageFloat(usage, "weekly_used_percent"); ok {
		weeklyUsedPercent = &pct
	}

	if used, ok := cursorUsageFloat(usage, "total_usage"); ok {
		metrics = append(metrics, ProviderQuotaMetric{
			Metric: "total_usage",
			Label:  "Total account usage",
			Nature: NatureCost,
			Used:   &used,
			Unit:   &unit,
		})
	}
	if limit, ok := cursorUsageFloat(usage, "total_credits"); ok {
		if weeklyUsedPercent == nil {
			if used, ok := cursorUsageFloat(usage, "total_usage"); ok && limit > 0 {
				pct := (used / limit) * 100
				weeklyUsedPercent = &pct
			}
		}
		metrics = append(metrics, ProviderQuotaMetric{
			Metric:          "total_credits",
			Label:           "Weekly included budget",
			Nature:          NatureQuota,
			Limit:           &limit,
			Unit:            &unit,
			ConsumedPercent: weeklyUsedPercent,
		})
	}
	if weeklyUsedPercent != nil {
		metrics = append(metrics, ProviderQuotaMetric{
			Metric:          "weekly_used_percent",
			Label:           "Weekly used percent",
			Nature:          NatureUsage,
			ConsumedPercent: weeklyUsedPercent,
		})
	}
	if resetsAt, ok := cursorUsageString(usage, "weekly_resets_at"); ok {
		metrics = append(metrics, ProviderQuotaMetric{
			Metric:   "weekly_resets_at",
			Label:    "Weekly reset",
			Nature:   NatureUsage,
			ResetsAt: &resetsAt,
		})
	}
	if prepaid, ok := cursorUsageFloat(usage, "prepaid_available_credits"); ok {
		metrics = append(metrics, ProviderQuotaMetric{
			Metric:    "prepaid_available_credits",
			Label:     "Prepaid credits available",
			Nature:    NatureQuota,
			Remaining: &prepaid,
			Unit:      &unit,
		})
	}
	if agentTotal, ok := cursorUsageFloat(usage, "analytics_agent_requests_total"); ok {
		metrics = append(metrics, ProviderQuotaMetric{
			Metric: "analytics_agent_requests_total",
			Label:  "Agent requests (period)",
			Nature: NatureUsage,
			Used:   &agentTotal,
		})
	}
	return metrics
}

func cursorUsageFloat(usage map[string]interface{}, key string) (float64, bool) {
	value, ok := usage[key]
	if !ok {
		return 0, false
	}
	pointer := numericPointer(value)
	if pointer == nil {
		return 0, false
	}
	return *pointer, true
}

func cursorUsageString(usage map[string]interface{}, key string) (string, bool) {
	value, ok := usage[key]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	return text, ok && strings.TrimSpace(text) != ""
}

func numericPointer(value interface{}) *float64 {
	switch typed := value.(type) {
	case float64:
		return &typed
	case json.Number:
		if f, err := typed.Float64(); err == nil {
			return &f
		}
	}
	return nil
}
