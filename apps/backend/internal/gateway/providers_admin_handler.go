package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/operational"
	"aihub/internal/providers"
)

const (
	providerNotFoundType         = "provider_not_found"
	providerConflictType         = "provider_conflict"
	credentialMissingType        = "credential_missing"
	gatewayCredentialMissingType = "gateway_credential_missing"
	providerUnreachableType      = "provider_unreachable"

	providerRequestMaxBytes              = 16 << 10
	providerNotFoundMsg                  = "provider configuration not found"
	providerConflictMsg                  = "provider configuration already exists"
	invalidRequestMsg                    = "request body is not a valid provider payload"
	credentialMissingMsg                 = "provider has no credential configured"
	gatewayCredentialMissingMsg          = "provider has no gateway API key configured"
	providerUnreachableMsg               = "the provider rejected the connectivity check or is unreachable"
	adminInternalMsg                     = "the provider administration request failed"
	proxySessionClearFailedType          = "proxy_session_clear_failed"
	proxySessionClearFailedMsg           = "the connexto proxy rejected clearing runtime sessions"
	proxyAntigravityAuthFailedType       = "proxy_antigravity_auth_failed"
	proxyAntigravityAuthFailedMsg        = "the connexto proxy rejected the Antigravity OAuth request"
	proxyCursorAuthFailedType            = "proxy_cursor_auth_failed"
	proxyCursorAuthFailedMsg             = "the connexto proxy rejected the Cursor usage auth request"
	proxyCursorAuthBrowserUnavailableMsg = "Não foi possível abrir o Chromium no proxy. Confira CHROMIUM_EXECUTABLE_PATH, DISPLAY=:0 (WSL) no .env do proxy e reinicie pnpm dev; feche janelas Chrome do login anteriores."
)

// ProviderAdmin serves the administrative provider endpoints. Responses carry
// the configuration only; the credential is never returned, logged or echoed.
type ProviderAdmin struct {
	service        *providers.Service
	logger         *slog.Logger
	eventHub       *operational.Hub
	statuses       func() []providers.Status
	quotaService   *providers.QuotaService
	usageAnalytics *providers.CursorUsageAnalyticsService
	catalogManager *catalog.Manager
}

// NewProviderAdmin builds the administrative handlers.
func NewProviderAdmin(service *providers.Service, logger *slog.Logger) *ProviderAdmin {
	return &ProviderAdmin{service: service, logger: logger}
}

// BindCatalogManager attaches the catalog manager for model synchronization.
func (a *ProviderAdmin) BindCatalogManager(manager *catalog.Manager) {
	a.catalogManager = manager
}

// Available handles GET /api/providers/available.
func (a *ProviderAdmin) Available() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		available := a.service.AvailableProviders()
		writeJSON(w, http.StatusOK, map[string]any{"providers": available})
	})
}

// BindOperationalEvents connects provider administration actions to the operational SSE hub.
func (a *ProviderAdmin) BindOperationalEvents(hub *operational.Hub, statuses func() []providers.Status) {
	a.eventHub = hub
	a.statuses = statuses
}

// BindQuotaService attaches the quota query service to provider administration.
func (a *ProviderAdmin) BindQuotaService(quotaService *providers.QuotaService) {
	a.quotaService = quotaService
}

// BindCursorUsageAnalytics attaches the cursor usage analytics collection service.
func (a *ProviderAdmin) BindCursorUsageAnalytics(service *providers.CursorUsageAnalyticsService) {
	a.usageAnalytics = service
}

// Quotas handles GET /api/ui/providers/quotas.
func (a *ProviderAdmin) Quotas() http.Handler {
	return NewProviderQuotasHandler(a.quotaService, a.logger)
}

// CursorUsageAnalytics handles GET /api/ui/providers/cursor/usage-analytics.
func (a *ProviderAdmin) CursorUsageAnalytics() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		if a.usageAnalytics == nil {
			writeJSONError(w, http.StatusInternalServerError, "usage_analytics_unavailable", "cursor usage analytics service not configured")
			return
		}
		forceRefresh := r.URL.Query().Get("refresh") == "true"
		envelope, err := a.usageAnalytics.Get(r.Context(), r.URL.Query().Get("providerId"), forceRefresh)
		if err != nil {
			a.writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, envelope)
	})
}

func (a *ProviderAdmin) notifyProviderStatus() {
	PublishProviderStatus(a.eventHub, a.statuses)
}

// Create handles POST /api/providers.
func (a *ProviderAdmin) Create() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request createProviderRequest
		if !decodeProviderBody(w, r, &request) {
			return
		}

		enabled := true
		if request.Enabled != nil {
			enabled = *request.Enabled
		}

		id := request.ID
		if id == "" {
			id = request.ProviderID
		}

		record, err := a.service.Create(r.Context(), providers.CreateInput{
			ID:                   id,
			TemplateID:           request.TemplateID,
			Name:                 request.Name,
			Type:                 request.Type,
			Endpoint:             request.Endpoint,
			Credential:           request.Credential,
			ManagementCredential: request.ManagementCredential,
			Enabled:              enabled,
		})
		if err != nil {
			a.writeError(w, err)

			return
		}

		writeJSON(w, http.StatusCreated, toProviderResponse(record))
	})
}

// List handles GET /api/providers.
func (a *ProviderAdmin) List() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		records, err := a.service.List(r.Context())
		if err != nil {
			a.writeError(w, err)

			return
		}

		responses := make([]providerResponse, 0, len(records))
		for _, record := range records {
			responses = append(responses, toProviderResponse(record))
		}

		writeJSON(w, http.StatusOK, map[string]any{"providers": responses})
	})
}

// Get handles GET /api/providers/{id}.
func (a *ProviderAdmin) Get() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record, err := a.service.Get(r.Context(), r.PathValue("id"))
		if err != nil {
			a.writeError(w, err)

			return
		}

		writeJSON(w, http.StatusOK, toProviderResponse(record))
	})
}

// Update handles PATCH /api/providers/{id}.
func (a *ProviderAdmin) Update() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request updateProviderRequest
		if !decodeProviderBody(w, r, &request) {
			return
		}

		record, err := a.service.Update(r.Context(), r.PathValue("id"), providers.UpdateInput{
			Name:                 request.Name,
			Endpoint:             request.Endpoint,
			Credential:           request.Credential,
			ManagementCredential: request.ManagementCredential,
		})
		if err != nil {
			a.writeError(w, err)

			return
		}

		writeJSON(w, http.StatusOK, toProviderResponse(record))
	})
}

// Validate handles POST /api/providers/{id}/validate. It reports probe reachability and updates runtime health.
func (a *ProviderAdmin) Validate() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var err error

		var validateReq validateProviderRequest
		bodyBytes, readErr := io.ReadAll(http.MaxBytesReader(w, r.Body, providerRequestMaxBytes))
		if readErr == nil && len(bytes.TrimSpace(bodyBytes)) > 0 {
			if decErr := json.Unmarshal(bodyBytes, &validateReq); decErr == nil {
				switch {
				case validateReq.ManagementCredential != "":
					err = a.service.ValidateManagementCredential(r.Context(), id, validateReq.ManagementCredential)
				case validateReq.Credential != "":
					err = a.service.ValidateCredential(r.Context(), id, validateReq.Credential)
				default:
					err = a.service.Validate(r.Context(), id)
				}
			} else {
				err = a.service.Validate(r.Context(), id)
			}
		} else {
			err = a.service.Validate(r.Context(), id)
		}

		if err == nil {
			a.notifyProviderStatus()
			writeJSON(w, http.StatusOK, map[string]string{"status": "reachable"})

			return
		}

		switch {
		case errors.Is(err, providers.ErrNoCredential):
			writeJSONError(w, http.StatusConflict, credentialMissingType, credentialMissingMsg)
		case errors.Is(err, providers.ErrNoGatewayKey):
			writeJSONError(w, http.StatusConflict, gatewayCredentialMissingType, gatewayCredentialMissingMsg)
		case errors.Is(err, providers.ErrNotFound), errors.Is(err, providers.ErrCredentialUnreadable):
			a.writeError(w, err)
		default:
			a.logger.Warn("provider validation failed", "provider", id)
			writeJSONError(w, http.StatusBadGateway, providerUnreachableType, providerUnreachableMsg)
		}
	})
}

// ClearAntigravityProxyRuntimeSessions handles POST /api/providers/{id}/proxy/runtime/sessions/clear.
// It forwards to the Connexto Proxy POST /v1/runtime/sessions/clear with namespace antigravity.
func (a *ProviderAdmin) ClearAntigravityProxyRuntimeSessions() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		id := r.PathValue("id")
		result, err := a.service.ClearAntigravityProxyRuntimeSessions(r.Context(), id)
		if err == nil {
			payload := map[string]any{"cleared": true, "namespace": "antigravity"}
			if result != nil {
				if result.MappedAcpIDsCount != nil {
					payload["mapped_acp_ids_count"] = *result.MappedAcpIDsCount
				}
				if result.ListedAcpSessionsCount != nil {
					payload["listed_acp_sessions_count"] = *result.ListedAcpSessionsCount
				}
				if result.AcpSessionsClosed != nil {
					payload["acp_sessions_closed"] = *result.AcpSessionsClosed
				}
				if result.AcpCloseFailures != nil {
					payload["acp_close_failures"] = *result.AcpCloseFailures
				}
				if len(result.Warnings) > 0 {
					payload["warnings"] = result.Warnings
				}
			}
			writeJSON(w, http.StatusOK, payload)
			return
		}
		switch {
		case errors.Is(err, providers.ErrNotFound):
			a.writeError(w, err)
		case errors.Is(err, providers.ErrInvalid):
			writeJSONError(w, http.StatusBadRequest, invalidRequestType, err.Error())
		case errors.Is(err, providers.ErrNoGatewayKey):
			writeJSONError(w, http.StatusConflict, gatewayCredentialMissingType, gatewayCredentialMissingMsg)
		case errors.Is(err, providers.ErrProxySessionClearFailed):
			a.logger.Warn("proxy runtime session clear failed", "provider", id, "error", err)
			writeJSONError(w, http.StatusBadGateway, proxySessionClearFailedType, proxySessionClearFailedMsg)
		default:
			a.logger.Error("proxy runtime session clear", "provider", id, "error", err.Error())
			writeJSONError(w, http.StatusInternalServerError, internalErrorType, adminInternalMsg)
		}
	})
}

// ClearCursorUsageProxyRuntimeSessions handles POST /api/providers/{id}/proxy/cursor/runtime/sessions/clear.
func (a *ProviderAdmin) ClearCursorUsageProxyRuntimeSessions() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		id := r.PathValue("id")
		result, err := a.service.ClearCursorUsageProxyRuntimeSessions(r.Context(), id)
		if err == nil {
			payload := map[string]any{"cleared": true, "namespace": "cursor-usage"}
			if result != nil && result.Namespace != "" {
				payload["namespace"] = result.Namespace
			}
			writeJSON(w, http.StatusOK, payload)
			return
		}
		switch {
		case errors.Is(err, providers.ErrNotFound):
			a.writeError(w, err)
		case errors.Is(err, providers.ErrInvalid):
			writeJSONError(w, http.StatusBadRequest, invalidRequestType, err.Error())
		case errors.Is(err, providers.ErrNoGatewayKey):
			writeJSONError(w, http.StatusConflict, gatewayCredentialMissingType, gatewayCredentialMissingMsg)
		case errors.Is(err, providers.ErrProxySessionClearFailed):
			a.logger.Warn("proxy cursor-usage session clear failed", "provider", id, "error", err)
			writeJSONError(w, http.StatusBadGateway, proxySessionClearFailedType, proxySessionClearFailedMsg)
		default:
			a.logger.Error("proxy cursor-usage session clear", "provider", id, "error", err.Error())
			writeJSONError(w, http.StatusInternalServerError, internalErrorType, adminInternalMsg)
		}
	})
}

// CursorProxyUsageCollect handles POST /api/providers/{id}/proxy/cursor/usage/collect.
func (a *ProviderAdmin) CursorProxyUsageCollect() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		id := r.PathValue("id")
		payload, err := a.service.PostCursorProxyUsageCollect(r.Context(), id)
		if err != nil {
			switch {
			case errors.Is(err, providers.ErrNotFound):
				a.writeError(w, err)
			case errors.Is(err, providers.ErrInvalid):
				writeJSONError(w, http.StatusBadRequest, invalidRequestType, err.Error())
			case errors.Is(err, providers.ErrNoGatewayKey):
				writeJSONError(w, http.StatusConflict, gatewayCredentialMissingType, gatewayCredentialMissingMsg)
			case errors.Is(err, providers.ErrProxyCursorUsageFailed):
				a.logger.Warn("proxy cursor usage collect failed", "provider", id, "error", err)
				writeJSONError(w, http.StatusBadGateway, proxyCursorAuthFailedType, proxyCursorAuthFailureMessage(err))
			default:
				a.logger.Error("proxy cursor usage collect", "provider", id, "error", err.Error())
				writeJSONError(w, http.StatusInternalServerError, internalErrorType, adminInternalMsg)
			}
			return
		}
		writeJSON(w, http.StatusOK, payload)
	})
}

// CursorProxyUsageSession handles GET /api/providers/{id}/proxy/cursor/usage/session.
func (a *ProviderAdmin) CursorProxyUsageSession() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		id := r.PathValue("id")
		payload, err := a.service.GetCursorProxyUsageSession(r.Context(), id)
		if err != nil {
			switch {
			case errors.Is(err, providers.ErrNotFound):
				a.writeError(w, err)
			case errors.Is(err, providers.ErrInvalid):
				writeJSONError(w, http.StatusBadRequest, invalidRequestType, err.Error())
			case errors.Is(err, providers.ErrNoGatewayKey):
				writeJSONError(w, http.StatusConflict, gatewayCredentialMissingType, gatewayCredentialMissingMsg)
			case errors.Is(err, providers.ErrProxyCursorUsageFailed):
				a.logger.Warn("proxy cursor usage session lookup failed", "provider", id, "error", err)
				writeJSONError(w, http.StatusBadGateway, proxyCursorAuthFailedType, proxyCursorAuthFailureMessage(err))
			default:
				a.logger.Error("proxy cursor usage session", "provider", id, "error", err.Error())
				writeJSONError(w, http.StatusInternalServerError, internalErrorType, adminInternalMsg)
			}
			return
		}
		writeJSON(w, http.StatusOK, payload)
	})
}

// CursorProxyAuthStatus handles GET /api/providers/{id}/proxy/cursor/auth/{sessionId}.
func (a *ProviderAdmin) CursorProxyAuthStatus() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		id := r.PathValue("id")
		sessionID := r.PathValue("sessionId")
		payload, err := a.service.GetCursorProxyAuthStatus(r.Context(), id, sessionID)
		if err != nil {
			switch {
			case errors.Is(err, providers.ErrNotFound):
				a.writeError(w, err)
			case errors.Is(err, providers.ErrInvalid):
				writeJSONError(w, http.StatusBadRequest, invalidRequestType, err.Error())
			case errors.Is(err, providers.ErrNoGatewayKey):
				writeJSONError(w, http.StatusConflict, gatewayCredentialMissingType, gatewayCredentialMissingMsg)
			case errors.Is(err, providers.ErrProxyCursorUsageFailed):
				a.logger.Warn("proxy cursor auth status failed", "provider", id, "error", err)
				writeJSONError(w, http.StatusBadGateway, proxyCursorAuthFailedType, proxyCursorAuthFailureMessage(err))
			default:
				a.logger.Error("proxy cursor auth status", "provider", id, "error", err.Error())
				writeJSONError(w, http.StatusInternalServerError, internalErrorType, adminInternalMsg)
			}
			return
		}
		writeJSON(w, http.StatusOK, payload)
	})
}

// CursorProxyAuthStart handles POST /api/providers/{id}/proxy/cursor/auth/start.
func (a *ProviderAdmin) CursorProxyAuthStart() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		id := r.PathValue("id")
		payload, err := a.service.StartCursorProxyAuth(r.Context(), id)
		if err != nil {
			switch {
			case errors.Is(err, providers.ErrNotFound):
				a.writeError(w, err)
			case errors.Is(err, providers.ErrInvalid):
				writeJSONError(w, http.StatusBadRequest, invalidRequestType, err.Error())
			case errors.Is(err, providers.ErrNoGatewayKey):
				writeJSONError(w, http.StatusConflict, gatewayCredentialMissingType, gatewayCredentialMissingMsg)
			case errors.Is(err, providers.ErrProxyCursorUsageFailed):
				a.logger.Warn("proxy cursor auth start failed", "provider", id, "error", err)
				writeJSONError(w, http.StatusBadGateway, proxyCursorAuthFailedType, proxyCursorAuthFailureMessage(err))
			default:
				a.logger.Error("proxy cursor auth start", "provider", id, "error", err.Error())
				writeJSONError(w, http.StatusInternalServerError, internalErrorType, adminInternalMsg)
			}
			return
		}
		writeJSON(w, http.StatusOK, payload)
	})
}

// AntigravityProxyAuthStatus handles GET /api/providers/{id}/proxy/antigravity/auth/status.
// It forwards to the configured proxy base URL + /providers/antigravity/auth/status (base typically ends with /v1).
func (a *ProviderAdmin) AntigravityProxyAuthStatus() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		id := r.PathValue("id")
		payload, err := a.service.GetAntigravityProxyAuthStatus(r.Context(), id)
		if err != nil {
			switch {
			case errors.Is(err, providers.ErrNotFound):
				a.writeError(w, err)
			case errors.Is(err, providers.ErrInvalid):
				writeJSONError(w, http.StatusBadRequest, invalidRequestType, err.Error())
			case errors.Is(err, providers.ErrNoGatewayKey):
				writeJSONError(w, http.StatusConflict, gatewayCredentialMissingType, gatewayCredentialMissingMsg)
			case errors.Is(err, providers.ErrProxyAntigravityAuthFailed):
				a.logger.Warn("proxy antigravity auth status failed", "provider", id, "error", err)
				writeJSONError(w, http.StatusBadGateway, proxyAntigravityAuthFailedType, proxyAntigravityAuthFailureMessage(err))
			default:
				a.logger.Error("proxy antigravity auth status", "provider", id, "error", err.Error())
				writeJSONError(w, http.StatusInternalServerError, internalErrorType, adminInternalMsg)
			}
			return
		}
		writeJSON(w, http.StatusOK, payload)
	})
}

// AntigravityProxyAuthStart handles POST /api/providers/{id}/proxy/antigravity/auth/start.
// It forwards to the configured proxy base URL + /providers/antigravity/auth/start (base typically ends with /v1).
func (a *ProviderAdmin) AntigravityProxyAuthStart() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		id := r.PathValue("id")
		var body antigravityProxyAuthStartRequest
		rawBody, readErr := io.ReadAll(http.MaxBytesReader(w, r.Body, providerRequestMaxBytes))
		if readErr != nil {
			writeJSONError(w, http.StatusBadRequest, invalidRequestType, invalidRequestMsg)
			return
		}
		if len(bytes.TrimSpace(rawBody)) > 0 {
			if err := json.Unmarshal(rawBody, &body); err != nil {
				writeJSONError(w, http.StatusBadRequest, invalidRequestType, invalidRequestMsg)
				return
			}
		}
		var proxyBody *providers.AntigravityProxyAuthStartBody
		if body.RedirectURI != nil {
			proxyBody = &providers.AntigravityProxyAuthStartBody{RedirectURI: body.RedirectURI}
		}
		payload, err := a.service.StartAntigravityProxyAuth(r.Context(), id, proxyBody)
		if err != nil {
			switch {
			case errors.Is(err, providers.ErrNotFound):
				a.writeError(w, err)
			case errors.Is(err, providers.ErrInvalid):
				writeJSONError(w, http.StatusBadRequest, invalidRequestType, err.Error())
			case errors.Is(err, providers.ErrNoGatewayKey):
				writeJSONError(w, http.StatusConflict, gatewayCredentialMissingType, gatewayCredentialMissingMsg)
			case errors.Is(err, providers.ErrProxyAntigravityAuthFailed):
				a.logger.Warn("proxy antigravity auth start failed", "provider", id, "error", err)
				writeJSONError(w, http.StatusBadGateway, proxyAntigravityAuthFailedType, proxyAntigravityAuthFailureMessage(err))
			default:
				a.logger.Error("proxy antigravity auth start", "provider", id, "error", err.Error())
				writeJSONError(w, http.StatusInternalServerError, internalErrorType, adminInternalMsg)
			}
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"operation_id": payload.OperationID,
			"status":       payload.Status,
			"auth_url":     payload.AuthURL,
			"state":        payload.State,
			"expires_at":   payload.ExpiresAt,
		})
	})
}

// SyncModels handles POST /api/providers/{id}/sync-models.
func (a *ProviderAdmin) SyncModels() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.catalogManager == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "service_unavailable", "catalog manager is unavailable")
			return
		}
		id := r.PathValue("id")
		result, err := a.catalogManager.Sync(r.Context(), id)
		if err != nil {
			switch {
			case errors.Is(err, catalog.ErrProviderNotFound):
				writeJSONError(w, http.StatusNotFound, providerNotFoundType, "provider not found in catalog")
			case errors.Is(err, catalog.ErrProviderDisabled):
				writeJSONError(w, http.StatusConflict, "provider_disabled", "provider is disabled")
			case errors.Is(err, catalog.ErrSyncInProgress):
				writeJSONError(w, http.StatusConflict, "sync_in_progress", "a synchronization is already in progress")
			default:
				a.logger.Warn("provider catalog synchronization failed", "provider", id, "error", err)
				writeJSONError(w, http.StatusBadGateway, "sync_failed", "failed to synchronize models from upstream provider")
			}
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"provider":    result.ProviderID,
			"outcome":     result.Outcome,
			"counts":      result.Counts,
			"duration_ms": result.Duration.Milliseconds(),
		})
	})
}

// Status handles GET /api/providers/{id}/status.
func (a *ProviderAdmin) Status() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		record, err := a.service.Get(r.Context(), id)
		if err != nil {
			a.writeError(w, err)
			return
		}

		connState := "unknown"
		if a.statuses != nil {
			for _, st := range a.statuses() {
				if st.ID == id {
					connState = string(st.State)
					break
				}
			}
		}

		credState := "not_configured"
		if record.HasCredential {
			credState = "configured"
		}

		importedModels := 0
		var lastSynced *time.Time
		syncOutcome := "never"
		if a.catalogManager != nil {
			if syncState, ok := a.catalogManager.ProviderSyncState(id); ok {
				lastSynced = syncState.LastSuccessAt
				syncOutcome = string(syncState.Outcome)
				importedModels = syncState.Counts.Discovered
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"id":                    record.ID,
			"name":                  record.Name,
			"type":                  record.Type,
			"enabled":               record.Enabled,
			"connection_state":      connState,
			"credential_state":      credState,
			"last_validated_at":     record.UpdatedAt,
			"last_synced_at":        lastSynced,
			"imported_models_count": importedModels,
			"sync_outcome":          syncOutcome,
		})
	})
}

// Enable handles POST /api/providers/{id}/enable.
func (a *ProviderAdmin) Enable() http.Handler {
	return a.stateChange(a.service.Enable)
}

// Disable handles POST /api/providers/{id}/disable.
func (a *ProviderAdmin) Disable() http.Handler {
	return a.stateChange(a.service.Disable)
}

// Delete handles DELETE /api/providers/{id}.
func (a *ProviderAdmin) Delete() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := a.service.Delete(r.Context(), id); err != nil {
			a.writeError(w, err)
			return
		}
		if a.catalogManager != nil {
			a.catalogManager.RemoveProvider(id)
		}
		a.notifyProviderStatus()
		w.WriteHeader(http.StatusNoContent)
	})
}

func (a *ProviderAdmin) stateChange(change func(context.Context, string) (providers.Record, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record, err := change(r.Context(), r.PathValue("id"))
		if err != nil {
			a.writeError(w, err)

			return
		}

		writeJSON(w, http.StatusOK, toProviderResponse(record))
		a.notifyProviderStatus()
	})
}

func decodeProviderBody(w http.ResponseWriter, r *http.Request, target any) bool {
	body := http.MaxBytesReader(w, r.Body, providerRequestMaxBytes)
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil || decodeHasTrailingData(decoder) {
		writeJSONError(w, http.StatusBadRequest, invalidRequestType, invalidRequestMsg)

		return false
	}

	return true
}

func decodeHasTrailingData(decoder *json.Decoder) bool {
	_, err := decoder.Token()

	return !errors.Is(err, io.EOF)
}

func proxyCursorAuthFailureMessage(err error) string {
	var usageErr *providers.ProxyCursorUsageFailedError
	if errors.As(err, &usageErr) {
		if usageErr.Code == "browser_unavailable" {
			if strings.TrimSpace(usageErr.Detail) != "" {
				return fmt.Sprintf("%s Detalhe do proxy: %s", proxyCursorAuthBrowserUnavailableMsg, strings.TrimSpace(usageErr.Detail))
			}
			return proxyCursorAuthBrowserUnavailableMsg
		}
		if strings.TrimSpace(usageErr.Detail) != "" {
			return fmt.Sprintf("%s (%s)", proxyCursorAuthFailedMsg, strings.TrimSpace(usageErr.Detail))
		}
	}
	return proxyCursorAuthFailedMsg
}

func proxyAntigravityAuthFailureMessage(err error) string {
	var authErr *providers.ProxyAntigravityAuthFailedError
	if errors.As(err, &authErr) && strings.TrimSpace(authErr.Detail) != "" {
		return fmt.Sprintf("%s (%s)", proxyAntigravityAuthFailedMsg, strings.TrimSpace(authErr.Detail))
	}
	return proxyAntigravityAuthFailedMsg
}

func (a *ProviderAdmin) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, providers.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, providerNotFoundType, providerNotFoundMsg)
	case errors.Is(err, providers.ErrConflict):
		writeJSONError(w, http.StatusConflict, providerConflictType, providerConflictMsg)
	case errors.Is(err, providers.ErrInvalid):
		writeJSONError(w, http.StatusBadRequest, invalidRequestType, err.Error())
	case errors.Is(err, providers.ErrNoCredential):
		writeJSONError(w, http.StatusConflict, credentialMissingType, credentialMissingMsg)
	case errors.Is(err, providers.ErrNoGatewayKey):
		writeJSONError(w, http.StatusConflict, gatewayCredentialMissingType, gatewayCredentialMissingMsg)
	default:
		a.logger.Error("provider administration failed", "error", err.Error())
		writeJSONError(w, http.StatusInternalServerError, internalErrorType, adminInternalMsg)
	}
}
