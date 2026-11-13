package providers

import (
	"encoding/json"
	"strings"
)

// portableStrippedPartTypes are non-replayable thinking/reasoning blobs. Do NOT
// include Codex custom_tool_* here: OpenCode Go emits and consumes them across turns.
var portableStrippedPartTypes = map[string]struct{}{
	"redacted_thinking":        {},
	"thinking":                 {},
	"reasoning":                {},
	"reasoning_summary_text":   {},
	"reasoning_text":           {},
	"encrypted_content":        {},
}

// openRouterStrippedPartTypes extends the portable set with Codex custom tool
// inventory items that OpenRouter routers reject.
var openRouterStrippedPartTypes = map[string]struct{}{
	"redacted_thinking":        {},
	"thinking":                 {},
	"reasoning":                {},
	"reasoning_summary_text":   {},
	"reasoning_text":           {},
	"encrypted_content":        {},
	"custom_tool_call":         {},
	"custom_tool_call_output":  {},
}

// ScrubNonPortableThinkingInput removes thinking blocks from a Responses input array.
// Preserves custom_tool_call / custom_tool_call_output for OpenCode continuity.
func ScrubNonPortableThinkingInput(raw json.RawMessage) (json.RawMessage, bool, error) {
	return scrubInputWithDropSet(raw, portableStrippedPartTypes)
}

// ScrubOpenRouterResponsesInput removes thinking blocks and Codex custom tool
// items that OpenRouter upstream routers reject.
func ScrubOpenRouterResponsesInput(raw json.RawMessage) (json.RawMessage, bool, error) {
	return scrubInputWithDropSet(raw, openRouterStrippedPartTypes)
}

func scrubInputWithDropSet(raw json.RawMessage, dropSet map[string]struct{}) (json.RawMessage, bool, error) {
	if len(raw) == 0 || raw[0] == '"' {
		return raw, false, nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return raw, false, nil
	}
	scrubbed, changed := scrubOpenRouterInputValue(decoded, dropSet)
	if !changed {
		return raw, false, nil
	}
	next, err := json.Marshal(scrubbed)
	if err != nil {
		return raw, false, err
	}
	return next, true, nil
}

// NormalizeOpenRouterResponsesPayload strips non-replayable thinking blocks and
// Codex custom tool history before OpenRouter upstream routers.
func NormalizeOpenRouterResponsesPayload(body []byte) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return body, nil
	}

	changed := false
	if rawInput, ok := envelope["input"]; ok {
		if scrubbed, scrubChanged, err := ScrubOpenRouterResponsesInput(rawInput); err != nil {
			return nil, err
		} else if scrubChanged {
			envelope["input"] = scrubbed
			changed = true
		}
	}
	if stripOpenCodePortableReasoningField(envelope) {
		changed = true
	}
	if stripOpenRouterIncludeField(envelope) {
		changed = true
	}
	if !changed {
		return body, nil
	}
	return json.Marshal(envelope)
}

func stripOpenRouterIncludeField(envelope map[string]json.RawMessage) bool {
	rawInclude, ok := envelope["include"]
	if !ok {
		return false
	}
	var includes []string
	if err := json.Unmarshal(rawInclude, &includes); err != nil {
		return false
	}
	filtered := make([]string, 0, len(includes))
	changed := false
	for _, item := range includes {
		trimmed := strings.TrimSpace(item)
		if strings.EqualFold(trimmed, "reasoning.encrypted_content") || strings.HasPrefix(strings.ToLower(trimmed), "reasoning.") {
			changed = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !changed {
		return false
	}
	if len(filtered) == 0 {
		delete(envelope, "include")
		return true
	}
	next, err := json.Marshal(filtered)
	if err != nil {
		return false
	}
	envelope["include"] = next
	return true
}

func scrubOpenRouterInputValue(value any, dropSet map[string]struct{}) (any, bool) {
	switch typed := value.(type) {
	case []any:
		out := make([]any, 0, len(typed))
		changed := false
		for _, item := range typed {
			next, itemChanged := scrubOpenRouterInputValue(item, dropSet)
			if next == nil && itemChanged {
				changed = true
				continue
			}
			if itemChanged {
				changed = true
			}
			if next == nil {
				continue
			}
			out = append(out, next)
		}
		if !changed {
			return value, false
		}
		return out, true
	case map[string]any:
		if shouldDropOpenRouterPart(typed, dropSet) {
			return nil, true
		}
		out := make(map[string]any, len(typed))
		changed := false
		for key, child := range typed {
			if strings.EqualFold(key, "redacted_thinking") {
				changed = true
				continue
			}
			next, childChanged := scrubOpenRouterInputValue(child, dropSet)
			if childChanged {
				changed = true
			}
			if next == nil {
				if child != nil {
					changed = true
				}
				continue
			}
			out[key] = next
		}
		if len(out) == 0 && changed {
			return nil, true
		}
		if !changed {
			return value, false
		}
		return out, true
	default:
		return value, false
	}
}

func shouldDropOpenRouterPart(item map[string]any, dropSet map[string]struct{}) bool {
	itemType, _ := item["type"].(string)
	if _, drop := dropSet[strings.TrimSpace(itemType)]; drop {
		return true
	}
	if _, hasThinking := item["redacted_thinking"]; hasThinking {
		return true
	}
	return false
}
