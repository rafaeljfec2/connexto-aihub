package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	openRouterKeyPath     = "/key"
	openRouterCreditsPath = "/credits"
	defaultMaxQuotaBytes  = 1 << 20 // 1 MB
)

// OpenRouterQuotaAdapter queries OpenRouter's official GET /api/v1/key endpoint.
type OpenRouterQuotaAdapter struct {
	httpClient   *http.Client
	allowedHosts map[string]struct{}
}

// NewOpenRouterQuotaAdapter builds a quota adapter for OpenRouter accounts.
func NewOpenRouterQuotaAdapter(httpClient *http.Client, allowedHosts []string) *OpenRouterQuotaAdapter {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	allowedMap := make(map[string]struct{}, len(allowedHosts))
	for _, host := range allowedHosts {
		h := strings.ToLower(strings.TrimSpace(host))
		if h != "" {
			allowedMap[h] = struct{}{}
		}
	}
	// Always allow openrouter.ai by default if empty
	if len(allowedMap) == 0 {
		allowedMap["openrouter.ai"] = struct{}{}
	}

	return &OpenRouterQuotaAdapter{
		httpClient:   httpClient,
		allowedHosts: allowedMap,
	}
}

type openRouterKeyEnvelope struct {
	Data openRouterKeyData `json:"data"`
}

type openRouterKeyData struct {
	Label            string   `json:"label"`
	Usage            *float64 `json:"usage"`
	Limit            *float64 `json:"limit"`
	LimitRemaining   *float64 `json:"limit_remaining"`
	LimitReset       *string  `json:"limit_reset"`
	UsageDaily       *float64 `json:"usage_daily"`
	UsageWeekly      *float64 `json:"usage_weekly"`
	UsageMonthly     *float64 `json:"usage_monthly"`
	ByokUsage        *float64 `json:"byok_usage"`
	ByokUsageDaily   *float64 `json:"byok_usage_daily"`
	ByokUsageWeekly  *float64 `json:"byok_usage_weekly"`
	ByokUsageMonthly *float64 `json:"byok_usage_monthly"`
	IsFreeTier       *bool    `json:"is_free_tier"`
}

type openRouterCreditsEnvelope struct {
	Data openRouterCreditsData `json:"data"`
}

type openRouterCreditsData struct {
	TotalCredits *float64 `json:"total_credits"`
	TotalUsage   *float64 `json:"total_usage"`
}

// FetchQuota executes GET /api/v1/key against OpenRouter.
func (a *OpenRouterQuotaAdapter) FetchQuota(ctx context.Context, endpoint, credential string) (ProviderQuotaSnapshot, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	snapshot := ProviderQuotaSnapshot{
		ProviderID: TypeOpenRouter,
		Scope:      ScopeAPIKey,
		Status:     QuotaStatusError,
		Quotas:     make([]ProviderQuotaMetric, 0, 11),
		Source: ProviderQuotaSource{
			Type:      "provider_api",
			Name:      "openrouter_key_api",
			Scope:     ScopeAPIKey,
			FetchedAt: now,
		},
	}

	trimmedCred := strings.TrimSpace(credential)
	if trimmedCred == "" {
		snapshot.Status = QuotaStatusUnconfiguredCredential
		snapshot.Error = &ProviderQuotaError{
			Code:    "unconfigured_credential",
			Message: "Conta sem credencial configurada",
		}
		return snapshot, nil
	}

	targetURL := strings.TrimRight(endpoint, "/")
	if targetURL == "" {
		targetURL = OpenRouterEndpoint
	}
	if !strings.HasSuffix(targetURL, openRouterKeyPath) {
		targetURL = targetURL + openRouterKeyPath
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		snapshot.Status = QuotaStatusError
		snapshot.Error = &ProviderQuotaError{
			Code:    "invalid_endpoint",
			Message: fmt.Sprintf("URL de endpoint inválida: %v", err),
		}
		return snapshot, nil
	}

	// Strictly validate destination host
	host := strings.ToLower(parsedURL.Hostname())
	if _, ok := a.allowedHosts[host]; !ok {
		snapshot.Status = QuotaStatusError
		snapshot.Error = &ProviderQuotaError{
			Code:    "host_not_allowed",
			Message: fmt.Sprintf("Host '%s' não autorizado pela política de segurança", host),
		}
		return snapshot, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		snapshot.Status = QuotaStatusError
		snapshot.Error = &ProviderQuotaError{
			Code:    "request_build_failed",
			Message: err.Error(),
		}
		return snapshot, nil
	}

	req.Header.Set("Authorization", "Bearer "+trimmedCred)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Connexto-AI-Hub/1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || osIsTimeout(err) {
			snapshot.Status = QuotaStatusTimeout
			snapshot.Error = &ProviderQuotaError{
				Code:    "timeout",
				Message: "Tempo limite na consulta ao OpenRouter excedido",
			}
			return snapshot, nil
		}
		snapshot.Status = QuotaStatusError
		snapshot.Error = &ProviderQuotaError{
			Code:    "network_error",
			Message: err.Error(),
		}
		return snapshot, nil
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, defaultMaxQuotaBytes))
	if err != nil {
		snapshot.Status = QuotaStatusError
		snapshot.Error = &ProviderQuotaError{
			Code:    "body_read_error",
			Message: "Falha ao ler resposta da API",
		}
		return snapshot, nil
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// Process 200 OK below
	case http.StatusUnauthorized, http.StatusForbidden:
		snapshot.Status = QuotaStatusUnauthorized
		snapshot.Error = &ProviderQuotaError{
			Code:    "unauthorized",
			Message: "Credencial inválida ou sem permissão para consultar chave",
		}
		return snapshot, nil
	case http.StatusTooManyRequests:
		snapshot.Status = QuotaStatusRateLimited
		snapshot.Error = &ProviderQuotaError{
			Code:    "rate_limited",
			Message: "Limite de requisições excedido na API do OpenRouter",
		}
		return snapshot, nil
	default:
		snapshot.Status = QuotaStatusError
		snapshot.Error = &ProviderQuotaError{
			Code:    fmt.Sprintf("http_%d", resp.StatusCode),
			Message: fmt.Sprintf("OpenRouter retornou status HTTP %d", resp.StatusCode),
		}
		return snapshot, nil
	}

	var env openRouterKeyEnvelope
	if err := json.Unmarshal(bodyBytes, &env); err != nil {
		snapshot.Status = QuotaStatusError
		snapshot.Error = &ProviderQuotaError{
			Code:    "json_parse_error",
			Message: "Formato de resposta inesperado do provedor",
		}
		return snapshot, nil
	}

	d := env.Data
	usdUnit := "USD"

	// 1. limit
	if d.Limit != nil {
		m := ProviderQuotaMetric{
			Metric: "limit",
			Label:  "Limite da chave",
			Nature: NatureQuota,
			Limit:  d.Limit,
			Unit:   &usdUnit,
			Period: d.LimitReset,
		}
		snapshot.Quotas = append(snapshot.Quotas, m)
	}

	// 2. limit_remaining
	if d.LimitRemaining != nil {
		m := ProviderQuotaMetric{
			Metric:    "limit_remaining",
			Label:     "Saldo da chave",
			Nature:    NatureQuota,
			Remaining: d.LimitRemaining,
			Unit:      &usdUnit,
			Period:    d.LimitReset,
		}
		snapshot.Quotas = append(snapshot.Quotas, m)
	}

	// 3. usage (total cost)
	if d.Usage != nil {
		m := ProviderQuotaMetric{
			Metric: "usage",
			Label:  "Consumo total",
			Nature: NatureCost,
			Used:   d.Usage,
			Unit:   &usdUnit,
		}
		snapshot.Quotas = append(snapshot.Quotas, m)
	}

	// 4. usage_daily
	if d.UsageDaily != nil {
		period := "daily"
		m := ProviderQuotaMetric{
			Metric: "usage_daily",
			Label:  "Consumo diário",
			Nature: NatureCost,
			Used:   d.UsageDaily,
			Unit:   &usdUnit,
			Period: &period,
		}
		snapshot.Quotas = append(snapshot.Quotas, m)
	}

	// 5. usage_weekly
	if d.UsageWeekly != nil {
		period := "weekly"
		m := ProviderQuotaMetric{
			Metric: "usage_weekly",
			Label:  "Consumo semanal",
			Nature: NatureCost,
			Used:   d.UsageWeekly,
			Unit:   &usdUnit,
			Period: &period,
		}
		snapshot.Quotas = append(snapshot.Quotas, m)
	}

	// 6. usage_monthly
	if d.UsageMonthly != nil {
		period := "monthly"
		m := ProviderQuotaMetric{
			Metric: "usage_monthly",
			Label:  "Consumo mensal",
			Nature: NatureCost,
			Used:   d.UsageMonthly,
			Unit:   &usdUnit,
			Period: &period,
		}
		snapshot.Quotas = append(snapshot.Quotas, m)
	}

	// 7. byok_usage
	if d.ByokUsage != nil {
		m := ProviderQuotaMetric{
			Metric: "byok_usage",
			Label:  "Consumo BYOK total",
			Nature: NatureCost,
			Used:   d.ByokUsage,
			Unit:   &usdUnit,
		}
		snapshot.Quotas = append(snapshot.Quotas, m)
	}

	// 8. byok_usage_daily
	if d.ByokUsageDaily != nil {
		period := "daily"
		m := ProviderQuotaMetric{
			Metric: "byok_usage_daily",
			Label:  "Consumo BYOK diário",
			Nature: NatureCost,
			Used:   d.ByokUsageDaily,
			Unit:   &usdUnit,
			Period: &period,
		}
		snapshot.Quotas = append(snapshot.Quotas, m)
	}

	// 9. byok_usage_weekly
	if d.ByokUsageWeekly != nil {
		period := "weekly"
		m := ProviderQuotaMetric{
			Metric: "byok_usage_weekly",
			Label:  "Consumo BYOK semanal",
			Nature: NatureCost,
			Used:   d.ByokUsageWeekly,
			Unit:   &usdUnit,
			Period: &period,
		}
		snapshot.Quotas = append(snapshot.Quotas, m)
	}

	// 10. byok_usage_monthly
	if d.ByokUsageMonthly != nil {
		period := "monthly"
		m := ProviderQuotaMetric{
			Metric: "byok_usage_monthly",
			Label:  "Consumo BYOK mensal",
			Nature: NatureCost,
			Used:   d.ByokUsageMonthly,
			Unit:   &usdUnit,
			Period: &period,
		}
		snapshot.Quotas = append(snapshot.Quotas, m)
	}

	// Optionally query /api/v1/credits to enrich with account-level total credits and usage
	creditsURL := strings.Replace(targetURL, openRouterKeyPath, openRouterCreditsPath, 1)
	if credsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, creditsURL, nil); err == nil {
		credsReq.Header.Set("Authorization", "Bearer "+trimmedCred)
		credsReq.Header.Set("Accept", "application/json")
		credsReq.Header.Set("User-Agent", "Connexto-AI-Hub/1.0")

		if credsResp, err := a.httpClient.Do(credsReq); err == nil {
			defer credsResp.Body.Close()
			if credsResp.StatusCode == http.StatusOK {
				if credsBody, err := io.ReadAll(io.LimitReader(credsResp.Body, defaultMaxQuotaBytes)); err == nil {
					var credsEnv openRouterCreditsEnvelope
					if err := json.Unmarshal(credsBody, &credsEnv); err == nil {
						if credsEnv.Data.TotalCredits != nil {
							snapshot.Quotas = append(snapshot.Quotas, ProviderQuotaMetric{
								Metric: "total_credits",
								Label:  "Créditos da conta",
								Nature: NatureQuota,
								Limit:  credsEnv.Data.TotalCredits,
								Unit:   &usdUnit,
							})
						}
						if credsEnv.Data.TotalUsage != nil {
							snapshot.Quotas = append(snapshot.Quotas, ProviderQuotaMetric{
								Metric: "total_usage",
								Label:  "Consumo total da conta",
								Nature: NatureCost,
								Used:   credsEnv.Data.TotalUsage,
								Unit:   &usdUnit,
							})
						}
					}
				}
			}
		}
	}

	snapshot.Status = QuotaStatusAvailable
	return snapshot, nil
}

func osIsTimeout(err error) bool {
	if err == nil {
		return false
	}
	type timeoutErr interface {
		Timeout() bool
	}
	var te timeoutErr
	if errors.As(err, &te) {
		return te.Timeout()
	}
	return false
}

