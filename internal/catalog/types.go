// Package catalog owns the model catalog domain: provider-scoped model
// identity, capability claims, listing semantics and the sync/persistence
// lifecycle.
package catalog

import (
	"encoding/json"
	"time"

	"aihub/internal/providers"
)

// CapabilityState is the honest state of a normalized capability.
type CapabilityState string

// Capability states.
const (
	CapabilitySupported   CapabilityState = "supported"
	CapabilityUnsupported CapabilityState = "unsupported"
	CapabilityUnknown     CapabilityState = "unknown"
)

// CapabilityProvenance records where a capability claim came from.
type CapabilityProvenance string

// Provenance values.
const (
	ProvenanceProviderCatalog CapabilityProvenance = "provider_catalog"
)

// CapabilityConfidence is the qualitative basis of a capability claim.
type CapabilityConfidence string

// Confidence values.
const (
	ConfidenceExplicit CapabilityConfidence = "explicit"
	ConfidenceDerived  CapabilityConfidence = "derived"
	ConfidenceNone     CapabilityConfidence = "none"
)

// VerificationState records whether execution evidence exists.
type VerificationState string

// Verification values.
const (
	VerificationUnverified VerificationState = "unverified"
)

// CapabilityClaim is one normalized capability with provenance and confidence.
type CapabilityClaim struct {
	State        CapabilityState      `json:"state"`
	Provenance   CapabilityProvenance `json:"provenance"`
	Confidence   CapabilityConfidence `json:"confidence"`
	Verification VerificationState    `json:"verification"`
}

// Modalities preserves the provider modality declarations. A nil Modalities on
// a ModelEntry means the provider omitted the field.
type Modalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

// Limits preserves optional token limits.
type Limits struct {
	Context *int64 `json:"context,omitempty"`
	Input   *int64 `json:"input,omitempty"`
	Output  *int64 `json:"output,omitempty"`
}

// ModelKey is the provider-scoped identity of a catalog entry.
type ModelKey struct {
	ProviderID string
	ModelID    string
}

// ModelEntry is one provider-scoped model. Identity is the pair
// (ProviderID, ModelID).
type ModelEntry struct {
	ProviderID       string                     `json:"-"`
	ModelID          string                     `json:"model_id"`
	CanonicalID      string                     `json:"canonical_id,omitempty"`
	Name             string                     `json:"name"`
	Type             string                     `json:"type,omitempty"`
	Family           string                     `json:"family,omitempty"`
	Modalities       *Modalities                `json:"modalities,omitempty"`
	Limits           Limits                     `json:"limits"`
	Capabilities     map[string]CapabilityClaim `json:"capabilities"`
	ProviderMetadata json.RawMessage            `json:"provider_metadata,omitempty"`
	FirstSeenAt      time.Time                  `json:"first_seen_at"`
	LastSeenAt       time.Time                  `json:"last_seen_at"`
	MissingSince     *time.Time                 `json:"missing_since,omitempty"`
}

// SyncOutcome is the result of the latest synchronization attempt.
type SyncOutcome string

// Sync outcomes.
const (
	SyncOutcomeNever         SyncOutcome = "never"
	SyncOutcomeSuccess       SyncOutcome = "success"
	SyncOutcomeRejectedEmpty SyncOutcome = "rejected_empty"
	SyncOutcomeFailed        SyncOutcome = "failed"
	SyncOutcomeSkipped       SyncOutcome = "skipped"
)

// SyncErrorClass classifies the latest synchronization attempt.
type SyncErrorClass string

// Sync error classes.
const (
	SyncErrorNone              SyncErrorClass = "none"
	SyncErrorProviderDisabled  SyncErrorClass = "provider_disabled"
	SyncErrorProviderUnhealthy SyncErrorClass = "provider_unhealthy"
	SyncErrorTransport         SyncErrorClass = "transport"
	SyncErrorTimeout           SyncErrorClass = "timeout"
	SyncErrorHTTPStatus        SyncErrorClass = "http_status"
	SyncErrorInvalidPayload    SyncErrorClass = "invalid_payload"
	SyncErrorEmptyCatalog      SyncErrorClass = "empty_catalog"
	SyncErrorPersistence       SyncErrorClass = "persistence"
)

// SyncCounts describes the latest accepted synchronization result.
type SyncCounts struct {
	Discovered int `json:"discovered"`
	Added      int `json:"added"`
	Updated    int `json:"updated"`
	Unchanged  int `json:"unchanged"`
	Missing    int `json:"missing"`
}

// ProviderSyncState is the operational and accepted synchronization state of a
// provider.
type ProviderSyncState struct {
	LastAttemptAt  *time.Time     `json:"last_attempt_at,omitempty"`
	LastSuccessAt  *time.Time     `json:"last_success_at,omitempty"`
	Outcome        SyncOutcome    `json:"outcome"`
	ErrorClass     SyncErrorClass `json:"error_class"`
	UpstreamStatus int            `json:"upstream_status"`
	Counts         SyncCounts     `json:"counts"`
}

// Presence is whether the model appears in the last accepted catalog.
type Presence string

// Presence values.
const (
	PresencePresent Presence = "present"
	PresenceMissing Presence = "missing"
	PresenceUnknown Presence = "unknown"
)

// Freshness is the age of the last accepted synchronization.
type Freshness string

// Freshness values.
const (
	FreshnessFresh Freshness = "fresh"
	FreshnessStale Freshness = "stale"
	FreshnessNever Freshness = "never"
)

// ListingState is the derived catalog listing condition.
type ListingState string

// Listing states.
const (
	ListingCurrent ListingState = "current"
	ListingStale   ListingState = "stale"
	ListingMissing ListingState = "missing"
	ListingUnknown ListingState = "unknown"
)

// ListingReason explains a derived listing state.
type ListingReason string

// Listing reasons.
const (
	ReasonNeverSynced           ListingReason = "never_synced"
	ReasonModelMissing          ListingReason = "model_missing"
	ReasonCatalogStale          ListingReason = "catalog_stale"
	ReasonProviderDisabled      ListingReason = "provider_disabled"
	ReasonProviderUnhealthy     ListingReason = "provider_unhealthy"
	ReasonProviderStateUnknown  ListingReason = "provider_state_unknown"
	ReasonLastSyncFailed        ListingReason = "last_sync_failed"
	ReasonLastSyncRejectedEmpty ListingReason = "last_sync_rejected_empty"
)

// View is the derived, read-time projection of a model and its provider state.
type View struct {
	Entry                 ModelEntry
	Presence              Presence
	Freshness             Freshness
	ProviderState         providers.State
	ExecutionVerification VerificationState
	ListingState          ListingState
	ListingReasons        []ListingReason
}

// ProviderView is the derived, read-time projection of one provider.
type ProviderView struct {
	ProviderID    string
	ProviderState providers.State
	Freshness     Freshness
	Sync          ProviderSyncState
	ModelsKnown   int
	ModelsPresent int
	ModelsMissing int
}
