package routing

import (
	"strings"

	"aihub/internal/providers"
)

// IsCursorProxyModelID reports whether the model id belongs to the Connexto Proxy
// Cursor catalog (Composer, Grok). These ids are not valid on OpenCode Go or OpenRouter.
func IsCursorProxyModelID(modelID string) bool {
	normalized := strings.TrimSpace(modelID)
	if normalized == "" {
		return false
	}
	switch normalized {
	case "composer-2.5", "composer-2.5-fast",
		"grok-4.6", "grok-4.6-fast",
		"grok-4.5", "grok-4.5-fast":
		return true
	default:
		lower := strings.ToLower(normalized)
		return strings.HasPrefix(lower, "composer-") || strings.HasPrefix(lower, "grok-")
	}
}

func firstCursorProviderID(snapshot Snapshot) string {
	var count int
	var only string
	for id := range snapshot.Admin {
		if providerTypeOf(snapshot, id) == providers.TypeCursor {
			count++
			only = id
		}
	}
	if count == 1 {
		return only
	}
	if count > 1 {
		return ""
	}
	return providers.CursorID
}
