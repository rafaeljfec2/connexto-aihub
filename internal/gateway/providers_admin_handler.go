package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

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
	service *providers.Service
	logger  *slog.Logger
}

// NewProviderAdmin builds the administrative handlers.
func NewProviderAdmin(service *providers.Service, logger *slog.Logger) *ProviderAdmin {
	return &ProviderAdmin{service: service, logger: logger}
}

type providerResponse struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Type                 string    `json:"type"`
	Endpoint             string    `json:"endpoint"`
	Enabled              bool      `json:"enabled"`
	CredentialConfigured bool      `json:"credential_configured"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type createProviderRequest struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Endpoint   string `json:"endpoint"`
	Credential string `json:"credential"`
	Enabled    *bool  `json:"enabled"`
}

type updateProviderRequest struct {
	Name       *string `json:"name"`
	Endpoint   *string `json:"endpoint"`
	Credential *string `json:"credential"`
}

func toProviderResponse(record providers.Record) providerResponse {
	return providerResponse{
		ID:                   record.ID,
		Name:                 record.Name,
		Type:                 record.Type,
		Endpoint:             record.Endpoint,
		Enabled:              record.Enabled,
		CredentialConfigured: record.HasCredential,
		CreatedAt:            record.CreatedAt,
		UpdatedAt:            record.UpdatedAt,
	}
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

		record, err := a.service.Create(r.Context(), providers.CreateInput{
			ID:         request.ID,
			Name:       request.Name,
			Type:       request.Type,
			Endpoint:   request.Endpoint,
			Credential: request.Credential,
			Enabled:    enabled,
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
			Name:       request.Name,
			Endpoint:   request.Endpoint,
			Credential: request.Credential,
		})
		if err != nil {
			a.writeError(w, err)

			return
		}

		writeJSON(w, http.StatusOK, toProviderResponse(record))
	})
}

// Validate handles POST /api/providers/{id}/validate. It changes no state.
func (a *ProviderAdmin) Validate() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := a.service.Validate(r.Context(), r.PathValue("id"))
		if err == nil {
			writeJSON(w, http.StatusOK, map[string]string{"status": "reachable"})

			return
		}

		switch {
		case errors.Is(err, providers.ErrNoCredential):
			writeJSONError(w, http.StatusConflict, credentialMissingType, credentialMissingMsg)
		case errors.Is(err, providers.ErrNotFound), errors.Is(err, providers.ErrCredentialUnreadable):
			a.writeError(w, err)
		default:
			a.logger.Warn("provider validation failed", "provider", r.PathValue("id"), "error", err.Error())
			writeJSONError(w, http.StatusBadGateway, providerUnreachableType, providerUnreachableMsg)
		}
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
