package providers

import (
	"encoding/json"
	"strings"
)

var openRouterStrippedPartTypes = map[string]struct{}{
	"redacted_thinking":        {},
	"thinking":               {},
	"reasoning":              {},
	"reasoning_summary_text": {},
	"reasoning_text":         {},
	"encrypted_content":      {},
}

// NormalizeOpenRouterResponsesPayload strips non-replayable thinking blocks from
// Responses bodies before they reach OpenRouter upstream routers.
func NormalizeOpenRouterResponsesPayload(body []byte) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return body, nil
	}

	inputChanged := false
	if rawInput, ok := envelope["input"]; ok && len(rawInput) > 0 && rawInput[0] != '"' {
		var decoded any
		if err := json.Unmarshal(rawInput, &decoded); err == nil {
			scrubbed, changed := scrubOpenRouterInputValue(decoded)
			if changed {
				next, err := json.Marshal(scrubbed)
				if err != nil {
					return nil, err
				}
				envelope["input"] = next
				inputChanged = true
			}
		}
	}
	includeChanged := stripOpenRouterIncludeField(envelope)
	if !inputChanged && !includeChanged {
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

func scrubOpenRouterInputValue(value any) (any, bool) {
	switch typed := value.(type) {
	case []any:
		out := make([]any, 0, len(typed))
		changed := false
		for _, item := range typed {
			next, itemChanged := scrubOpenRouterInputValue(item)
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
		if shouldDropOpenRouterPart(typed) {
			return nil, true
		}
		out := make(map[string]any, len(typed))
		changed := false
		for key, child := range typed {
			if strings.EqualFold(key, "redacted_thinking") {
				changed = true
				continue
			}
			next, childChanged := scrubOpenRouterInputValue(child)
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

func shouldDropOpenRouterPart(item map[string]any) bool {
	itemType, _ := item["type"].(string)
	if _, drop := openRouterStrippedPartTypes[strings.TrimSpace(itemType)]; drop {
		return true
	}
	if _, hasThinking := item["redacted_thinking"]; hasThinking {
		return true
	}
	return false
}
