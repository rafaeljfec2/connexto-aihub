package gateway

import "encoding/json"

// promoteAdditionalToolsToTopLevel moves Codex-style input[].additional_tools
// inventories into top-level tools and removes those items from input.
// OpenCode Go and OpenRouter reject additional_tools entries that lack role.
func promoteAdditionalToolsToTopLevel(body []byte) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return body, nil
	}
	rawInput, ok := envelope["input"]
	if !ok || len(rawInput) == 0 || rawInput[0] == '"' {
		return body, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(rawInput, &items); err != nil {
		return body, nil
	}

	promoted := make([]map[string]any, 0)
	kept := make([]json.RawMessage, 0, len(items))
	changed := false
	for _, raw := range items {
		var item map[string]any
		if err := json.Unmarshal(raw, &item); err != nil {
			kept = append(kept, raw)
			continue
		}
		itemType, _ := item["type"].(string)
		if itemType != "additional_tools" {
			kept = append(kept, raw)
			continue
		}
		changed = true
		collectPromotedToolNodes(item["tools"], &promoted)
	}
	if !changed {
		return body, nil
	}

	existing := make([]map[string]any, 0)
	if rawTools, ok := envelope["tools"]; ok && len(rawTools) > 0 {
		_ = json.Unmarshal(rawTools, &existing)
	}
	merged := mergeToolMaps(existing, promoted)
	toolsRaw, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}
	envelope["tools"] = toolsRaw

	inputRaw, err := json.Marshal(kept)
	if err != nil {
		return nil, err
	}
	envelope["input"] = inputRaw
	return json.Marshal(envelope)
}

func collectPromotedToolNodes(value any, out *[]map[string]any) {
	nodes, ok := value.([]any)
	if !ok {
		return
	}
	for _, node := range nodes {
		item, ok := node.(map[string]any)
		if !ok {
			continue
		}
		itemType, _ := item["type"].(string)
		if itemType == "namespace" {
			collectPromotedToolNodes(item["tools"], out)
			continue
		}
		name := toolNameFromNode(item)
		if name == "" {
			continue
		}
		tool := map[string]any{"type": "function", "name": name}
		if itemType == "custom" {
			tool["type"] = "custom"
		}
		if desc, ok := item["description"].(string); ok && desc != "" {
			tool["description"] = desc
		}
		if params, ok := item["parameters"]; ok {
			tool["parameters"] = params
		}
		if fn, ok := item["function"].(map[string]any); ok {
			if params, ok := fn["parameters"]; ok {
				tool["parameters"] = params
			}
			if desc, ok := fn["description"].(string); ok && desc != "" {
				tool["description"] = desc
			}
		}
		*out = append(*out, tool)
	}
}

func toolNameFromNode(item map[string]any) string {
	if name, ok := item["name"].(string); ok && name != "" {
		return name
	}
	if fn, ok := item["function"].(map[string]any); ok {
		if name, ok := fn["name"].(string); ok {
			return name
		}
	}
	return ""
}

func mergeToolMaps(existing, promoted []map[string]any) []map[string]any {
	seen := make(map[string]struct{}, len(existing)+len(promoted))
	out := make([]map[string]any, 0, len(existing)+len(promoted))
	for _, tool := range existing {
		name := toolNameFromNode(tool)
		if name != "" {
			seen[name] = struct{}{}
		}
		out = append(out, tool)
	}
	for _, tool := range promoted {
		name := toolNameFromNode(tool)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, tool)
	}
	return out
}
