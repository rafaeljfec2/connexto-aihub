package routing

import "aihub/internal/providers"

// AutoMVPAllowlist is the exclusive model set for aihub/auto (discovery §B).
var AutoMVPAllowlist = []Target{
	{ProviderID: providers.OpenRouterID, ModelID: "anthropic/claude-haiku-4.5"},
	{ProviderID: providers.OpenCodeGoID, ModelID: "glm-5.3-flash"},
	{ProviderID: providers.OpenCodeGoID, ModelID: "deepseek-v4.1-flash"},
	{ProviderID: providers.OpenCodeGoID, ModelID: "claude-haiku-5-5"},
	{ProviderID: providers.OpenCodeGoID, ModelID: "gpt-6-luna"},
	{ProviderID: providers.CursorID, ModelID: "composer-2.5"},
	{ProviderID: providers.AntigravityID, ModelID: "antigravity:claude-opus-4-6-thinking"},
	{ProviderID: providers.AntigravityID, ModelID: "antigravity:claude-sonnet-4-6"},
	{ProviderID: providers.AntigravityID, ModelID: "antigravity:gemini-3.8-flash"},
	{ProviderID: providers.AntigravityID, ModelID: "antigravity:gpt-oss-120b-medium"},
}

// AutoAdminTieOrder is the full deterministic order inside Band A/B (discovery §G).
var AutoAdminTieOrder = []Target{
	{ProviderID: providers.OpenCodeGoID, ModelID: "gpt-6-luna"},
	{ProviderID: providers.OpenRouterID, ModelID: "anthropic/claude-haiku-4.5"},
	{ProviderID: providers.OpenCodeGoID, ModelID: "glm-5.3-flash"},
	{ProviderID: providers.OpenCodeGoID, ModelID: "deepseek-v4.1-flash"},
	{ProviderID: providers.OpenCodeGoID, ModelID: "claude-haiku-5-5"},
	{ProviderID: providers.CursorID, ModelID: "composer-2.5"},
	{ProviderID: providers.AntigravityID, ModelID: "antigravity:gemini-3.8-flash"},
	{ProviderID: providers.AntigravityID, ModelID: "antigravity:claude-sonnet-4-6"},
	{ProviderID: providers.AntigravityID, ModelID: "antigravity:claude-opus-4-6-thinking"},
	{ProviderID: providers.AntigravityID, ModelID: "antigravity:gpt-oss-120b-medium"},
}

// IsAutoAllowlistTarget reports whether t is one of the ten MVP ids.
func IsAutoAllowlistTarget(t Target) bool {
	for _, entry := range AutoMVPAllowlist {
		if entry == t {
			return true
		}
	}
	return false
}

// FilterEligibleToAutoAllowlist keeps only allowlisted pairs (auto path only).
func FilterEligibleToAutoAllowlist(eligible []Target) []Target {
	if len(eligible) == 0 {
		return nil
	}
	out := make([]Target, 0, len(eligible))
	for _, t := range eligible {
		if IsAutoAllowlistTarget(t) {
			out = append(out, t)
		}
	}
	return out
}

// FilterEligibleToAutoAllowlistForSnapshot keeps allowlisted pairs matching either exact
// ProviderID or the account's ProviderType from snapshot.Admin.
func FilterEligibleToAutoAllowlistForSnapshot(eligible []Target, snapshot Snapshot) []Target {
	if len(eligible) == 0 {
		return nil
	}
	out := make([]Target, 0, len(eligible))
	for _, t := range eligible {
		pType := providerTypeOf(snapshot, t.ProviderID)
		if IsAutoAllowlistTarget(t) || IsAutoAllowlistTarget(Target{ProviderID: pType, ModelID: t.ModelID}) {
			out = append(out, t)
		}
	}
	return out
}
