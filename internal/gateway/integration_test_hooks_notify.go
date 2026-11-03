package gateway

func notifyIntegrationRoutingSelected(providerID, modelID string) {
	if IntegrationTestHooks.OnRoutingSelected != nil {
		IntegrationTestHooks.OnRoutingSelected(providerID, modelID)
	}
}
