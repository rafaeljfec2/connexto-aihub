package gateway

import "encoding/json"

// normalizeOpenRouterResponsesInput applies OpenRouter transport shaping only:
// wrap flat input parts (same as OpenCode) and map function tools to custom tools,
// which OpenRouter's Responses API accepts. Instructions and message text are unchanged.
func normalizeOpenRouterResponsesInput(body []byte) ([]byte, error) {
	wrapped, err := normalizeOpenCodeGoResponsesInput(body)
	if err != nil {
		return nil, err
	}
	return normalizeOpenRouterFunctionTools(wrapped)
}

func normalizeOpenRouterFunctionTools(body []byte) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return body, nil
	}
	rawTools, ok := envelope["tools"]
	if !ok || len(rawTools) == 0 {
		return body, nil
	}
	var tools []map[string]any
	if err := json.Unmarshal(rawTools, &tools); err != nil {
		return body, nil
	}
	changed := false
	for i, tool := range tools {
		custom, ok := openRouterFunctionToolAsCustom(tool)
		if !ok {
			continue
		}
		tools[i] = custom
		changed = true
	}
	if !changed {
		return body, nil
	}
	next, err := json.Marshal(tools)
	if err != nil {
		return nil, err
	}
	envelope["tools"] = next
	return json.Marshal(envelope)
}

func openRouterFunctionToolAsCustom(tool map[string]any) (map[string]any, bool) {
	toolType, _ := tool["type"].(string)
	if toolType != "function" {
		return nil, false
	}
	custom := map[string]any{"type": "custom"}
	if name, _ := tool["name"].(string); name != "" {
		custom["name"] = name
	} else if fn, ok := tool["function"].(map[string]any); ok {
		if name, _ := fn["name"].(string); name != "" {
			custom["name"] = name
		}
		if params, ok := fn["parameters"]; ok {
			custom["parameters"] = params
		}
		if desc, ok := fn["description"].(string); ok && desc != "" {
			custom["description"] = desc
		}
	}
	if _, ok := custom["name"]; !ok {
		return nil, false
	}
	if params, ok := tool["parameters"]; ok {
		custom["parameters"] = params
	}
	if desc, ok := tool["description"].(string); ok && desc != "" {
		custom["description"] = desc
	}
	return custom, true
}

// OpenRouterResponsesPayloadForUpstream applies OpenRouter input shaping for tests.
func OpenRouterResponsesPayloadForUpstream(body []byte) ([]byte, error) {
	return normalizeOpenRouterResponsesInput(body)
}
