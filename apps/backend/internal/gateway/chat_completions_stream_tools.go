package gateway

import "encoding/json"

type chatStreamToolCall struct {
	callID    string
	name      string
	arguments string
	custom    bool
}

func toolCallFromOutputItem(item chatResponseOutputItem) (chatStreamToolCall, bool) {
	switch item.Type {
	case "function_call":
		if item.CallID == "" || item.Name == "" {
			return chatStreamToolCall{}, false
		}
		return chatStreamToolCall{
			callID: item.CallID, name: item.Name, arguments: item.Arguments,
		}, true
	case "custom_tool_call":
		if item.CallID == "" || item.Name == "" {
			return chatStreamToolCall{}, false
		}
		return chatStreamToolCall{
			callID: item.CallID, name: item.Name, arguments: item.Input, custom: true,
		}, true
	default:
		return chatStreamToolCall{}, false
	}
}

// responsesStreamToolCall extracts a completed tool call from a single Responses SSE data block.
func responsesStreamToolCall(eventName string, data []byte) (chatStreamToolCall, bool) {
	kind := streamEventKind(eventName, data)
	switch kind {
	case "response.output_item.done":
		var envelope struct {
			Item chatResponseOutputItem `json:"item"`
		}
		if json.Unmarshal(data, &envelope) != nil {
			return chatStreamToolCall{}, false
		}
		tc, ok := toolCallFromOutputItem(envelope.Item)
		if !ok || !toolCallReady(tc) {
			return chatStreamToolCall{}, false
		}
		return tc, true
	case "response.output_item.added":
		var envelope struct {
			Item chatResponseOutputItem `json:"item"`
		}
		if json.Unmarshal(data, &envelope) != nil {
			return chatStreamToolCall{}, false
		}
		tc, ok := toolCallFromOutputItem(envelope.Item)
		if !ok || !toolCallReady(tc) {
			return chatStreamToolCall{}, false
		}
		return tc, true
	default:
		return chatStreamToolCall{}, false
	}
}

func responsesStreamToolCallPartial(eventName string, data []byte) (itemID string, partial chatStreamToolCall, argsOnly bool, ok bool) {
	kind := streamEventKind(eventName, data)
	var envelope struct {
		Type      string                   `json:"type"`
		ItemID    string                   `json:"item_id"`
		Item      chatResponseOutputItem   `json:"item"`
		Arguments string                   `json:"arguments"`
		Input     string                   `json:"input"`
		Response  struct {
			Output []chatResponseOutputItem `json:"output"`
		} `json:"response"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return "", chatStreamToolCall{}, false, false
	}

	switch kind {
	case "response.output_item.added":
		tc, itemOK := toolCallFromOutputItem(envelope.Item)
		if !itemOK {
			return "", chatStreamToolCall{}, false, false
		}
		id, ok := outputItemStreamID(data)
		if !ok {
			id = envelope.Item.CallID
		}
		return id, tc, false, true
	case "response.function_call_arguments.done":
		if envelope.ItemID == "" || envelope.Arguments == "" {
			return "", chatStreamToolCall{}, false, false
		}
		return envelope.ItemID, chatStreamToolCall{arguments: envelope.Arguments}, true, true
	case "response.custom_tool_call_input.done":
		if envelope.ItemID == "" || envelope.Input == "" {
			return "", chatStreamToolCall{}, false, false
		}
		return envelope.ItemID, chatStreamToolCall{arguments: envelope.Input, custom: true}, true, true
	default:
		return "", chatStreamToolCall{}, false, false
	}
}

func responsesCompletedOutputTools(data []byte) []chatStreamToolCall {
	var envelope struct {
		Response struct {
			Output []chatResponseOutputItem `json:"output"`
		} `json:"response"`
		Output []chatResponseOutputItem `json:"output"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return nil
	}
	output := envelope.Response.Output
	if len(output) == 0 {
		output = envelope.Output
	}
	calls := make([]chatStreamToolCall, 0, len(output))
	for _, item := range output {
		if tc, ok := toolCallFromOutputItem(item); ok && tc.arguments != "" {
			calls = append(calls, tc)
		}
	}
	return calls
}

func streamEventKind(eventName string, data []byte) string {
	if eventName != "" {
		return eventName
	}
	var envelope struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(data, &envelope) == nil && envelope.Type != "" {
		return envelope.Type
	}
	return ""
}

func outputItemStreamID(data []byte) (string, bool) {
	var envelope struct {
		Item struct {
			ID string `json:"id"`
		} `json:"item"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.Item.ID == "" {
		return "", false
	}
	return envelope.Item.ID, true
}

func mergePartialToolCall(base chatStreamToolCall, patch chatStreamToolCall) chatStreamToolCall {
	if patch.callID != "" {
		base.callID = patch.callID
	}
	if patch.name != "" {
		base.name = patch.name
	}
	if patch.arguments != "" {
		base.arguments = patch.arguments
	}
	base.custom = base.custom || patch.custom
	return base
}

func toolCallReady(tc chatStreamToolCall) bool {
	return tc.callID != "" && tc.name != "" && tc.arguments != ""
}
