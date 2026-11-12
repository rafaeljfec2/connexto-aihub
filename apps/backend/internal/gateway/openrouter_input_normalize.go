package gateway

import "aihub/internal/providers"

// normalizeOpenRouterResponsesInput removes non-replayable thinking blocks before
// OpenRouter execution.
func normalizeOpenRouterResponsesInput(body []byte) ([]byte, error) {
	return providers.NormalizeOpenRouterResponsesPayload(body)
}

// OpenRouterResponsesPayloadForUpstream applies OpenRouter input shaping for tests.
func OpenRouterResponsesPayloadForUpstream(body []byte) ([]byte, error) {
	return normalizeOpenRouterResponsesInput(body)
}
