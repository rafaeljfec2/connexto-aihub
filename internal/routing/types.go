package routing

import (
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
)

// Target is one provider-scoped model identity (ADR-009).
type Target struct {
	ProviderID string
	ModelID    string
}

// ExclusionClass distinguishes objective incompatibility from insufficient evidence.
type ExclusionClass string

const (
	ExclusionIncompatible   ExclusionClass = "incompatible"
	ExclusionInsufficient   ExclusionClass = "insufficient"
	ExclusionNotApplicable  ExclusionClass = "not_applicable"
)

// CandidateEvaluation records one evaluated provider/model pair.
type CandidateEvaluation struct {
	Target         Target
	Eligible       bool
	Reasons        []ReasonCode
	ExclusionClass ExclusionClass
}

// ProviderCoverage reports catalog evidence for one provider relevant to routing.
type ProviderCoverage struct {
	ProviderID           string
	PassesProviderChecks bool
	CatalogFreshAccepted bool
	SyncOutcome          catalog.SyncOutcome
}

// CoverageEvidence summarizes whether catalog facts are complete enough for uniqueness conclusions.
type CoverageEvidence struct {
	Providers                        []ProviderCoverage
	CompleteFreshCoverageForEligible bool
}

// RequirementStatus describes structural capability projection for the request.
type RequirementStatus struct {
	RequiredCapabilities []string
	Unmapped             bool
	MaxOutputTokens      *int64
}

// EligibilityResult is the E08.2 output consumed by SelectDeterministic (E08.3).
// It never selects a winner among multiple eligible candidates.
type EligibilityResult struct {
	RequestID          string
	Evaluated          []CandidateEvaluation
	Eligible           []Target
	Coverage           CoverageEvidence
	Requirements       RequirementStatus
	RequirementUnmapped bool
}

// RoutingContext carries explicit routing constraints from the request boundary.
type RoutingContext struct {
	RequestID         string
	ExplicitProvider  string
	ExplicitModel     string
	Structural        StructuralRequest
	Understanding     RequestUnderstanding
}

// StructuralRequest is the validated structural subset of a Responses body.
type StructuralRequest struct {
	ModelPresent      bool
	ModelID           string
	MaxOutputTokens   *int64
}

// AdminProviderFacts are configuration facts without credential material.
type AdminProviderFacts struct {
	ID            string
	Known         bool
	Enabled       bool
	HasCredential bool
}

// RuntimeProviderFacts describe registry binding and execution gate.
type RuntimeProviderFacts struct {
	Bound               bool
	ResponsesSupported  bool
	RuntimeEnabled      bool
	Health              providers.State
}

// Snapshot is a coherent routing snapshot (contract §2.4).
type Snapshot struct {
	CapturedAt time.Time
	StaleAfter   time.Duration
	Admin        map[string]AdminProviderFacts
	Runtime      map[string]RuntimeProviderFacts
	Models       []catalog.View
	Sync         map[string]catalog.ProviderSyncState
	AutoQuota    AutoQuotaView
}
