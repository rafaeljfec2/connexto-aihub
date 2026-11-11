package capture

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

const maxFinishReasonLength = 128

// ExtractFinishReason reads a normalized terminal finish signal from captured provider bytes.
func ExtractFinishReason(result Result) *string {
	payload := terminalPayload(result)
	if len(payload) == 0 {
		return nil
	}
	var envelope struct {
		Response *struct {
			Status            string `json:"status"`
			IncompleteDetails *struct {
				Reason string `json:"reason"`
			} `json:"incomplete_details"`
		} `json:"response"`
		Choices []struct {
			FinishReason       string `json:"finish_reason"`
			NativeFinishReason string `json:"native_finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil
	}
	if envelope.Response != nil && envelope.Response.IncompleteDetails != nil {
		if normalized := normalizeFinishReason(envelope.Response.IncompleteDetails.Reason); normalized != nil {
			return normalized
		}
	}
	if envelope.Response != nil {
		if normalized := normalizeFinishReason(envelope.Response.Status); normalized != nil {
			return normalized
		}
	}
	if len(envelope.Choices) > 0 {
		if normalized := normalizeFinishReason(envelope.Choices[0].FinishReason); normalized != nil {
			return normalized
		}
		if normalized := normalizeFinishReason(envelope.Choices[0].NativeFinishReason); normalized != nil {
			return normalized
		}
	}
	return nil
}

func terminalPayload(result Result) []byte {
	if len(result.FinalEventData) > 0 {
		return result.FinalEventData
	}
	if len(result.ResponseBodyTail) > 0 {
		return result.ResponseBodyTail
	}
	return nil
}

func normalizeFinishReason(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || len(trimmed) > maxFinishReasonLength || !utf8.ValidString(trimmed) {
		return nil
	}
	for _, char := range trimmed {
		if char < 0x20 || char == 0x7f {
			return nil
		}
	}
	return &trimmed
}
