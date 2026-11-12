package providers

import (
	"encoding/json"
	"strings"
)

var openCodeNativeReasoningFields = []string{
	"context",
	"summary",
	"encrypted_content",
	"max_tokens",
}

// NormalizeOpenCodeGoResponsesPayload keeps Responses payloads compatible with routed
// OpenCode Go execution (non-native Responses targets).
func NormalizeOpenCodeGoResponsesPayload(body []byte) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return body, nil
	}

	changed := false
	if rawInput, ok := envelope["input"]; ok {
		if scrubbed, scrubChanged, err := ScrubNonPortableThinkingInput(rawInput); err != nil {
			return nil, err
		} else if scrubChanged {
			envelope["input"] = scrubbed
			changed = true
		}
	}
	if stripOpenCodePortableReasoningField(envelope) {
		changed = true
	}
	if stripOpenCodePortableIncludeField(envelope) {
		changed = true
	}
	if !changed {
		return body, nil
	}
	return json.Marshal(envelope)
}

func stripOpenCodePortableReasoningField(envelope map[string]json.RawMessage) bool {
	rawReasoning, ok := envelope["reasoning"]
	if !ok {
		return false
	}
	var reasoning map[string]any
	if err := json.Unmarshal(rawReasoning, &reasoning); err != nil {
		return false
	}
	fieldChanged := false
	for _, key := range openCodeNativeReasoningFields {
		if _, exists := reasoning[key]; exists {
			delete(reasoning, key)
			fieldChanged = true
		}
	}
	if !fieldChanged {
		return false
	}
	if len(reasoning) == 0 {
		delete(envelope, "reasoning")
		return true
	}
	next, err := json.Marshal(reasoning)
	if err != nil {
		return false
	}
	envelope["reasoning"] = next
	return true
}

func stripOpenCodePortableIncludeField(envelope map[string]json.RawMessage) bool {
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
		if strings.HasPrefix(strings.ToLower(trimmed), "reasoning.") {
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
