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
	openCodeDefaultRoot     = "https://opencode.ai"
	openCodeGoUsagePath     = "/zen/go/v1/usage"
	openCodeBudgetsPath     = "/console/api/policies/current-budgets"
	openCodeGoStatusPath    = "/console/api/go/status"
	openCodeQuotaSourceName = "opencode_go_usage_api"
)

// OpenCodeQuotaAdapter queries official OpenCode Go franchise meters and Console budgets.
type OpenCodeQuotaAdapter struct {
	httpClient   *http.Client
	allowedHosts map[string]struct{}
}

// NewOpenCodeQuotaAdapter builds a quota adapter for OpenCode Go accounts.
func NewOpenCodeQuotaAdapter(httpClient *http.Client, allowedHosts []string) *OpenCodeQuotaAdapter {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	allowedMap := make(map[string]struct{}, len(allowedHosts))
	for _, host := range allowedHosts {
		if host = strings.ToLower(strings.TrimSpace(host)); host != "" {
			allowedMap[host] = struct{}{}
		}
	}
	if len(allowedMap) == 0 {
		allowedMap["opencode.ai"] = struct{}{}
	}
	return &OpenCodeQuotaAdapter{httpClient: httpClient, allowedHosts: allowedMap}
}

type openCodeGoUsageEnvelope struct {
	Usage openCodeGoUsageWindows `json:"usage"`
}

type openCodeGoUsageWindows struct {
	Rolling *openCodeGoMeter `json:"rolling"`
	Weekly  *openCodeGoMeter `json:"weekly"`
	Monthly *openCodeGoMeter `json:"monthly"`
}

type openCodeGoMeter struct {
	Status   string   `json:"status"`
	Percent  *float64 `json:"percent"`
	ResetsAt string   `json:"resetsAt"`
}

type openCodeBudget struct {
	ID       string   `json:"id"`
	Limit    *float64 `json:"limit"`
	Spent    *float64 `json:"spent"`
	Scope    string   `json:"scope"`
	Per      string   `json:"per"`
	Coverage string   `json:"coverage"`
	ResetsAt *float64 `json:"resetsAt"`
}

type openCodeGoStatus struct {
	Product string `json:"product"`
}

// FetchQuota reads Go usage meters, Console budgets, and optional product label.
func (a *OpenCodeQuotaAdapter) FetchQuota(ctx context.Context, endpoint, credential string) (ProviderQuotaSnapshot, error) {
	snapshot := newOpenCodeSnapshot()
	if strings.TrimSpace(credential) == "" {
		applyQuotaError(&snapshot, &ProviderQuotaError{
			Code: "unconfigured_credential", Message: "Credencial não configurada",
		})
		return snapshot, nil
	}

	root, err := a.quotaRoot(endpoint)
	if err != nil {
		if errors.Is(err, errQuotaHostNotAllowed) {
			applyQuotaError(&snapshot, &ProviderQuotaError{
				Code: "host_not_allowed", Message: "Host não autorizado pela política de segurança",
			})
			return snapshot, nil
		}
		applyQuotaError(&snapshot, &ProviderQuotaError{
			Code: "invalid_endpoint", Message: "URL de endpoint inválida",
		})
		return snapshot, nil
	}

	usageBody, usageErr := a.request(ctx, root+openCodeGoUsagePath, credential)
	if usageErr != nil {
		applyQuotaError(&snapshot, usageErr)
		return snapshot, nil
	}
	var usage openCodeGoUsageEnvelope
	if err := json.Unmarshal(usageBody, &usage); err != nil {
		applyQuotaError(&snapshot, &ProviderQuotaError{
			Code: "json_parse_error", Message: "Formato de resposta inesperado do provedor",
		})
		return snapshot, nil
	}
	mapOpenCodeGoMeters(&snapshot, usage.Usage)

	budgetBody, budgetErr := a.request(ctx, root+openCodeBudgetsPath, credential)
	if budgetErr == nil {
		var budgets []openCodeBudget
		if err := json.Unmarshal(budgetBody, &budgets); err == nil {
			mapOpenCodeBudgets(&snapshot, budgets)
		}
	}

	if statusBody, statusErr := a.request(ctx, root+openCodeGoStatusPath, credential); statusErr == nil {
		var status openCodeGoStatus
		if err := json.Unmarshal(statusBody, &status); err == nil && strings.TrimSpace(status.Product) != "" {
			product := strings.TrimSpace(status.Product)
			snapshot.WorkspaceName = &product
		}
	}

	snapshot.Status = QuotaStatusAvailable
	return snapshot, nil
}

func (a *OpenCodeQuotaAdapter) request(ctx context.Context, target, credential string) ([]byte, *ProviderQuotaError) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, &ProviderQuotaError{Code: "request_build_failed", Message: "Falha ao preparar consulta ao OpenCode"}
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(credential))
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Connexto-AI-Hub/1.0")

	response, err := a.httpClient.Do(request)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || isProviderTimeout(err) {
			return nil, &ProviderQuotaError{Code: "timeout", Message: "Tempo limite na consulta ao OpenCode excedido"}
		}
		return nil, &ProviderQuotaError{Code: "network_error", Message: "Falha de rede ao consultar OpenCode"}
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, openCodeStatusError(response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, defaultMaxQuotaBytes))
	if err != nil {
		return nil, &ProviderQuotaError{Code: "body_read_error", Message: "Falha ao ler resposta da API"}
	}
	return body, nil
}

func (a *OpenCodeQuotaAdapter) quotaRoot(endpoint string) (string, error) {
	root := openCodeQuotaRoot(endpoint)
	parsed, err := url.Parse(root)
	if err != nil {
		return "", err
	}
	if _, ok := a.allowedHosts[strings.ToLower(parsed.Hostname())]; !ok {
		return "", errQuotaHostNotAllowed
	}
	return strings.TrimRight(root, "/"), nil
}

func openCodeQuotaRoot(endpoint string) string {
	root := strings.TrimSpace(endpoint)
	if root == "" {
		return openCodeDefaultRoot
	}
	root = strings.TrimRight(root, "/")
	for _, suffix := range []string{"/inference/go", "/inference"} {
		if strings.HasSuffix(strings.ToLower(root), suffix) {
			root = root[:len(root)-len(suffix)]
			break
		}
	}
	root = strings.TrimRight(root, "/")
	if root == "" || root == "https:" || root == "http:" {
		return openCodeDefaultRoot
	}
	return root
}

func newOpenCodeSnapshot() ProviderQuotaSnapshot {
	now := time.Now().UTC().Format(time.RFC3339)
	return ProviderQuotaSnapshot{
		ProviderID: TypeOpenCodeGo,
		Scope:      ScopeSubscription,
		Status:     QuotaStatusError,
		Quotas:     []ProviderQuotaMetric{},
		Source: ProviderQuotaSource{
			Type: "provider_api", Name: openCodeQuotaSourceName, Scope: ScopeSubscription, FetchedAt: now,
		},
	}
}

func mapOpenCodeGoMeters(snapshot *ProviderQuotaSnapshot, windows openCodeGoUsageWindows) {
	appendOpenCodeMeter(snapshot, "go_rolling", "Go rolling", "rolling", windows.Rolling)
	appendOpenCodeMeter(snapshot, "go_weekly", "Go weekly", "weekly", windows.Weekly)
	appendOpenCodeMeter(snapshot, "go_monthly", "Go monthly", "monthly", windows.Monthly)
}

func appendOpenCodeMeter(snapshot *ProviderQuotaSnapshot, metric, label, period string, meter *openCodeGoMeter) {
	if meter == nil || meter.Percent == nil {
		return
	}
	periodCopy := period
	entry := ProviderQuotaMetric{
		Metric:          metric,
		Label:           label,
		Nature:          NatureQuota,
		ConsumedPercent: meter.Percent,
		Period:          &periodCopy,
	}
	if meter.ResetsAt != "" {
		resetsAt := meter.ResetsAt
		entry.ResetsAt = &resetsAt
	}
	if meter.Status != "" && meter.Status != "ok" {
		scope := "status:" + meter.Status
		entry.Scope = scope
	}
	snapshot.Quotas = append(snapshot.Quotas, entry)
}

func mapOpenCodeBudgets(snapshot *ProviderQuotaSnapshot, budgets []openCodeBudget) {
	usd := "USD"
	for _, budget := range budgets {
		id := strings.TrimSpace(budget.ID)
		if id == "" {
			continue
		}
		metric := ProviderQuotaMetric{
			Metric: "budget_" + id,
			Label:  "Console budget",
			Nature: NatureCost,
			Used:   budget.Spent,
			Limit:  budget.Limit,
			Unit:   &usd,
		}
		if budget.Scope != "" {
			metric.Scope = budget.Scope
		}
		if budget.Per != "" {
			period := budget.Per
			if budget.Coverage != "" {
				period = budget.Per + "/" + budget.Coverage
			}
			metric.Period = &period
		}
		if budget.ResetsAt != nil {
			seconds := *budget.ResetsAt / 1000
			resetsAt := time.Unix(int64(seconds), 0).UTC().Format(time.RFC3339)
			metric.ResetsAt = &resetsAt
		}
		snapshot.Quotas = append(snapshot.Quotas, metric)
	}
}

func openCodeStatusError(status int) *ProviderQuotaError {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &ProviderQuotaError{Code: "unauthorized", Message: "Credencial inválida ou sem permissão para esta consulta"}
	case http.StatusTooManyRequests:
		return &ProviderQuotaError{Code: "rate_limited", Message: "Limite de requisições excedido na API do OpenCode"}
	default:
		return &ProviderQuotaError{Code: fmt.Sprintf("http_%d", status), Message: fmt.Sprintf("OpenCode retornou status HTTP %d", status)}
	}
}
