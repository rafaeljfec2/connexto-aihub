package routing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const maxModelIDBytes = 4096

// ValidateStructure parses routing-relevant fields and rejects malformed input.
func ValidateStructure(body []byte) (StructuralRequest, *StructuralError) {
	if len(body) == 0 {
		return StructuralRequest{}, &StructuralError{Code: StructuralInvalidRequest, Detail: "empty body"}
	}

	if duplicate, err := duplicateTopLevelKey(body, "model"); err != nil {
		return StructuralRequest{}, &StructuralError{Code: StructuralInvalidRequest, Detail: err.Error()}
	} else if duplicate {
		return StructuralRequest{}, &StructuralError{Code: StructuralInvalidRequest, Detail: "duplicate model key"}
	}

	if duplicate, err := duplicateTopLevelKey(body, "max_output_tokens"); err != nil {
		return StructuralRequest{}, &StructuralError{Code: StructuralInvalidRequest, Detail: err.Error()}
	} else if duplicate {
		return StructuralRequest{}, &StructuralError{Code: StructuralInvalidRequest, Detail: "duplicate max_output_tokens key"}
	}

	topLevel := map[string]json.RawMessage{}
	if err := json.Unmarshal(body, &topLevel); err != nil {
		return StructuralRequest{}, &StructuralError{Code: StructuralInvalidRequest, Detail: "invalid json"}
	}

	result := StructuralRequest{}

	if modelRaw, ok := topLevel["model"]; ok {
		if bytes.Equal(modelRaw, []byte("null")) {
			return StructuralRequest{}, &StructuralError{Code: StructuralInvalidRequest, Detail: "model null"}
		}

		var modelID string
		if err := json.Unmarshal(modelRaw, &modelID); err != nil {
			return StructuralRequest{}, &StructuralError{Code: StructuralInvalidRequest, Detail: "model type"}
		}

		if strings.TrimSpace(modelID) == "" {
			return StructuralRequest{}, &StructuralError{Code: StructuralInvalidRequest, Detail: "model empty"}
		}

		if len(modelID) > maxModelIDBytes {
			return StructuralRequest{}, &StructuralError{Code: StructuralInvalidRequest, Detail: "model too long"}
		}

		result.ModelPresent = true
		result.ModelID = modelID
	}

	if maxRaw, ok := topLevel["max_output_tokens"]; ok {
		if bytes.Equal(maxRaw, []byte("null")) {
			return result, nil
		}

		if len(maxRaw) > 0 && maxRaw[0] == '"' {
			return StructuralRequest{}, &StructuralError{Code: StructuralInvalidRequest, Detail: "max_output_tokens type"}
		}

		var number json.Number
		if err := json.Unmarshal(maxRaw, &number); err != nil {
			return StructuralRequest{}, &StructuralError{Code: StructuralInvalidRequest, Detail: "max_output_tokens type"}
		}

		value, err := number.Int64()
		if err != nil {
			return StructuralRequest{}, &StructuralError{Code: StructuralInvalidRequest, Detail: "max_output_tokens not integer"}
		}

		if value <= 0 {
			return StructuralRequest{}, &StructuralError{Code: StructuralInvalidRequest, Detail: "max_output_tokens nonpositive"}
		}

		result.MaxOutputTokens = &value
	}

	return result, nil
}

// StructuralError describes invalid routing input separate from eligibility.
type StructuralError struct {
	Code   StructuralErrorCode
	Detail string
}

func (e *StructuralError) Error() string {
	if e == nil {
		return ""
	}

	return fmt.Sprintf("%s: %s", e.Code, e.Detail)
}

func duplicateTopLevelKey(body []byte, key string) (bool, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	token, err := decoder.Token()
	if err != nil {
		return false, err
	}

	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return false, fmt.Errorf("expected object")
	}

	seen := false

	for decoder.More() {
		fieldToken, err := decoder.Token()
		if err != nil {
			return false, err
		}

		name, ok := fieldToken.(string)
		if !ok {
			return false, fmt.Errorf("expected field name")
		}

		if name == key {
			if seen {
				return true, nil
			}

			seen = true
		}

		var skip json.RawMessage
		if err := decoder.Decode(&skip); err != nil {
			return false, err
		}
	}

	_, err = decoder.Token()
	return false, err
}
