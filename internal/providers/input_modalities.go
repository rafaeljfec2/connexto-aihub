package providers

import "strings"

// DefaultVisionAgentInputModalities is the default input modality list for agent
// models synced through minimal OpenAI-style catalogs (Connexto Proxy) when the
// upstream payload omits modalities.input.
func DefaultVisionAgentInputModalities() []string {
	return []string{"text", "image/jpeg", "image/png", "image/gif", "image/webp"}
}

// NormalizeInputModalitiesForCatalog copies upstream modality tokens and expands
// industry shorthand so E03 image_input routing matches provider intent (SPEC §5.4 A2):
//   - OpenRouter and similar APIs use bare "image" (not image/* MIME types).
//   - When bare "image" is present without any image/* entry, append image/png
//     so catalog entries remain consistent with OpenCode-style MIME lists.
func NormalizeInputModalitiesForCatalog(modalities []string) []string {
	if len(modalities) == 0 {
		return modalities
	}

	out := append([]string(nil), modalities...)
	hasImageMIME := false
	hasBareImage := false
	for _, modality := range modalities {
		switch modality {
		case "image":
			hasBareImage = true
		default:
			if strings.HasPrefix(modality, "image/") {
				hasImageMIME = true
			}
		}
	}
	if hasBareImage && !hasImageMIME {
		out = append(out, "image/png")
	}

	return out
}
