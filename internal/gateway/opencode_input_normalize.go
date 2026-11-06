package gateway

import (
	"encoding/json"
	"strings"

	"aihub/internal/providers"
)

// adaptResponsesPayloadForProvider applies provider-specific request shaping before
// upstream execution. OpenCode Go rejects top-level input_text/input_image items;
// they must appear inside type=message (or role+content) entries.
func adaptResponsesPayloadForProvider(providerID string, body []byte) ([]byte, error) {
	if providerID != providers.OpenCodeGoID {
		return body, nil
	}
	return normalizeOpenCodeGoResponsesInput(body)
}

func normalizeOpenCodeGoResponsesInput(body []byte) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return body, nil
	}

	rawInput, ok := envelope["input"]
	if !ok {
		return body, nil
	}

	if rawInput[0] == '"' {
		return body, nil
	}

	var items []json.RawMessage
	if err := json.Unmarshal(rawInput, &items); err != nil {
		return body, nil
	}

	normalized, changed, err := normalizeOpenCodeInputItems(items)
	if err != nil {
		return nil, err
	}
	if !changed {
		return body, nil
	}

	normalizedRaw, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	envelope["input"] = normalizedRaw
	return json.Marshal(envelope)
}

func normalizeOpenCodeInputItems(items []json.RawMessage) ([]json.RawMessage, bool, error) {
	if len(items) == 0 {
		return items, false, nil
	}

	out := make([]json.RawMessage, 0, len(items))
	changed := false
	var pending []map[string]any

	flushPending := func() error {
		if len(pending) == 0 {
			return nil
		}
		message := map[string]any{
			"type":    "message",
			"role":    "user",
			"content": pending,
		}
		raw, err := json.Marshal(message)
		if err != nil {
			return err
		}
		out = append(out, raw)
		pending = nil
		changed = true
		return nil
	}

	for _, raw := range items {
		var item map[string]any
		if err := json.Unmarshal(raw, &item); err != nil {
			if err := flushPending(); err != nil {
				return nil, false, err
			}
			out = append(out, raw)
			continue
		}

		itemType, _ := item["type"].(string)
		if isOpenCodeFlatInputPart(itemType) {
			pending = append(pending, item)
			continue
		}

		if err := flushPending(); err != nil {
			return nil, false, err
		}

		if role, hasRole := item["role"].(string); hasRole && strings.TrimSpace(role) != "" {
			if _, hasType := item["type"]; !hasType {
				item["type"] = "message"
				raw, err := json.Marshal(item)
				if err != nil {
					return nil, false, err
				}
				out = append(out, raw)
				changed = true
				continue
			}
		}

		out = append(out, raw)
	}

	if err := flushPending(); err != nil {
		return nil, false, err
	}

	return out, changed, nil
}

// OpenCodeResponsesPayloadForUpstream applies OpenCode Go input shaping. Tests use
// this to assert the E02 contract: fields other than normalized input stay verbatim.
func OpenCodeResponsesPayloadForUpstream(body []byte) ([]byte, error) {
	return adaptResponsesPayloadForProvider(providers.OpenCodeGoID, body)
}

func isOpenCodeFlatInputPart(itemType string) bool {
	switch itemType {
	case "input_text", "input_image", "input_audio", "input_file":
		return true
	default:
		return false
	}
}
