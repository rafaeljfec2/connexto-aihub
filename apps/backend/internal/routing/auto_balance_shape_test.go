package routing

import (
	"testing"

	"aihub/internal/providers"
)

func TestDeprioritizeGeminiShortCalls_PutsGeminiLast(t *testing.T) {
	gemini := Target{ProviderID: providers.AntigravityID, ModelID: "antigravity:gemini-3.8-flash"}
	sonnet := Target{ProviderID: providers.AntigravityID, ModelID: "antigravity:claude-sonnet-4-6"}
	ordered := []Target{gemini, sonnet}
	understanding := RequestUnderstanding{
		OptimizationProfile:  ProfileEfficiency,
		EstimatedInputTokens: 100,
		IsStreaming:          false,
	}
	out := deprioritizeGeminiShortCalls(ordered, understanding)
	if out[0] == gemini {
		t.Fatalf("expected gemini at tail, order=%+v", out)
	}
	if out[len(out)-1] != gemini {
		t.Fatalf("expected gemini last, order=%+v", out)
	}
}
