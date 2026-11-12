package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	antigravityQuotaSourceName = "antigravity_proxy_usage_api"
	antigravityUsageRoute      = "/providers/antigravity/usage"
	defaultAntigravityTimeout  = 10 * time.Second
)

type quotaAccountContextKey struct{}

func withQuotaAccountID(ctx context.Context, accountID string) context.Context {
	return context.WithValue(ctx, quotaAccountContextKey{}, accountID)
}

func quotaAccountIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(quotaAccountContextKey{}).(string)
	return strings.TrimSpace(id)
}

// AntigravityQuotaAdapter queries the Connexto Proxy for Antigravity quotas.
type AntigravityQuotaAdapter struct {
	httpClient   *http.Client
	allowedHosts map[string]struct{}
}

// NewAntigravityQuotaAdapter creates a new Antigravity quota adapter.
func NewAntigravityQuotaAdapter(httpClient *http.Client, allowedHosts []string) *AntigravityQuotaAdapter {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultAntigravityTimeout}
	}
	allowedMap := make(map[string]struct{}, len(allowedHosts))
	for _, host := range allowedHosts {
		if host = strings.ToLower(strings.TrimSpace(host)); host != "" {
			allowedMap[host] = struct{}{}
		}
	}
	return &AntigravityQuotaAdapter{
		httpClient:   httpClient,
		allowedHosts: allowedMap,
	}
}

type proxyUsageAccount struct {
	ID         *string `json:"id"`
	Email      *string `json:"email"`
	Tier       *string `json:"tier"`
	TierName   *string `json:"tier_name"`
	Project    *string `json:"project"`
	GCPManaged *bool   `json:"gcp_managed"`
}

type proxyUsageMetric struct {
	ID                string   `json:"id"`
	Name              *string  `json:"name"`
	Nature            string   `json:"nature"`
	Used              *float64 `json:"used"`
	Limit             *float64 `json:"limit"`
	Remaining         *float64 `json:"remaining"`
	Percentage        *float64 `json:"percentage"`
	Unit              *string  `json:"unit"`
	Cycle             *string  `json:"cycle"`
	ResetAt           *string  `json:"reset_at"`
	ReportedAt        *string  `json:"reported_at"`
	Scope             *string  `json:"scope"`
	RemainingFraction *float64 `json:"remaining_fraction"`
}

type proxyUsageError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type proxyUsageResponse struct {
	Provider   string             `json:"provider"`
	Status     string             `json:"status"`
	Source     string             `json:"source"`
	Account    *proxyUsageAccount `json:"account"`
	Quotas     []proxyUsageMetric `json:"quotas"`
	ReportedAt *string            `json:"reported_at"`
	FetchedAt  string             `json:"fetched_at"`
	Error      *proxyUsageError   `json:"error"`
}

func newAntigravitySnapshot(scope DataScope) ProviderQuotaSnapshot {
	now := time.Now().UTC().Format(time.RFC3339)
	return ProviderQuotaSnapshot{
		ProviderID:   TypeAntigravity,
		ProviderName: "Google Antigravity",
		Scope:        scope,
		Status:       QuotaStatusError,
		Quotas:       []ProviderQuotaMetric{},
		Source: ProviderQuotaSource{
			Type:      "provider_api",
			Name:      antigravityQuotaSourceName,
			Scope:     scope,
			FetchedAt: now,
		},
	}
}

// FetchQuota queries the Proxy GET /v1/providers/antigravity/usage endpoint.
func (a *AntigravityQuotaAdapter) FetchQuota(ctx context.Context, endpoint, credential string) (ProviderQuotaSnapshot, error) {
	snapshot := newAntigravitySnapshot(ScopeAccount)

	if strings.TrimSpace(credential) == "" {
		snapshot.Status = QuotaStatusUnconfiguredCredential
		snapshot.Error = &ProviderQuotaError{
			Code:    "unconfigured_credential",
			Message: "Gateway API Key não configurada para comunicação com o proxy",
		}
		return snapshot, nil
	}

	base := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if base == "" {
		base = "http://127.0.0.1:9090/v1"
	}
	reqURL := base
	if !strings.HasSuffix(reqURL, "/v1") {
		reqURL = reqURL + "/v1"
	}
	reqURL = reqURL + antigravityUsageRoute

	parsed, err := url.Parse(reqURL)
	if err != nil {
		snapshot.Status = QuotaStatusError
		snapshot.Error = &ProviderQuotaError{
			Code:    "invalid_endpoint",
			Message: fmt.Sprintf("Endpoint do provedor inválido: %s", err.Error()),
		}
		return snapshot, nil
	}

	if len(a.allowedHosts) > 0 {
		hostname := strings.ToLower(parsed.Hostname())
		if _, ok := a.allowedHosts[hostname]; !ok {
			snapshot.Status = QuotaStatusError
			snapshot.Error = &ProviderQuotaError{
				Code:    "host_not_allowed",
				Message: fmt.Sprintf("Host %s não está na lista de hosts permitidos", hostname),
			}
			return snapshot, nil
		}
	}

	accountID := quotaAccountIDFromContext(ctx)
	if accountID != "" {
		q := parsed.Query()
		q.Set("accountId", accountID)
		parsed.RawQuery = q.Encode()
		reqURL = parsed.String()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		snapshot.Status = QuotaStatusError
		snapshot.Error = &ProviderQuotaError{
			Code:    "request_build_failed",
			Message: err.Error(),
		}
		return snapshot, fmt.Errorf("build antigravity quota request: %w", err)
	}

	req.Header.Set("X-Gateway-API-Key", credential)
	req.Header.Set("Accept", "application/json")
	if accountID != "" {
		req.Header.Set("X-Account-ID", accountID)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
			snapshot.Status = QuotaStatusTimeout
			snapshot.Error = &ProviderQuotaError{
				Code:    "timeout",
				Message: "A consulta de quotas do Antigravity ao proxy excedeu o tempo limite",
			}
			return snapshot, nil
		}
		if errors.Is(err, context.Canceled) {
			return snapshot, err
		}
		snapshot.Status = QuotaStatusError
		snapshot.Error = &ProviderQuotaError{
			Code:    "network_error",
			Message: err.Error(),
		}
		return snapshot, nil
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		snapshot.Status = QuotaStatusError
		snapshot.Error = &ProviderQuotaError{
			Code:    "read_error",
			Message: "Falha ao ler corpo da resposta do proxy",
		}
		return snapshot, nil
	}

	var payload proxyUsageResponse
	if decodeErr := json.Unmarshal(body, &payload); decodeErr != nil {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			snapshot.Status = QuotaStatusUnauthorized
			snapshot.Error = &ProviderQuotaError{
				Code:    "unauthorized",
				Message: "Gateway API Key rejeitada pelo proxy",
			}
			return snapshot, nil
		}
		snapshot.Status = QuotaStatusError
		snapshot.Error = &ProviderQuotaError{
			Code:    "upstream_error",
			Message: fmt.Sprintf("Proxy retornou status %d: %s", resp.StatusCode, strings.TrimSpace(string(body))),
		}
		return snapshot, nil
	}

	if payload.FetchedAt != "" {
		snapshot.Source.FetchedAt = payload.FetchedAt
	}
	if payload.ReportedAt != nil {
		snapshot.Source.ProviderReportedAt = payload.ReportedAt
	}

	if payload.Account != nil {
		if payload.Account.Email != nil && strings.TrimSpace(*payload.Account.Email) != "" {
			snapshot.AccountLabel = strings.TrimSpace(*payload.Account.Email)
		}
		if payload.Account.TierName != nil && strings.TrimSpace(*payload.Account.TierName) != "" {
			snapshot.WorkspaceName = payload.Account.TierName
		} else if payload.Account.Tier != nil && strings.TrimSpace(*payload.Account.Tier) != "" {
			snapshot.WorkspaceName = payload.Account.Tier
		}
	}

	switch payload.Status {
	case "available":
		snapshot.Status = QuotaStatusAvailable
	case "unauthenticated":
		snapshot.Status = QuotaStatusUnauthorized
		msg := "Sessão local Antigravity não autenticada no Proxy"
		if payload.Error != nil && payload.Error.Message != "" {
			msg = payload.Error.Message
		}
		snapshot.Error = &ProviderQuotaError{
			Code:    "unauthenticated",
			Message: msg,
		}
		return snapshot, nil
	case "unsupported":
		snapshot.Status = QuotaStatusUnsupported
		msg := "Consulta de usage do Antigravity não suportada pelo Proxy"
		if payload.Error != nil && payload.Error.Message != "" {
			msg = payload.Error.Message
		}
		snapshot.Error = &ProviderQuotaError{
			Code:    "unsupported",
			Message: msg,
		}
		return snapshot, nil
	case "error":
		snapshot.Status = QuotaStatusError
		code := "upstream_error"
		msg := "Falha ao obter usage do Antigravity no Proxy"
		if payload.Error != nil {
			if payload.Error.Code != "" {
				code = payload.Error.Code
			}
			if payload.Error.Message != "" {
				msg = payload.Error.Message
			}
		}
		snapshot.Error = &ProviderQuotaError{
			Code:    code,
			Message: msg,
		}
		return snapshot, nil
	default:
		if resp.StatusCode != http.StatusOK {
			snapshot.Status = QuotaStatusError
			snapshot.Error = &ProviderQuotaError{
				Code:    "upstream_error",
				Message: fmt.Sprintf("Proxy retornou status HTTP %d com status payload '%s'", resp.StatusCode, payload.Status),
			}
			return snapshot, nil
		}
		snapshot.Status = QuotaStatusAvailable
	}

	quotas := make([]ProviderQuotaMetric, 0, len(payload.Quotas))
	for _, q := range payload.Quotas {
		metric := ProviderQuotaMetric{
			Metric: q.ID,
			Label:  q.ID,
			Nature: NatureRateLimit,
			Scope:  "model",
		}
		if q.Name != nil && strings.TrimSpace(*q.Name) != "" {
			metric.Label = *q.Name
		}
		if q.Nature != "" {
			metric.Nature = MetricNature(q.Nature)
		}
		if q.Scope != nil && *q.Scope != "" {
			metric.Scope = *q.Scope
		}
		if q.ResetAt != nil && *q.ResetAt != "" {
			metric.ResetsAt = q.ResetAt
		}
		if q.Unit != nil && *q.Unit != "" {
			metric.Unit = q.Unit
		}
		if q.Cycle != nil && *q.Cycle != "" {
			metric.Period = q.Cycle
		}

		if q.RemainingFraction != nil {
			frac := *q.RemainingFraction
			metric.RemainingFraction = &frac
			remPct := math.Round(frac*10000.0) / 100.0
			metric.RemainingPercent = &remPct
			consumedPct := math.Round((1.0-frac)*10000.0) / 100.0
			metric.ConsumedPercent = &consumedPct
		}

		quotas = append(quotas, metric)
	}
	snapshot.Quotas = quotas

	return snapshot, nil
}
