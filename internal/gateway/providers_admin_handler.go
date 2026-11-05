package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/operational"
	"aihub/internal/providers"
)

const (
	providerNotFoundType    = "provider_not_found"
	providerConflictType    = "provider_conflict"
	credentialMissingType   = "credential_missing"
	providerUnreachableType = "provider_unreachable"

	providerRequestMaxBytes = 16 << 10
	providerNotFoundMsg     = "provider configuration not found"
	providerConflictMsg     = "provider configuration already exists"
	invalidRequestMsg       = "request body is not a valid provider payload"
	credentialMissingMsg    = "provider has no credential configured"
	providerUnreachableMsg  = "the provider rejected the connectivity check or is unreachable"
	adminInternalMsg        = "the provider administration request failed"
)

// ProviderAdmin serves the administrative provider endpoints. Responses carry
// the configuration only; the credential is never returned, logged or echoed.
type ProviderAdmin struct {
	service        *providers.Service
	logger         *slog.Logger
	eventHub       *operational.Hub
	statuses       func() []providers.Status
	quotaService   *providers.QuotaService
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

// Quotas handles GET /api/ui/providers/quotas.
func (a *ProviderAdmin) Quotas() http.Handler {
	return NewProviderQuotasHandler(a.quotaService, a.logger)
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
			if decErr := json.Unmarshal(bodyBytes, &validateReq); decErr == nil && validateReq.Credential != "" {
				err = a.service.ValidateCredential(r.Context(), id, validateReq.Credential)
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
		case errors.Is(err, providers.ErrNotFound), errors.Is(err, providers.ErrCredentialUnreadable):
			a.writeError(w, err)
		default:
			a.logger.Warn("provider validation failed", "provider", id)
			writeJSONError(w, http.StatusBadGateway, providerUnreachableType, providerUnreachableMsg)
		}
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
	default:
		a.logger.Error("provider administration failed", "error", err.Error())
		writeJSONError(w, http.StatusInternalServerError, internalErrorType, adminInternalMsg)
	}
}
