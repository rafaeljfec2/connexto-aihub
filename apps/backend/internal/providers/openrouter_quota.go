package providers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultMaxQuotaBytes = 1 << 20

var errQuotaHostNotAllowed = errors.New("quota host not allowed")

// OpenRouterQuotaAdapter queries OpenRouter's official quota endpoints.
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
		if host = strings.ToLower(strings.TrimSpace(host)); host != "" {
			allowedMap[host] = struct{}{}
		}
	}
	if len(allowedMap) == 0 {
		allowedMap["openrouter.ai"] = struct{}{}
	}
	return &OpenRouterQuotaAdapter{httpClient: httpClient, allowedHosts: allowedMap}
}

func (a *OpenRouterQuotaAdapter) request(ctx context.Context, endpoint, path, credential string) ([]byte, *ProviderQuotaError) {
	if strings.TrimSpace(credential) == "" {
		return nil, &ProviderQuotaError{Code: "unconfigured_credential", Message: "Credencial não configurada"}
	}
	target, err := a.target(endpoint, path)
	if err != nil {
		if errors.Is(err, errQuotaHostNotAllowed) {
			return nil, &ProviderQuotaError{Code: "host_not_allowed", Message: "Host não autorizado pela política de segurança"}
		}
		return nil, &ProviderQuotaError{Code: "invalid_endpoint", Message: "URL de endpoint inválida"}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, &ProviderQuotaError{Code: "request_build_failed", Message: "Falha ao preparar consulta ao OpenRouter"}
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(credential))
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Connexto-AI-Hub/1.0")

	response, err := a.httpClient.Do(request)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || isProviderTimeout(err) {
			return nil, &ProviderQuotaError{Code: "timeout", Message: "Tempo limite na consulta ao OpenRouter excedido"}
		}
		return nil, &ProviderQuotaError{Code: "network_error", Message: "Falha de rede ao consultar OpenRouter"}
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, openRouterStatusError(response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, defaultMaxQuotaBytes))
	if err != nil {
		return nil, &ProviderQuotaError{Code: "body_read_error", Message: "Falha ao ler resposta da API"}
	}
	return body, nil
}

func (a *OpenRouterQuotaAdapter) target(endpoint, path string) (string, error) {
	root := strings.TrimRight(endpoint, "/")
	if root == "" {
		root = OpenRouterEndpoint
	}
	root = strings.TrimSuffix(strings.TrimSuffix(root, "/key"), "/credits")
	parsed, err := url.Parse(root + path)
	if err != nil {
		return "", err
	}
	if _, ok := a.allowedHosts[strings.ToLower(parsed.Hostname())]; !ok {
		return "", errQuotaHostNotAllowed
	}
	return parsed.String(), nil
}

func newOpenRouterSnapshot(scope DataScope, sourceName string) ProviderQuotaSnapshot {
	now := time.Now().UTC().Format(time.RFC3339)
	return ProviderQuotaSnapshot{
		ProviderID: TypeOpenRouter,
		Scope:      scope,
		Status:     QuotaStatusError,
		Quotas:     []ProviderQuotaMetric{},
		Source: ProviderQuotaSource{
			Type: "provider_api", Name: sourceName, Scope: scope, FetchedAt: now,
		},
	}
}

func openRouterStatusError(status int) *ProviderQuotaError {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &ProviderQuotaError{Code: "unauthorized", Message: "Credencial inválida ou sem permissão para esta consulta"}
	case http.StatusTooManyRequests:
		return &ProviderQuotaError{Code: "rate_limited", Message: "Limite de requisições excedido na API do OpenRouter"}
	default:
		return &ProviderQuotaError{Code: fmt.Sprintf("http_%d", status), Message: fmt.Sprintf("OpenRouter retornou status HTTP %d", status)}
	}
}

func isProviderTimeout(err error) bool {
	type timeoutError interface{ Timeout() bool }
	var timeout timeoutError
	return errors.As(err, &timeout) && timeout.Timeout()
}

func applyQuotaError(snapshot *ProviderQuotaSnapshot, providerError *ProviderQuotaError) {
	snapshot.Status = QuotaStatusError
	snapshot.Error = providerError
	if providerError == nil {
		return
	}
	switch providerError.Code {
	case "unconfigured_credential":
		snapshot.Status = QuotaStatusUnconfiguredCredential
	case "unauthorized":
		snapshot.Status = QuotaStatusUnauthorized
	case "rate_limited":
		snapshot.Status = QuotaStatusRateLimited
	case "timeout":
		snapshot.Status = QuotaStatusTimeout
	}
}
