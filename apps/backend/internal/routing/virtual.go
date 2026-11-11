package routing

// Virtual model identifiers accepted by the responses gateway.
const (
	VirtualModelAuto           = "aihub/auto"
	VirtualModelOpenRouterAuto = "openrouter/auto"
)

// HTTP transparency header names injected before response commit.
const (
	HeaderResolvedModel    = "X-AIHub-Resolved-Model"
	HeaderResolvedProvider = "X-AIHub-Resolved-Provider"
	HeaderRoutingProfile   = "X-AIHub-Routing-Profile"
	HeaderRoutingReason    = "X-AIHub-Routing-Reason"
)

// IsVirtualModel returns true if the model identifier represents a virtual routing intent.
func IsVirtualModel(modelID string) bool {
	return modelID == VirtualModelAuto || modelID == VirtualModelOpenRouterAuto
}

