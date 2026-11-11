package routing

import "aihub/internal/providers"

// DefaultVirtualPriority defines the legacy administrative priority (shadow comparisons).
var DefaultVirtualPriority = []Target{
	{ProviderID: providers.OpenCodeGoID, ModelID: "gpt-6-luna"},
	{ProviderID: providers.OpenRouterID, ModelID: "anthropic/claude-haiku-4.5"},
	{ProviderID: providers.OpenCodeGoID, ModelID: "glm-5.3-flash"},
	{ProviderID: providers.OpenCodeGoID, ModelID: "claude-haiku-5-5"},
}

