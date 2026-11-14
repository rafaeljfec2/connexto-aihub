package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const maxResponsesCallIDLength = 64

func limitResponsesCallIDs(body []byte) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return body, nil
	}
	rawInput, ok := envelope["input"]
	if !ok || len(rawInput) == 0 || rawInput[0] != '[' {
		return body, nil
	}
	var items []map[string]any
	if err := json.Unmarshal(rawInput, &items); err != nil {
		return body, nil
	}
	changed := false
	for _, item := range items {
		raw, ok := item["call_id"].(string)
		if !ok {
			continue
		}
		next := fitResponsesCallID(raw)
		if next == raw {
			continue
		}
		item["call_id"] = next
		changed = true
	}
	if !changed {
		return body, nil
	}
	nextInput, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}
	envelope["input"] = nextInput
	return json.Marshal(envelope)
}

func fitResponsesCallID(raw string) string {
	if len(raw) <= maxResponsesCallIDLength {
		return raw
	}
	sum := sha256.Sum256([]byte(raw))
	return "call_" + hex.EncodeToString(sum[:16])
}
