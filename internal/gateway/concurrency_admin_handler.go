package gateway

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"aihub/internal/concurrency"
)

const (
	concurrencyPolicyNotFoundType  = "concurrency_policy_not_found"
	concurrencyPolicyNotFoundMsg   = "concurrency policy not found"
	concurrencyPolicyConflictType  = "concurrency_policy_conflict"
	concurrencyPolicyConflictMsg   = "concurrency policy already exists for scope"
	concurrencyAdminInternalMsg    = "the concurrency administration request failed"
)

// ConcurrencyAdmin serves E09.3 policy and usage administration.
type ConcurrencyAdmin struct {
	service *concurrency.Service
	logger  *slog.Logger
}

// NewConcurrencyAdmin builds administrative handlers.
func NewConcurrencyAdmin(service *concurrency.Service, logger *slog.Logger) *ConcurrencyAdmin {
	return &ConcurrencyAdmin{service: service, logger: logger}
}

type concurrencyPolicyResponse struct {
	ID               int64  `json:"id"`
	Scope            string `json:"scope"`
	ProviderID       string `json:"provider_id,omitempty"`
	ModelID          string `json:"model_id,omitempty"`
	ConcurrencyLimit int    `json:"concurrency_limit"`
	Enabled          bool   `json:"enabled"`
	UpdatedAt        string `json:"updated_at"`
}

type concurrencyUsageResponse struct {
	PolicyID         int64  `json:"policy_id"`
	Scope            string `json:"scope"`
	ProviderID       string `json:"provider_id,omitempty"`
	ModelID          string `json:"model_id,omitempty"`
	ConcurrencyLimit int    `json:"concurrency_limit"`
	Enabled          bool   `json:"enabled"`
	ActiveCount      int    `json:"active_count"`
	Available        int    `json:"available"`
}

type upsertConcurrencyPolicyRequest struct {
	Scope            string `json:"scope"`
	ProviderID       string `json:"provider_id"`
	ModelID          string `json:"model_id"`
	ConcurrencyLimit int    `json:"concurrency_limit"`
	Enabled          *bool  `json:"enabled"`
}

func (a *ConcurrencyAdmin) List() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		policies, err := a.service.ListPolicies(r.Context())
		if err != nil {
			a.writeError(w, err)
			return
		}
		out := make([]concurrencyPolicyResponse, 0, len(policies))
		for _, policy := range policies {
			out = append(out, concurrencyPolicyResponse{
				ID: policy.ID, Scope: string(policy.Scope.Kind),
				ProviderID: policy.Scope.ProviderID, ModelID: policy.Scope.ModelID,
				ConcurrencyLimit: policy.ConcurrencyLimit, Enabled: policy.Enabled,
				UpdatedAt: policy.UpdatedAt.UTC().Format(time.RFC3339),
			})
		}
		writeAdminJSON(w, http.StatusOK, map[string]any{"policies": out})
	})
}

func (a *ConcurrencyAdmin) Upsert() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request upsertConcurrencyPolicyRequest
		if !decodeResourceAdminBody(w, r, &request) {
			return
		}
		scope, err := scopeFromRequest(request.Scope, request.ProviderID, request.ModelID)
		if err != nil {
			writeAdminError(w, http.StatusBadRequest, invalidRequestType, err.Error())
			return
		}
		enabled := true
		if request.Enabled != nil {
			enabled = *request.Enabled
		}
		policy, created, err := a.service.UpsertPolicy(r.Context(), concurrency.UpsertPolicyInput{
			Scope:            concurrency.Scope(scope),
			ConcurrencyLimit: request.ConcurrencyLimit,
			Enabled:          enabled,
		})
		if err != nil {
			a.writeError(w, err)
			return
		}
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		writeAdminJSON(w, status, concurrencyPolicyResponse{
			ID: policy.ID, Scope: string(policy.Scope.Kind),
			ProviderID: policy.Scope.ProviderID, ModelID: policy.Scope.ModelID,
			ConcurrencyLimit: policy.ConcurrencyLimit, Enabled: policy.Enabled,
			UpdatedAt: policy.UpdatedAt.UTC().Format(time.RFC3339),
		})
	})
}

func (a *ConcurrencyAdmin) Usage() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := rejectUnknownQueryParams(r, "provider_id", "model_id"); err != nil {
			writeAdminError(w, http.StatusBadRequest, invalidRequestType, err.Error())
			return
		}
		query := concurrency.UsageQuery{
			ProviderID: strings.TrimSpace(r.URL.Query().Get("provider_id")),
			ModelID:    strings.TrimSpace(r.URL.Query().Get("model_id")),
		}
		snapshots, err := a.service.ListUsage(r.Context(), query)
		if err != nil {
			a.writeError(w, err)
			return
		}
		out := make([]concurrencyUsageResponse, 0, len(snapshots))
		for _, snapshot := range snapshots {
			out = append(out, concurrencyUsageResponse{
				PolicyID: snapshot.Policy.ID, Scope: string(snapshot.Policy.Scope.Kind),
				ProviderID: snapshot.Policy.Scope.ProviderID, ModelID: snapshot.Policy.Scope.ModelID,
				ConcurrencyLimit: snapshot.Policy.ConcurrencyLimit, Enabled: snapshot.Policy.Enabled,
				ActiveCount: snapshot.ActiveCount, Available: snapshot.Available,
			})
		}
		writeAdminJSON(w, http.StatusOK, map[string]any{"usage": out})
	})
}

func (a *ConcurrencyAdmin) Enable() http.Handler  { return a.setEnabled(true) }
func (a *ConcurrencyAdmin) Disable() http.Handler { return a.setEnabled(false) }

func (a *ConcurrencyAdmin) setEnabled(enabled bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := parseResourcePolicyID(r.PathValue("id"))
		if err != nil {
			writeAdminError(w, http.StatusBadRequest, invalidRequestType, "invalid concurrency policy id")
			return
		}
		policy, err := a.service.SetPolicyEnabled(r.Context(), id, enabled)
		if err != nil {
			a.writeError(w, err)
			return
		}
		writeAdminJSON(w, http.StatusOK, concurrencyPolicyResponse{
			ID: policy.ID, Scope: string(policy.Scope.Kind),
			ProviderID: policy.Scope.ProviderID, ModelID: policy.Scope.ModelID,
			ConcurrencyLimit: policy.ConcurrencyLimit, Enabled: policy.Enabled,
			UpdatedAt: policy.UpdatedAt.UTC().Format(time.RFC3339),
		})
	})
}

func (a *ConcurrencyAdmin) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, concurrency.ErrNotFound):
		writeAdminError(w, http.StatusNotFound, concurrencyPolicyNotFoundType, concurrencyPolicyNotFoundMsg)
	case errors.Is(err, concurrency.ErrConflict):
		writeAdminError(w, http.StatusConflict, concurrencyPolicyConflictType, concurrencyPolicyConflictMsg)
	case errors.Is(err, concurrency.ErrInvalidInput):
		writeAdminError(w, http.StatusBadRequest, invalidRequestType, err.Error())
	default:
		a.logger.Error("concurrency administration failed", "error", err.Error())
		writeAdminError(w, http.StatusInternalServerError, internalErrorType, concurrencyAdminInternalMsg)
	}
}
