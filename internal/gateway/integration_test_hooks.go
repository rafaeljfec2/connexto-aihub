package gateway

// IntegrationTestHooks exposes optional callbacks for end-to-end integration
// tests. Production code leaves them unset; gateway tests assign them for the
// duration of a case.
var IntegrationTestHooks struct {
	OnRoutingSelected func(providerID, modelID string)
	OnResourceAdmissionStarted func(providerID, modelID string)
	OnResourceAdmissionAccepted func(providerID, modelID string)
	OnResourceAdmissionRejected func(providerID, modelID string)
	OnResourceAdmissionFailed func(providerID, modelID string)
	OnConcurrencyAcquireStarted func(executionID, providerID, modelID string)
	OnConcurrencyAcquireAccepted func(executionID, providerID, modelID string)
	OnConcurrencyAcquireRejected func(executionID, providerID, modelID string)
	OnConcurrencyAcquireFailed func(executionID, providerID, modelID string)
	OnExecutionStarted func(providerID, modelID string)
	OnProviderCalled func(providerID, modelID string)
}
