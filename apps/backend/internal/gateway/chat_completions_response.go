package gateway

import (
	"encoding/json"
	"strings"
	"time"
)

type chatResponseOutputItem struct {
	Type      string `json:"type"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Input     string `json:"input"`
	Content   []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type chatResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// responsesBodyToChatCompletion converts a non-streaming Responses JSON body to chat.completion.
func responsesBodyToChatCompletion(body []byte, clientModel string) ([]byte, error) {
	var parsed struct {
		ID      string                   `json:"id"`
		Model   string                   `json:"model"`
		Created any                      `json:"created_at"`
		Output  []chatResponseOutputItem `json:"output"`
		Usage   *chatResponseUsage       `json:"usage"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	model := clientModel
	if model == "" {
		model = parsed.Model
	}
	text, toolCalls := responseOutputToChat(parsed.Output)
	created := time.Now().Unix()
	if timestamp, ok := jsonNumberToInt64(parsed.Created); ok {
		created = timestamp
	}
	id := parsed.ID
	if id == "" {
		id = "chatcmpl_aihub"
	}
	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}
	message := map[string]any{"role": "assistant", "content": text}
	if len(toolCalls) > 0 {
		message["tool_calls"] = toolCalls
	}
	payload := map[string]any{
		"id": id, "object": "chat.completion", "created": created, "model": model,
		"choices": []map[string]any{{"index": 0, "message": message, "finish_reason": finishReason}},
		"usage":   chatUsageFromResponse(parsed.Usage),
	}
	return json.Marshal(payload)
}

func responseOutputToChat(output []chatResponseOutputItem) (string, []map[string]any) {
	parts := make([]string, 0, len(output))
	toolCalls := make([]map[string]any, 0)
	for _, item := range output {
		switch item.Type {
		case "function_call", "custom_tool_call":
			callType, arguments := "function", item.Arguments
			if item.Type == "custom_tool_call" {
				callType, arguments = "custom", item.Input
			}
			toolCalls = append(toolCalls, map[string]any{
				"id": item.CallID, "type": callType,
				"function": map[string]any{"name": item.Name, "arguments": arguments},
			})
		default:
			for _, content := range item.Content {
				if (content.Type == "output_text" || content.Type == "text") && content.Text != "" {
					parts = append(parts, content.Text)
				}
			}
		}
	}
	return strings.Join(parts, ""), toolCalls
}

func chatUsageFromResponse(usage *chatResponseUsage) any {
	if usage == nil {
		return nil
	}
	total := usage.TotalTokens
	if total == 0 {
		total = usage.InputTokens + usage.OutputTokens
	}
	return map[string]any{
		"prompt_tokens": usage.InputTokens, "completion_tokens": usage.OutputTokens,
		"total_tokens": total,
	}
}

func jsonNumberToInt64(value any) (int64, bool) {
	switch number := value.(type) {
	case float64:
		return int64(number), true
	case json.Number:
		parsed, err := number.Int64()
		return parsed, err == nil
	default:
		return 0, false
	}
}
