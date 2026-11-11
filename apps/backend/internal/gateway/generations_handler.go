package gateway

import (
	"net/http"
	"time"

	"aihub/internal/generations"
	"aihub/internal/records"
)

type generationUsageDTO struct {
	Status               string  `json:"status"`
	Source               string  `json:"source"`
	InputTokens          *int64  `json:"input_tokens"`
	OutputTokens         *int64  `json:"output_tokens"`
	ProviderReportedCost *string `json:"provider_reported_cost"`
	ProviderListCost     *string `json:"provider_list_cost"`
	Currency             *string `json:"currency"`
}

type generationTelemetryDTO struct {
	Streaming              *bool   `json:"streaming"`
	ProviderLatencyNS      *int64  `json:"provider_latency_ns"`
	RoutingOverheadNS      *int64  `json:"routing_overhead_ns"`
	TimeToFirstTokenNS     *int64  `json:"time_to_first_token_ns"`
	OutputStreamDurationNS *int64  `json:"output_stream_duration_ns"`
	FinishReason           *string `json:"finish_reason"`
}

type generationListDTO struct {
	ExecutionID      string                 `json:"execution_id"`
	RequestID        *string                `json:"request_id"`
	ProviderID       string                 `json:"provider_id"`
	ModelID          *string                `json:"model_id"`
	StartedAt        string                 `json:"started_at"`
	CompletedAt      string                 `json:"completed_at"`
	DurationNS       int64                  `json:"duration_ns"`
	Outcome          string                 `json:"outcome"`
	Phase            string                 `json:"phase"`
	Cause            *string                `json:"cause"`
	UpstreamStatus   *int                   `json:"upstream_status"`
	Usage            generationUsageDTO     `json:"usage"`
	Telemetry        generationTelemetryDTO `json:"telemetry"`
	EnrichmentStatus string                 `json:"enrichment_status"`
}

type generationEventDTO struct {
	Sequence       int    `json:"sequence"`
	Type           string `json:"type"`
	ObservedAt     string `json:"observed_at"`
	ProviderID     string `json:"provider_id"`
	ModelID        string `json:"model_id"`
	UpstreamStatus *int   `json:"upstream_status"`
	DurationNS     *int64 `json:"duration_ns"`
}

type generationsListDTO struct {
	Generations []generationListDTO `json:"generations"`
	Count       int                 `json:"count"`
	NextCursor  *string             `json:"next_cursor"`
	Degraded    bool                `json:"degraded"`
}

type generationDetailDTO struct {
	generationListDTO
	ProviderSessionID *string              `json:"provider_session_id"`
	ProviderResponses []generationEventDTO `json:"provider_responses"`
	Degraded          bool                 `json:"degraded"`
}

// NewGenerationsHandler exposes a filtered cursor page of content-free generations.
func NewGenerationsHandler(service *generations.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		filter, err := parseRecordFilter(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, invalidRequestType, err.Error())
			return
		}
		page, err := service.List(r.Context(), filter)
		if err != nil {
			writeJSONError(w, http.StatusServiceUnavailable, recordsUnavailableType, recordsUnavailableMessage)
			return
		}
		payload := generationsListDTO{
			Generations: make([]generationListDTO, 0, len(page.Generations)),
			Count:       page.Count,
			Degraded:    page.Degraded,
		}
		for _, generation := range page.Generations {
			payload.Generations = append(payload.Generations, newGenerationListDTO(generation))
		}
		if page.NextCursor != "" {
			payload.NextCursor = &page.NextCursor
		}
		writeJSON(w, http.StatusOK, payload)
	})
}

// NewGenerationHandler exposes one retained generation and its optional detail metadata.
func NewGenerationHandler(service *generations.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		executionID := r.PathValue("execution_id")
		if !records.ValidExecutionID(executionID) {
			writeJSONError(w, http.StatusBadRequest, invalidRequestType, "invalid execution_id")
			return
		}
		generation, found, degraded, err := service.Get(r.Context(), executionID)
		if err != nil {
			writeJSONError(w, http.StatusServiceUnavailable, recordsUnavailableType, recordsUnavailableMessage)
			return
		}
		if !found {
			writeJSONError(w, http.StatusNotFound, recordNotFoundType, recordNotFoundMessage)
			return
		}
		payload := generationDetailDTO{
			generationListDTO: newGenerationListDTO(generation),
			ProviderSessionID: generation.Telemetry.ProviderSessionID,
			ProviderResponses: make([]generationEventDTO, 0, len(generation.ProviderEvents)),
			Degraded:          degraded,
		}
		for _, event := range generation.ProviderEvents {
			payload.ProviderResponses = append(payload.ProviderResponses, generationEventDTO{
				Sequence:       event.Sequence,
				Type:           event.Type,
				ObservedAt:     event.ObservedAt.UTC().Format(time.RFC3339Nano),
				ProviderID:     event.ProviderID,
				ModelID:        event.ModelID,
				UpstreamStatus: event.UpstreamStatus,
				DurationNS:     event.DurationNS,
			})
		}
		writeJSON(w, http.StatusOK, payload)
	})
}

func newGenerationListDTO(generation generations.Generation) generationListDTO {
	record := generation.Record
	return generationListDTO{
		ExecutionID:      record.ExecutionID,
		RequestID:        record.RequestID,
		ProviderID:       record.ProviderID,
		ModelID:          record.ModelID,
		StartedAt:        record.StartedAt.UTC().Format(time.RFC3339Nano),
		CompletedAt:      record.CompletedAt.UTC().Format(time.RFC3339Nano),
		DurationNS:       record.DurationNS,
		Outcome:          record.Outcome,
		Phase:            record.Phase,
		Cause:            record.Cause,
		UpstreamStatus:   record.UpstreamStatus,
		EnrichmentStatus: generation.EnrichmentStatus,
		Usage: generationUsageDTO{
			Status:               generation.Usage.Status,
			Source:               generation.Usage.Source,
			InputTokens:          generation.Usage.InputTokens,
			OutputTokens:         generation.Usage.OutputTokens,
			ProviderReportedCost: generation.Usage.ProviderReportedCost,
			ProviderListCost:     generation.Usage.ProviderListCost,
			Currency:             generation.Usage.Currency,
		},
		Telemetry: generationTelemetryDTO{
			Streaming:              generation.Telemetry.Streaming,
			ProviderLatencyNS:      generation.Telemetry.ProviderLatencyNS,
			RoutingOverheadNS:      generation.Telemetry.RoutingOverheadNS,
			TimeToFirstTokenNS:     generation.Telemetry.TimeToFirstTokenNS,
			OutputStreamDurationNS: generation.Telemetry.OutputStreamDurationNS,
			FinishReason:           generation.Telemetry.FinishReason,
		},
	}
}
