package routing

// SelectionOrigin identifies why a target was chosen (contract §4.1).
type SelectionOrigin string

const (
	OriginExplicitProviderAndModel SelectionOrigin = "explicit_provider_and_model"
	OriginExplicitProvider         SelectionOrigin = "explicit_provider"
	OriginExplicitModel            SelectionOrigin = "explicit_model"
	OriginCompatibilityDefault     SelectionOrigin = "compatibility_default"
	OriginAutomaticUniqueCandidate SelectionOrigin = "automatic_unique_candidate"
	OriginVirtualAuto              SelectionOrigin = "virtual_auto"
)

// RejectionCode is a stable routing decision failure identifier (contract §6.3).
type RejectionCode string

const (
	RejectRequirementUnmapped   RejectionCode = "requirement_unmapped"
	RejectRoutingUnavailable    RejectionCode = "routing_unavailable"
	RejectRoutingAmbiguous      RejectionCode = "routing_ambiguous"
	RejectModelNotFound         RejectionCode = "model_not_found"
	RejectProviderModelConflict RejectionCode = "provider_model_conflict"
	RejectNoEligibleModel       RejectionCode = "no_eligible_model"
	RejectProviderNotFound      RejectionCode = "provider_not_found"
)

// RoutingDecision is the E08.3 output consumed by gateway integration (E08.4).
type RoutingDecision struct {
	RequestID   string
	Selected    *Target
	Origin      SelectionOrigin
	Criteria    []string
	Rejection   RejectionCode
	Eligibility EligibilityResult
}
