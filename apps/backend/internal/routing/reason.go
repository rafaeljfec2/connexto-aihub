package routing

// ReasonCode is a stable eligibility exclusion identifier from the E08 contract.
type ReasonCode string

// Contract reason codes for provider and model eligibility.
const (
	ReasonProviderUnknown        ReasonCode = "provider_unknown"
	ReasonProviderDisabled       ReasonCode = "provider_disabled"
	ReasonCredentialMissing      ReasonCode = "credential_missing"
	ReasonProviderUnbound        ReasonCode = "provider_unbound"
	ReasonResponsesUnsupported   ReasonCode = "responses_unsupported"
	ReasonProviderUnavailable    ReasonCode = "provider_unavailable"
	ReasonModelNotFound          ReasonCode = "model_not_found"
	ReasonModelMissing           ReasonCode = "model_missing"
	ReasonCatalogStale           ReasonCode = "catalog_stale"
	ReasonCapabilityUnsupported  ReasonCode = "capability_unsupported"
	ReasonCapabilityUnknown      ReasonCode = "capability_unknown"
	ReasonLimitExceeded          ReasonCode = "limit_exceeded"
	ReasonLimitUnknown           ReasonCode = "limit_unknown"
	ReasonRequirementUnmapped    ReasonCode = "requirement_unmapped"
)

// StructuralErrorCode identifies malformed routing input.
type StructuralErrorCode string

const (
	StructuralInvalidRequest StructuralErrorCode = "invalid_request"
)
