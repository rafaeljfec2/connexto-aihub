package gateway

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"aihub/internal/resources"
)

const (
	resourcePolicyNotFoundType = "resource_policy_not_found"
	resourcePolicyNotFoundMsg  = "resource policy not found"
	resourceAdminInternalMsg   = "the resource administration request failed"
	resourceRequestMaxBytes    = 4 << 10
)

// ResourceAdmin serves E09 policy and usage administration (PostgreSQL mode only).
type ResourceAdmin struct {
	service *resources.Service
	logger  *slog.Logger
}

// NewResourceAdmin builds the administrative handlers.
func NewResourceAdmin(service *resources.Service, logger *slog.Logger) *ResourceAdmin {
	return &ResourceAdmin{service: service, logger: logger}
}

type resourcePolicyResponse struct {
	ID         int64     `json:"id"`
	Scope      string    `json:"scope"`
	ProviderID string    `json:"provider_id,omitempty"`
	ModelID    string    `json:"model_id,omitempty"`
	DailyLimit int       `json:"daily_limit"`
	Enabled    bool      `json:"enabled"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type resourceUsageResponse struct {
	PolicyID           int64     `json:"policy_id"`
	Scope              string    `json:"scope"`
	ProviderID         string    `json:"provider_id,omitempty"`
	ModelID            string    `json:"model_id,omitempty"`
	DailyLimit         int       `json:"daily_limit"`
	Enabled            bool      `json:"enabled"`
	PeriodStart        time.Time `json:"period_start"`
	ResetAt            time.Time `json:"reset_at"`
	AdmittedCount      int       `json:"admitted_count"`
	RemainingRequests  int       `json:"remaining_requests"`
}

type upsertResourcePolicyRequest struct {
	Scope      string `json:"scope"`
	ProviderID string `json:"provider_id"`
	ModelID    string `json:"model_id"`
	DailyLimit int    `json:"daily_limit"`
	Enabled    *bool  `json:"enabled"`
}

func toResourcePolicyResponse(policy resources.ResourcePolicy) resourcePolicyResponse {
	return resourcePolicyResponse{
		ID:         policy.ID,
		Scope:      string(policy.Scope.Kind),
		ProviderID: policy.Scope.ProviderID,
		ModelID:    policy.Scope.ModelID,
		DailyLimit: policy.DailyLimit,
		Enabled:    policy.Enabled,
		UpdatedAt:  policy.UpdatedAt.UTC(),
	}
}

func remainingRequests(policy resources.ResourcePolicy, admitted int) int {
	if !policy.Enabled {
		return 0
	}
	remaining := policy.DailyLimit - admitted
	if remaining < 0 {
		return 0
	}
	return remaining
}

func writeAdminJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, status, payload)
}

func writeAdminError(w http.ResponseWriter, status int, failureType, message string) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSONError(w, status, failureType, message)
}

// List handles GET /api/resource-policies.
func (a *ResourceAdmin) List() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		policies, err := a.service.ListPolicies(r.Context())
		if err != nil {
			a.writeError(w, err)
			return
		}
		responses := make([]resourcePolicyResponse, 0, len(policies))
		for _, policy := range policies {
			responses = append(responses, toResourcePolicyResponse(policy))
		}
		writeAdminJSON(w, http.StatusOK, map[string]any{"policies": responses})
	})
}

// Upsert handles PUT /api/resource-policies.
func (a *ResourceAdmin) Upsert() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request upsertResourcePolicyRequest
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

		_, lookupErr := a.service.GetPolicyByScope(r.Context(), scope)
		policy, err := a.service.UpsertPolicy(r.Context(), resources.UpsertPolicyInput{
			Scope: scope, DailyLimit: request.DailyLimit, Enabled: enabled,
		})
		if err != nil {
			a.writeError(w, err)
			return
		}

		status := http.StatusOK
		if errors.Is(lookupErr, resources.ErrNotFound) {
			status = http.StatusCreated
		}
		writeAdminJSON(w, status, toResourcePolicyResponse(policy))
	})
}

// Usage handles GET /api/resource-usage.
func (a *ResourceAdmin) Usage() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query, err := usageQueryFromRequest(r)
		if err != nil {
			writeAdminError(w, http.StatusBadRequest, invalidRequestType, err.Error())
			return
		}
		snapshots, err := a.service.ListUsage(r.Context(), query)
		if err != nil {
			a.writeError(w, err)
			return
		}
		responses := make([]resourceUsageResponse, 0, len(snapshots))
		for _, snapshot := range snapshots {
			responses = append(responses, resourceUsageResponse{
				PolicyID:          snapshot.Policy.ID,
				Scope:             string(snapshot.Policy.Scope.Kind),
				ProviderID:        snapshot.Policy.Scope.ProviderID,
				ModelID:           snapshot.Policy.Scope.ModelID,
				DailyLimit:        snapshot.Policy.DailyLimit,
				Enabled:           snapshot.Policy.Enabled,
				PeriodStart:       snapshot.PeriodStart.UTC(),
				ResetAt:           snapshot.ResetAt.UTC(),
				AdmittedCount:     snapshot.AdmittedCount,
				RemainingRequests: remainingRequests(snapshot.Policy, snapshot.AdmittedCount),
			})
		}
		writeAdminJSON(w, http.StatusOK, map[string]any{"usage": responses})
	})
}

// Enable handles POST /api/resource-policies/{id}/enable.
func (a *ResourceAdmin) Enable() http.Handler {
	return a.setEnabled(true)
}

// Disable handles POST /api/resource-policies/{id}/disable.
func (a *ResourceAdmin) Disable() http.Handler {
	return a.setEnabled(false)
}

func (a *ResourceAdmin) setEnabled(enabled bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := parseResourcePolicyID(r.PathValue("id"))
		if err != nil {
			writeAdminError(w, http.StatusBadRequest, invalidRequestType, "invalid resource policy id")
			return
		}
		policy, err := a.service.SetPolicyEnabled(r.Context(), id, enabled)
		if err != nil {
			a.writeError(w, err)
			return
		}
		writeAdminJSON(w, http.StatusOK, toResourcePolicyResponse(policy))
	})
}

func scopeFromRequest(scope, providerID, modelID string) (resources.ResourceScope, error) {
	kind := resources.ScopeKind(strings.TrimSpace(scope))
	input := resources.ResourceScope{
		Kind:       kind,
		ProviderID: strings.TrimSpace(providerID),
		ModelID:    strings.TrimSpace(modelID),
	}
	switch kind {
	case resources.ScopeGlobal:
		if input.ProviderID != "" || input.ModelID != "" {
			return resources.ResourceScope{}, errors.New("global scope must not include provider_id or model_id")
		}
	case resources.ScopeProvider:
		if input.ProviderID == "" || input.ModelID != "" {
			return resources.ResourceScope{}, errors.New("provider scope requires provider_id only")
		}
	case resources.ScopeModel:
		if input.ProviderID == "" || input.ModelID == "" {
			return resources.ResourceScope{}, errors.New("model scope requires provider_id and model_id")
		}
	default:
		return resources.ResourceScope{}, errors.New("scope must be global, provider, or model")
	}
	return input, nil
}

func usageQueryFromRequest(r *http.Request) (resources.UsageQuery, error) {
	query := resources.UsageQuery{}
	for name := range r.URL.Query() {
		switch name {
		case "provider_id", "model_id", "period_start":
		default:
			return resources.UsageQuery{}, errors.New("unknown query parameter")
		}
	}
	query.ProviderID = strings.TrimSpace(r.URL.Query().Get("provider_id"))
	query.ModelID = strings.TrimSpace(r.URL.Query().Get("model_id"))
	if raw := strings.TrimSpace(r.URL.Query().Get("period_start")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return resources.UsageQuery{}, errors.New("period_start must be an RFC3339 timestamp")
		}
		query.PeriodStart = parsed.UTC()
	}
	return query, nil
}

func parseResourcePolicyID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func decodeResourceAdminBody(w http.ResponseWriter, r *http.Request, target any) bool {
	body := http.MaxBytesReader(w, r.Body, resourceRequestMaxBytes)
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil || decodeHasTrailingData(decoder) {
		writeAdminError(w, http.StatusBadRequest, invalidRequestType, invalidRequestMsg)
		return false
	}
	return true
}

func (a *ResourceAdmin) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, resources.ErrNotFound):
		writeAdminError(w, http.StatusNotFound, resourcePolicyNotFoundType, resourcePolicyNotFoundMsg)
	case errors.Is(err, resources.ErrConflict):
		writeAdminError(w, http.StatusConflict, providerConflictType, providerConflictMsg)
	case errors.Is(err, resources.ErrInvalidInput):
		writeAdminError(w, http.StatusBadRequest, invalidRequestType, err.Error())
	default:
		a.logger.Error("resource administration failed", "error", err.Error())
		writeAdminError(w, http.StatusInternalServerError, internalErrorType, resourceAdminInternalMsg)
	}
}
