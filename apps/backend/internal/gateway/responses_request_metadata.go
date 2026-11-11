package gateway

import (
	"encoding/json"
	"errors"
	"strings"

	"aihub/internal/routing"
)

const maxObservedModelIDBytes = 4096

func structuralRejectionFromRouteError(err error) string {
	if err == nil {
		return ""
	}
	var failure routingHTTPError
	if !errors.As(err, &failure) {
		return ""
	}
	if failure.typ != string(routing.StructuralInvalidRequest) && failure.typ != invalidRequestType {
		return ""
	}
	return failure.typ
}

func requestedModelID(payload []byte) string {
	var request struct {
		Model *string `json:"model"`
	}
	if err := json.Unmarshal(payload, &request); err != nil || request.Model == nil {
		return ""
	}
	modelID := *request.Model
	if strings.TrimSpace(modelID) == "" || len(modelID) > maxObservedModelIDBytes {
		return ""
	}
	return modelID
}
