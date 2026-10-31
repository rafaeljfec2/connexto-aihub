package gateway

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"aihub/internal/catalog"
)

const (
	invalidRequestType = "invalid_request"
	internalErrorType  = "internal_error"
	maxSyncBodyBytes   = 4 << 10
)

type catalogCapabilityDTO struct {
	State        string `json:"state"`
	Provenance   string `json:"provenance"`
	Confidence   string `json:"confidence"`
	Verification string `json:"verification"`
}

type catalogModalitiesDTO struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

type catalogLimitsDTO struct {
	Context *int64 `json:"context"`
	Input   *int64 `json:"input"`
	Output  *int64 `json:"output"`
}

type catalogModelDTO struct {
	Provider              string                          `json:"provider"`
	ModelID               string                          `json:"model_id"`
	CanonicalID           string                          `json:"canonical_id,omitempty"`
	Name                  string                          `json:"name"`
	Type                  string                          `json:"type,omitempty"`
	Family                string                          `json:"family,omitempty"`
	Modalities            *catalogModalitiesDTO           `json:"modalities,omitempty"`
	Limits                catalogLimitsDTO                `json:"limits"`
	Capabilities          map[string]catalogCapabilityDTO `json:"capabilities"`
	Presence              string                          `json:"presence"`
	Freshness             string                          `json:"freshness"`
	ProviderState         string                          `json:"provider_state"`
	ExecutionVerification string                          `json:"execution_verification"`
	ListingState          string                          `json:"listing_state"`
	ListingReasons        []string                        `json:"listing_reasons"`
	FirstSeenAt           time.Time                       `json:"first_seen_at"`
	LastSeenAt            time.Time                       `json:"last_seen_at"`
	MissingSince          *time.Time                      `json:"missing_since,omitempty"`
	ProviderMetadata      json.RawMessage                 `json:"provider_metadata,omitempty"`
}

type catalogListMeta struct {
	Count       int       `json:"count"`
	GeneratedAt time.Time `json:"generated_at"`
}

type catalogListResponse struct {
	Object string            `json:"object"`
	Data   []catalogModelDTO `json:"data"`
	Meta   catalogListMeta   `json:"meta"`
}

type catalogCountsDTO struct {
	Discovered int `json:"discovered"`
	Added      int `json:"added"`
	Updated    int `json:"updated"`
	Unchanged  int `json:"unchanged"`
	Missing    int `json:"missing"`
}

type catalogSyncDTO struct {
	LastAttemptAt  *time.Time       `json:"last_attempt_at,omitempty"`
	LastSuccessAt  *time.Time       `json:"last_success_at,omitempty"`
	Outcome        string           `json:"outcome"`
	ErrorClass     string           `json:"error_class"`
	UpstreamStatus int              `json:"upstream_status"`
	Counts         catalogCountsDTO `json:"counts"`
}

type catalogProviderModelsDTO struct {
	Known   int `json:"known"`
	Present int `json:"present"`
	Missing int `json:"missing"`
}

type catalogProviderDTO struct {
	Provider      string                   `json:"provider"`
	ProviderState string                   `json:"provider_state"`
	Freshness     string                   `json:"freshness"`
	Sync          catalogSyncDTO           `json:"sync"`
	Models        catalogProviderModelsDTO `json:"models"`
}

type catalogProvidersResponse struct {
	Object string               `json:"object"`
	Data   []catalogProviderDTO `json:"data"`
}

type catalogSyncRequest struct {
	Provider string `json:"provider"`
}

type catalogSyncResultDTO struct {
	Provider   string           `json:"provider"`
	Outcome    string           `json:"outcome"`
	Counts     catalogCountsDTO `json:"counts"`
	DurationMS int64            `json:"duration_ms"`
}

type catalogSyncResponse struct {
	Providers []catalogSyncResultDTO `json:"providers"`
}

// NewCatalogModelsHandler exposes the normalized catalog listing.
func NewCatalogModelsHandler(manager *catalog.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filter, err := parseCatalogFilter(r)
		if err != nil {
			writeCatalogError(w, http.StatusBadRequest, invalidRequestType, err.Error())

			return
		}

		views := manager.List(filter)
		data := make([]catalogModelDTO, 0, len(views))

		for _, view := range views {
			data = append(data, catalogModelView(view, false))
		}

		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, catalogListResponse{
			Object: "list",
			Data:   data,
			Meta: catalogListMeta{
				Count:       len(data),
				GeneratedAt: time.Now().UTC(),
			},
		})
	})
}

// NewCatalogModelHandler exposes one model with its provider metadata.
func NewCatalogModelHandler(manager *catalog.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerID := r.PathValue("provider")
		modelID := r.PathValue("model")

		if _, ok := manager.ProviderSyncState(providerID); !ok {
			writeCatalogError(w, http.StatusNotFound, "provider_not_found", "provider not found")

			return
		}

		view, ok := manager.Get(providerID, modelID)
		if !ok {
			writeCatalogError(w, http.StatusNotFound, "model_not_found", "model not found")

			return
		}

		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, catalogModelView(view, true))
	})
}

// NewCatalogProvidersHandler exposes provider sync state and catalog counts.
func NewCatalogProvidersHandler(manager *catalog.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		views := manager.Providers()
		data := make([]catalogProviderDTO, 0, len(views))

		for _, view := range views {
			data = append(data, catalogProviderView(view))
		}

		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, catalogProvidersResponse{Object: "list", Data: data})
	})
}

// NewCatalogSyncHandler triggers a bounded synchronization. A snapshot write
// failure publishes nothing and answers 500.
func NewCatalogSyncHandler(manager *catalog.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request catalogSyncRequest

		body := http.MaxBytesReader(w, r.Body, maxSyncBodyBytes)
		decoder := json.NewDecoder(body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&request); err != nil && !errors.Is(err, io.EOF) {
			writeCatalogError(w, http.StatusBadRequest, invalidRequestType, "invalid request body")

			return
		}

		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			writeCatalogError(w, http.StatusBadRequest, invalidRequestType, "invalid request body")

			return
		}

		providerIDs := manager.ProviderIDs()
		if request.Provider != "" {
			providerIDs = []string{request.Provider}
		}

		results := make([]catalogSyncResultDTO, 0, len(providerIDs))
		var firstErr error

		for _, providerID := range providerIDs {
			result, err := manager.Sync(r.Context(), providerID)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}

				continue
			}

			results = append(results, catalogSyncResultDTO{
				Provider:   result.ProviderID,
				Outcome:    string(result.Outcome),
				Counts:     catalogCounts(result.Counts),
				DurationMS: result.Duration.Milliseconds(),
			})
		}

		if len(results) == 0 && firstErr != nil {
			writeCatalogSyncError(w, firstErr)

			return
		}

		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, catalogSyncResponse{Providers: results})
	})
}

func parseCatalogFilter(r *http.Request) (catalog.Filter, error) {
	query := r.URL.Query()
	filter := catalog.Filter{
		ProviderID: query.Get("provider"),
		Capability: query.Get("capability"),
	}

	if filter.Capability != "" && !catalog.KnownCapability(filter.Capability) {
		return catalog.Filter{}, errors.New("unknown capability filter")
	}

	if value := query.Get("presence"); value != "" {
		presence := catalog.Presence(value)

		switch presence {
		case catalog.PresencePresent, catalog.PresenceMissing, catalog.PresenceUnknown:
		default:
			return catalog.Filter{}, errors.New("invalid presence filter")
		}

		filter.Presence = &presence
	}

	if value := query.Get("listing_state"); value != "" {
		listingState := catalog.ListingState(value)

		switch listingState {
		case catalog.ListingCurrent, catalog.ListingStale, catalog.ListingMissing, catalog.ListingUnknown:
		default:
			return catalog.Filter{}, errors.New("invalid listing_state filter")
		}

		filter.ListingState = &listingState
	}

	return filter, nil
}

func catalogModelView(view catalog.View, includeMetadata bool) catalogModelDTO {
	capabilities := make(map[string]catalogCapabilityDTO, len(view.Entry.Capabilities))

	for name, claim := range view.Entry.Capabilities {
		capabilities[name] = catalogCapabilityDTO{
			State:        string(claim.State),
			Provenance:   string(claim.Provenance),
			Confidence:   string(claim.Confidence),
			Verification: string(claim.Verification),
		}
	}

	reasons := make([]string, 0, len(view.ListingReasons))
	for _, reason := range view.ListingReasons {
		reasons = append(reasons, string(reason))
	}

	dto := catalogModelDTO{
		Provider:              view.Entry.ProviderID,
		ModelID:               view.Entry.ModelID,
		CanonicalID:           view.Entry.CanonicalID,
		Name:                  view.Entry.Name,
		Type:                  view.Entry.Type,
		Family:                view.Entry.Family,
		Limits:                catalogLimitsDTO{Context: view.Entry.Limits.Context, Input: view.Entry.Limits.Input, Output: view.Entry.Limits.Output},
		Capabilities:          capabilities,
		Presence:              string(view.Presence),
		Freshness:             string(view.Freshness),
		ProviderState:         string(view.ProviderState),
		ExecutionVerification: string(view.ExecutionVerification),
		ListingState:          string(view.ListingState),
		ListingReasons:        reasons,
		FirstSeenAt:           view.Entry.FirstSeenAt,
		LastSeenAt:            view.Entry.LastSeenAt,
	}

	if view.Entry.Modalities != nil {
		dto.Modalities = &catalogModalitiesDTO{
			Input:  view.Entry.Modalities.Input,
			Output: view.Entry.Modalities.Output,
		}
	}

	if includeMetadata {
		dto.MissingSince = view.Entry.MissingSince
		dto.ProviderMetadata = view.Entry.ProviderMetadata
	}

	return dto
}

func catalogProviderView(view catalog.ProviderView) catalogProviderDTO {
	return catalogProviderDTO{
		Provider:      view.ProviderID,
		ProviderState: string(view.ProviderState),
		Freshness:     string(view.Freshness),
		Sync: catalogSyncDTO{
			LastAttemptAt:  view.Sync.LastAttemptAt,
			LastSuccessAt:  view.Sync.LastSuccessAt,
			Outcome:        string(view.Sync.Outcome),
			ErrorClass:     string(view.Sync.ErrorClass),
			UpstreamStatus: view.Sync.UpstreamStatus,
			Counts:         catalogCounts(view.Sync.Counts),
		},
		Models: catalogProviderModelsDTO{
			Known:   view.ModelsKnown,
			Present: view.ModelsPresent,
			Missing: view.ModelsMissing,
		},
	}
}

func catalogCounts(counts catalog.SyncCounts) catalogCountsDTO {
	return catalogCountsDTO{
		Discovered: counts.Discovered,
		Added:      counts.Added,
		Updated:    counts.Updated,
		Unchanged:  counts.Unchanged,
		Missing:    counts.Missing,
	}
}

func writeCatalogSyncError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, catalog.ErrProviderNotFound):
		writeCatalogError(w, http.StatusNotFound, "provider_not_found", "provider not found")
	case errors.Is(err, catalog.ErrProviderDisabled):
		writeCatalogError(w, http.StatusConflict, "provider_disabled", "provider is disabled")
	case errors.Is(err, catalog.ErrSyncInProgress):
		writeCatalogError(w, http.StatusConflict, "sync_in_progress", "a synchronization is already in progress")
	case errors.Is(err, catalog.ErrPersistence), errors.Is(err, catalog.ErrStopped):
		writeCatalogError(w, http.StatusInternalServerError, internalErrorType, "the catalog request could not be completed")
	default:
		writeCatalogError(w, http.StatusBadGateway, upstreamUnavailableType, upstreamUnavailableMsg)
	}
}

func writeCatalogError(w http.ResponseWriter, status int, failureType, message string) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSONError(w, status, failureType, message)
}
