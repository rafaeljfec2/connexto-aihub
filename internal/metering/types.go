// Package metering captures provider-reported execution usage (E09.5).
package metering

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"aihub/internal/metering/capture"
)

// Status classifies how completely provider usage was captured.
type Status string

const (
	StatusReported    Status = "reported"
	StatusUnavailable Status = "unavailable"
	StatusPartial     Status = "partial"
	StatusInvalid     Status = "invalid"
)

// Source identifies which adapter contract produced the normalized record.
type Source string

const (
	SourceOpenCodeGo Source = "provider_opencode_go"
	SourceOpenRouter Source = "provider_openrouter"
	SourceUnknown    Source = "provider_unknown"
)

// Record is the normalized, content-free usage row for one execution.
type Record struct {
	ExecutionID          string
	ProviderID           string
	ModelID              string
	InputTokens          *int64
	OutputTokens         *int64
	TotalTokens          *int64
	CachedInputTokens    *int64
	ProviderReportedCost pgtype.Numeric
	ProviderListCost     pgtype.Numeric
	Currency             string
	Status               Status
	Source               Source
	RecordedAt           time.Time
}

// RecordInput binds one finished gateway execution to captured provider bytes.
type RecordInput struct {
	ExecutionID string
	ProviderID  string
	ModelID     string
	Capture     capture.Result
}
