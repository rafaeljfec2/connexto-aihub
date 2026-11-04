package generations

import (
	"context"
	"time"

	"aihub/internal/execution"
	"aihub/internal/records"
)

const (
	EnrichmentAvailable   = "available"
	EnrichmentPartial     = "partial"
	EnrichmentUnavailable = "unavailable"
	MaxProviderEvents     = 16
)

// Usage preserves the provider-reported source and nullable measurements.
type Usage struct {
	Status               string
	Source               string
	InputTokens          *int64
	OutputTokens         *int64
	ProviderReportedCost *string
	ProviderListCost     *string
	Currency             *string
}

// Telemetry holds only measurements observed by the gateway.
type Telemetry struct {
	Streaming              *bool
	ProviderLatencyNS      *int64
	RoutingOverheadNS      *int64
	TimeToFirstTokenNS     *int64
	OutputStreamDurationNS *int64
	FinishReason           *string
	ProviderSessionID      *string
}

// ProviderEvent is an ordered, content-free gateway event.
type ProviderEvent struct {
	Sequence       int
	Type           string
	ObservedAt     time.Time
	ProviderID     string
	ModelID        string
	UpstreamStatus *int
	DurationNS     *int64
}

// Enrichment joins optional usage, timing, and observed events for one record.
type Enrichment struct {
	Usage        Usage
	Telemetry    Telemetry
	Events       []ProviderEvent
	HasUsage     bool
	HasTelemetry bool
}

// Generation combines one immutable E06 record with optional E09.5 and E10 data.
type Generation struct {
	Record           records.Record
	Usage            Usage
	Telemetry        Telemetry
	ProviderEvents   []ProviderEvent
	EnrichmentStatus string
}

// Page is one cursor-bounded generation response.
type Page struct {
	Generations []Generation
	Count       int
	NextCursor  string
	Degraded    bool
}

// TelemetryInput binds observed execution metadata to its validated gateway session.
type TelemetryInput struct {
	ExecutionID  string
	ProviderID   string
	ModelID      string
	SessionID    string
	FinishReason *string
	Result       execution.Result
}

// Repository persists content-free per-execution telemetry and events.
type Repository interface {
	Insert(context.Context, TelemetryInput) error
	List(context.Context, []string) (map[string]Enrichment, error)
	DeleteExcept(context.Context, []string) error
}
