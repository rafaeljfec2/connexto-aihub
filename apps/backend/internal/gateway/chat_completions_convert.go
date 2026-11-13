package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var errChatModelRequired = errors.New("model is required")
var errChatMessagesRequired = errors.New("messages is required")

type chatCompletionRequest struct {
	Model       string          `json:"model"`
	Messages    json.RawMessage `json:"messages"`
	Stream      *bool           `json:"stream"`
	Tools       json.RawMessage `json:"tools,omitempty"`
	ToolChoice  json.RawMessage `json:"tool_choice,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
}

// chatRequestToResponses maps a Chat Completions body to the Responses shape.
func chatRequestToResponses(raw []byte) ([]byte, string, bool, error) {
	var chat chatCompletionRequest
	if err := json.Unmarshal(raw, &chat); err != nil {
		return nil, "", false, fmt.Errorf("invalid JSON: %w", err)
	}
	clientModel := strings.TrimSpace(chat.Model)
	if clientModel == "" {
		return nil, "", false, errChatModelRequired
	}
	var messages []map[string]json.RawMessage
	if len(chat.Messages) == 0 || json.Unmarshal(chat.Messages, &messages) != nil || len(messages) == 0 {
		return nil, "", false, errChatMessagesRequired
	}
	input, err := chatMessagesToResponsesInput(messages)
	if err != nil {
		return nil, "", false, err
	}
	envelope := map[string]any{"model": clientModel, "input": input}
	stream := false
	if chat.Stream != nil {
		stream = *chat.Stream
		envelope["stream"] = stream
	}
	if tools, present, err := chatToolsToResponses(chat.Tools); err != nil {
		return nil, "", false, err
	} else if present {
		envelope["tools"] = tools
	}
	if choice, present, err := chatToolChoiceToResponses(chat.ToolChoice); err != nil {
		return nil, "", false, err
	} else if present {
		envelope["tool_choice"] = choice
	}
	copyChatOptions(envelope, chat)
	out, err := json.Marshal(envelope)
	return out, clientModel, stream, err
}

func copyChatOptions(envelope map[string]any, chat chatCompletionRequest) {
	if chat.Temperature != nil {
		envelope["temperature"] = *chat.Temperature
	}
	if chat.TopP != nil {
		envelope["top_p"] = *chat.TopP
	}
	if chat.MaxTokens != nil {
		envelope["max_output_tokens"] = *chat.MaxTokens
	}
}

func chatToolsToResponses(raw json.RawMessage) ([]map[string]any, bool, error) {
	if len(raw) == 0 || bytesEqualNull(raw) {
		return nil, false, nil
	}
	var tools []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &tools); err != nil {
		return nil, false, fmt.Errorf("tools is invalid")
	}
	out := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		fn := tool
		if nested := tool["function"]; len(nested) > 0 {
			if err := json.Unmarshal(nested, &fn); err != nil {
				return nil, false, fmt.Errorf("tool function is invalid")
			}
		}
		name := rawStringField(fn, "name")
		if name == "" {
			return nil, false, fmt.Errorf("tool name is required")
		}
		normalized := map[string]any{"type": "function", "name": name}
		for _, key := range []string{"description", "parameters", "strict"} {
			if value, ok := fn[key]; ok {
				var decoded any
				if err := json.Unmarshal(value, &decoded); err != nil {
					return nil, false, fmt.Errorf("tool %s is invalid", key)
				}
				normalized[key] = decoded
			}
		}
		out = append(out, normalized)
	}
	return out, true, nil
}

func chatToolChoiceToResponses(raw json.RawMessage) (any, bool, error) {
	if len(raw) == 0 || bytesEqualNull(raw) {
		return nil, false, nil
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text, true, nil
	}
	var choice map[string]json.RawMessage
	if err := json.Unmarshal(raw, &choice); err != nil {
		return nil, false, fmt.Errorf("tool_choice is invalid")
	}
	fn := choice
	if nested := choice["function"]; len(nested) > 0 {
		if err := json.Unmarshal(nested, &fn); err != nil {
			return nil, false, fmt.Errorf("tool_choice function is invalid")
		}
	}
	name := rawStringField(fn, "name")
	if name == "" {
		name = rawStringField(choice, "name")
	}
	if name == "" {
		return choice, true, nil
	}
	return map[string]any{"type": "function", "name": name}, true, nil
}
