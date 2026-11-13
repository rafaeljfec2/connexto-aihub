package gateway

import (
	"encoding/json"
	"fmt"
	"strings"
)

func chatMessagesToResponsesInput(messages []map[string]json.RawMessage) (any, error) {
	if len(messages) == 1 && rawStringField(messages[0], "role") == "user" {
		if content, ok := messages[0]["content"]; ok {
			if text, ok := jsonRawString(content); ok {
				return text, nil
			}
		}
	}
	items := make([]map[string]any, 0, len(messages))
	toolKinds := make(map[string]string)
	for _, msg := range messages {
		role := rawStringField(msg, "role")
		if role == "" {
			role = "user"
		}
		if role == "tool" {
			items = append(items, chatToolMessageToResponseItem(msg, toolKinds))
			continue
		}
		if item, ok, err := chatMessageToResponseItem(msg, role); err != nil {
			return nil, err
		} else if ok {
			items = append(items, item)
		}
		if role == "assistant" {
			calls, err := chatToolCallsToResponseItems(msg["tool_calls"], toolKinds)
			if err != nil {
				return nil, err
			}
			items = append(items, calls...)
		}
	}
	return items, nil
}

func chatMessageToResponseItem(msg map[string]json.RawMessage, role string) (map[string]any, bool, error) {
	content, ok := msg["content"]
	if !ok || bytesEqualNull(content) {
		return nil, false, nil
	}
	normalized, err := normalizeChatMessageContent(content, role)
	if err != nil {
		return nil, false, err
	}
	return map[string]any{"role": role, "content": normalized}, true, nil
}

func chatToolMessageToResponseItem(msg map[string]json.RawMessage, kinds map[string]string) map[string]any {
	callID := rawStringField(msg, "tool_call_id")
	output, ok := jsonRawString(msg["content"])
	if !ok {
		output = ""
	}
	itemType := "function_call_output"
	if kinds[callID] == "custom_tool_call" {
		itemType = "custom_tool_call_output"
	}
	return map[string]any{"type": itemType, "call_id": callID, "output": output}
}

func chatToolCallsToResponseItems(raw json.RawMessage, kinds map[string]string) ([]map[string]any, error) {
	if len(raw) == 0 || bytesEqualNull(raw) {
		return nil, nil
	}
	var calls []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &calls); err != nil {
		return nil, fmt.Errorf("tool_calls is invalid")
	}
	items := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		item, err := chatToolCallToResponseItem(call)
		if err != nil {
			return nil, err
		}
		callID, _ := item["call_id"].(string)
		itemType, _ := item["type"].(string)
		kinds[callID] = itemType
		items = append(items, item)
	}
	return items, nil
}

func chatToolCallToResponseItem(call map[string]json.RawMessage) (map[string]any, error) {
	callID := rawStringField(call, "id")
	if callID == "" {
		return nil, fmt.Errorf("tool call id is required")
	}
	callType := rawStringField(call, "type")
	fn := call
	if nested := call["function"]; len(nested) > 0 {
		if err := json.Unmarshal(nested, &fn); err != nil {
			return nil, fmt.Errorf("tool call function is invalid")
		}
	}
	name := rawStringField(fn, "name")
	if name == "" {
		return nil, fmt.Errorf("tool call name is required")
	}
	item := map[string]any{"call_id": callID, "name": name}
	if callType == "custom" {
		input := rawStringField(fn, "arguments")
		if input == "" {
			input = rawStringField(fn, "input")
		}
		item["type"] = "custom_tool_call"
		item["input"] = input
		return item, nil
	}
	arguments := rawStringField(fn, "arguments")
	if arguments == "" {
		arguments = "{}"
	}
	item["type"] = "function_call"
	item["arguments"] = arguments
	return item, nil
}

func normalizeChatMessageContent(content json.RawMessage, role string) (any, error) {
	if text, ok := jsonRawString(content); ok {
		return []map[string]any{{"type": responseContentPartType(role), "text": text}}, nil
	}
	var parts []json.RawMessage
	if err := json.Unmarshal(content, &parts); err != nil {
		return nil, fmt.Errorf("message content is invalid")
	}
	out := make([]map[string]any, 0, len(parts))
	for _, raw := range parts {
		var part map[string]any
		if err := json.Unmarshal(raw, &part); err != nil {
			return nil, fmt.Errorf("message content is invalid")
		}
		normalized, err := normalizeChatContentPart(part, role)
		if err != nil {
			return nil, err
		}
		if normalized != nil {
			out = append(out, normalized)
		}
	}
	return out, nil
}

func normalizeChatContentPart(part map[string]any, role string) (map[string]any, error) {
	rawType, _ := part["type"].(string)
	switch rawType {
	case "text", "input_text", "output_text":
		text, _ := part["text"].(string)
		return map[string]any{"type": responseContentPartType(role), "text": text}, nil
	case "image_url":
		url := imageURLFromChatPart(part)
		if url == "" {
			return nil, nil
		}
		return map[string]any{"type": "input_image", "image_url": url}, nil
	case "input_image", "input_audio", "input_file", "refusal", "summary_text", "encrypted_content", "computer_screenshot":
		return part, nil
	default:
		if text, ok := part["text"].(string); ok && rawType == "" {
			return map[string]any{"type": responseContentPartType(role), "text": text}, nil
		}
		return part, nil
	}
}

func responseContentPartType(role string) string {
	if role == "assistant" {
		return "output_text"
	}
	return "input_text"
}

func imageURLFromChatPart(part map[string]any) string {
	switch image := part["image_url"].(type) {
	case string:
		return image
	case map[string]any:
		url, _ := image["url"].(string)
		return url
	}
	return ""
}

func rawStringField(values map[string]json.RawMessage, key string) string {
	value, ok := values[key]
	if !ok {
		return ""
	}
	text, _ := jsonRawString(value)
	return text
}

func jsonRawString(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}

func bytesEqualNull(raw json.RawMessage) bool {
	return strings.TrimSpace(string(raw)) == "null"
}
