package routing

import (
	"encoding/json"
	"sort"
)

// ProjectRequirements derives normalized capability requirements from structural JSON.
// Pass validated structural metadata so output token bounds come from ValidateStructure.
func ProjectRequirements(body []byte, structural StructuralRequest) (RequirementStatus, error) {
	var envelope struct {
		Input      json.RawMessage  `json:"input"`
		Tools      []toolItem       `json:"tools"`
		Reasoning  json.RawMessage  `json:"reasoning"`
		Text       *textEnvelope    `json:"text"`
		Modalities *modalitiesShape `json:"modalities"`
	}

	if err := json.Unmarshal(body, &envelope); err != nil {
		return RequirementStatus{}, err
	}

	required := map[string]struct{}{}

	if len(envelope.Input) > 0 {
		unmapped, err := projectInputRequirements(envelope.Input, required)
		if err != nil {
			return RequirementStatus{}, err
		}
		if unmapped {
			return RequirementStatus{Unmapped: true}, nil
		}
	}

	for _, tool := range envelope.Tools {
		switch tool.Type {
		case "function":
			required["function_tools"] = struct{}{}
		case "custom":
			required["custom_tools"] = struct{}{}
		}
	}

	if len(envelope.Reasoning) > 0 && string(envelope.Reasoning) != "null" && string(envelope.Reasoning) != "{}" {
		required["reasoning"] = struct{}{}
	}

	if envelope.Text != nil && envelope.Text.Format != nil {
		switch envelope.Text.Format.Type {
		case "json_schema", "json_object":
			required["structured_output"] = struct{}{}
		case "":
		default:
			return RequirementStatus{Unmapped: true}, nil
		}
	}

	if envelope.Modalities != nil {
		for _, modality := range envelope.Modalities.Output {
			if len(modality) >= 6 && modality[:6] == "audio/" {
				required["audio_output"] = struct{}{}
			}
		}
	}

	capabilities := make([]string, 0, len(required))
	for name := range required {
		capabilities = append(capabilities, name)
	}

	sort.Strings(capabilities)

	status := RequirementStatus{
		RequiredCapabilities: capabilities,
		Unmapped:             false,
		MaxOutputTokens:      structural.MaxOutputTokens,
	}

	return status, nil
}

func projectInputRequirements(raw json.RawMessage, required map[string]struct{}) (bool, error) {
	if len(raw) == 0 {
		return false, nil
	}

	if raw[0] == '"' {
		required["text_generation"] = struct{}{}
		return false, nil
	}

	var items []inputItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return false, err
	}

	for _, item := range items {
		switch item.Type {
		case "input_text", "":
			required["text_generation"] = struct{}{}
		case "input_image":
			required["image_input"] = struct{}{}
		case "input_audio":
			required["audio_input"] = struct{}{}
		case "input_file":
			return true, nil
		}
	}

	return false, nil
}

type inputItem struct {
	Type string `json:"type"`
}

type toolItem struct {
	Type string `json:"type"`
}

type textEnvelope struct {
	Format *formatType `json:"format"`
}

type formatType struct {
	Type string `json:"type"`
}

type modalitiesShape struct {
	Output []string `json:"output"`
}
