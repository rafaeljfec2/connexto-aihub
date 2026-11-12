package generations

import (
	"context"
	"log/slog"
	"time"

	"aihub/internal/records"
)

const persistenceTimeout = 2 * time.Second

// Service combines retained execution records with optional persisted enrichment.
type Service struct {
	records         *records.Service
	repository      Repository
	logger          *slog.Logger
	retention       RetentionConfig
	cancelRetention context.CancelFunc
	retentionDone   chan struct{}
}

// NewService builds a generation projection. A nil repository keeps E06 usable in file mode.
func NewService(recordService *records.Service, repository Repository, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{records: recordService, repository: repository, logger: logger}
}

// WithRetention configures periodic cleanup to match the E06 retention cadence.
func (s *Service) WithRetention(interval time.Duration) *Service {
	if interval > 0 {
		s.retention.Interval = interval
	}
	return s
}

// List returns one E06 cursor page with batched optional enrichment.
func (s *Service) List(ctx context.Context, filter records.Filter) (Page, error) {
	page, err := s.records.List(filter)
	if err != nil {
		return Page{}, err
	}
	result := Page{
		Generations: make([]Generation, 0, len(page.Records)),
		Count:       len(page.Records),
		NextCursor:  page.NextCursor,
		Degraded:    s.records.Degraded() || s.repository == nil,
	}
	for _, record := range page.Records {
		result.Generations = append(result.Generations, emptyGeneration(record))
	}
	if len(page.Records) == 0 || s.repository == nil {
		return result, nil
	}

	enrichment, err := s.repository.List(ctx, executionIDs(page.Records))
	if err != nil {
		s.logEnrichmentFailure("generation_list_enrichment_failed")
		result.Degraded = true
		return result, nil
	}
	for index, record := range page.Records {
		result.Generations[index] = applyEnrichment(result.Generations[index], enrichment[record.ExecutionID])
	}

	return result, nil
}

// Get returns one retained execution with optional detail enrichment.
func (s *Service) Get(ctx context.Context, executionID string) (Generation, bool, bool, error) {
	record, found, err := s.records.Get(executionID)
	if err != nil || !found {
		return Generation{}, found, false, err
	}
	generation := emptyGeneration(record)
	if s.repository == nil {
		return generation, true, true, nil
	}

	enrichment, err := s.repository.List(ctx, []string{executionID})
	if err != nil {
		s.logEnrichmentFailure("generation_detail_enrichment_failed")
		return generation, true, true, nil
	}
	return applyEnrichment(generation, enrichment[executionID]), true, s.records.Degraded(), nil
}

// RecordBestEffort stores gateway observations without affecting response delivery.
func (s *Service) RecordBestEffort(ctx context.Context, input TelemetryInput) {
	if s == nil || s.repository == nil {
		return
	}
	writeContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), persistenceTimeout)
	defer cancel()
	if sessionID := safeSessionID(input.SessionID); sessionID != nil {
		input.SessionID = *sessionID
	} else {
		input.SessionID = ""
	}
	if clientApp := SafeClientApp(input.ClientApp); clientApp != nil {
		input.ClientApp = *clientApp
	} else {
		input.ClientApp = ""
	}
	if err := s.repository.Insert(writeContext, input); err != nil {
		s.logEnrichmentFailure("generation_telemetry_persist_failed")
	}
}

func (s *Service) logEnrichmentFailure(class string) {
	s.logger.Warn("generation enrichment unavailable", "class", class)
}

func executionIDs(records []records.Record) []string {
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ExecutionID)
	}
	return ids
}

func emptyGeneration(record records.Record) Generation {
	return Generation{
		Record:           record,
		Usage:            Usage{Status: "unavailable", Source: "unknown"},
		EnrichmentStatus: EnrichmentUnavailable,
	}
}

func applyEnrichment(generation Generation, enrichment Enrichment) Generation {
	if enrichment.HasUsage {
		generation.Usage = enrichment.Usage
	}
	if enrichment.HasTelemetry {
		generation.Telemetry = enrichment.Telemetry
	}
	generation.ProviderEvents = enrichment.Events
	switch {
	case enrichment.HasUsage && enrichment.HasTelemetry:
		generation.EnrichmentStatus = EnrichmentAvailable
	case enrichment.HasUsage || enrichment.HasTelemetry:
		generation.EnrichmentStatus = EnrichmentPartial
	}

	return generation
}
