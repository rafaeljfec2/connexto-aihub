package generations

import (
	"context"
	"time"

	"aihub/internal/records"
)

// RetentionConfig aligns telemetry cleanup with execution-record cleanup.
type RetentionConfig struct {
	Interval time.Duration
}

// Name identifies the telemetry retention module in the application lifecycle.
func (s *Service) Name() string {
	return "generation-telemetry"
}

// Start cleans telemetry left by record retention, then keeps cleanup current.
func (s *Service) Start(context.Context) error {
	if s.repository == nil || s.retention.Interval <= 0 {
		return nil
	}
	s.prune()
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelRetention = cancel
	s.retentionDone = make(chan struct{})
	go s.runRetention(ctx)
	return nil
}

// Stop waits for the bounded retention loop to exit before the database closes.
func (s *Service) Stop(context.Context) error {
	if s.cancelRetention == nil {
		return nil
	}
	s.cancelRetention()
	<-s.retentionDone
	s.cancelRetention = nil
	return nil
}

func (s *Service) runRetention(ctx context.Context) {
	defer close(s.retentionDone)
	ticker := time.NewTicker(s.retention.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.prune()
		}
	}
}

func (s *Service) prune() {
	ctx, cancel := context.WithTimeout(context.Background(), persistenceTimeout)
	defer cancel()
	retained, err := s.records.ListAll(records.Filter{Limit: records.MaxPageLimit})
	if err != nil {
		s.logEnrichmentFailure("generation_telemetry_retention_read_failed")
		return
	}
	ids := executionIDs(retained)
	if err := s.repository.DeleteExcept(ctx, ids); err != nil {
		s.logEnrichmentFailure("generation_telemetry_retention_delete_failed")
	}
}
