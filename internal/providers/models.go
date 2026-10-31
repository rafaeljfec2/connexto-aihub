package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// maxModelsResponseBytes bounds the upstream catalog response. The real Go
// catalog is roughly 34 KiB; the cap protects memory against a hostile or
// corrupt upstream.
const maxModelsResponseBytes = 8 << 20

// UpstreamHTTPError reports a non-2xx response from the provider. It never
// carries the response body, URL query or credential.
type UpstreamHTTPError struct {
	StatusCode int
}

// Error implements error.
func (e *UpstreamHTTPError) Error() string {
	return fmt.Sprintf("provider returned status %d", e.StatusCode)
}

// DecodeError reports an unreadable or structurally invalid catalog payload.
// StatusCode is the upstream HTTP status observed before decoding (0 when no
// response was received).
type DecodeError struct {
	StatusCode int
	Err        error
}

// Error implements error.
func (e *DecodeError) Error() string {
	return fmt.Sprintf("decode provider catalog: %v", e.Err)
}

// Unwrap exposes the underlying cause.
func (e *DecodeError) Unwrap() error {
	return e.Err
}

// ModelDescriptor is the parsed representation of one model exposed by a
// provider. The pointer and presence fields distinguish an absent upstream
// field from an explicit false, zero or empty value.
type ModelDescriptor struct {
	ID          string
	CanonicalID string
	Name        string
	Type        string
	Family      string

	InputModalities         []string
	InputModalitiesPresent  bool
	OutputModalities        []string
	OutputModalitiesPresent bool

	LimitContext *int64
	LimitInput   *int64
	LimitOutput  *int64

	ToolsSupported     *bool
	ReasoningSupported *bool
	StructuredOutput   *bool

	Raw json.RawMessage
}

// FetchModelDescriptors retrieves and parses the provider catalog. It uses the
// vendor-specific models path and never mutates the E02 pass-through path.
func (c *Client) FetchModelDescriptors(ctx context.Context) ([]ModelDescriptor, error) {
	response, err := c.do(ctx, http.MethodGet, modelsPath, nil, nil)
	if err != nil {
		return nil, err
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))

		return nil, &UpstreamHTTPError{StatusCode: response.StatusCode}
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxModelsResponseBytes+1))
	if err != nil {
		return nil, &DecodeError{StatusCode: response.StatusCode, Err: err}
	}

	if len(body) > maxModelsResponseBytes {
		return nil, &DecodeError{StatusCode: response.StatusCode, Err: errors.New("catalog payload exceeds 8 MiB")}
	}

	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, &DecodeError{StatusCode: response.StatusCode, Err: err}
	}

	descriptors := make([]ModelDescriptor, 0, len(envelope.Data))
	seen := make(map[string]struct{}, len(envelope.Data))

	for _, raw := range envelope.Data {
		var model rawModel

		if err := json.Unmarshal(raw, &model); err != nil {
			return nil, &DecodeError{StatusCode: response.StatusCode, Err: err}
		}

		if model.ID == "" {
			return nil, &DecodeError{StatusCode: response.StatusCode, Err: errors.New("model with empty id")}
		}

		if _, duplicate := seen[model.ID]; duplicate {
			return nil, &DecodeError{StatusCode: response.StatusCode, Err: fmt.Errorf("duplicate model id %q", model.ID)}
		}

		seen[model.ID] = struct{}{}

		descriptors = append(descriptors, descriptorFromRaw(model, raw))
	}

	return descriptors, nil
}

type rawModel struct {
	ID          string `json:"id"`
	CanonicalID string `json:"canonical_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Family      string `json:"family"`

	Modalities   *rawModalities  `json:"modalities"`
	Limit        rawLimits       `json:"limit"`
	Capabilities rawCapabilities `json:"capabilities"`
}

type rawModalities struct {
	Input  *[]string `json:"input"`
	Output *[]string `json:"output"`
}

type rawLimits struct {
	Context *int64 `json:"context"`
	Input   *int64 `json:"input"`
	Output  *int64 `json:"output"`
}

type rawCapabilities struct {
	Tools            *rawSupported `json:"tools"`
	Reasoning        *rawSupported `json:"reasoning"`
	StructuredOutput *bool         `json:"structured_output"`
}

type rawSupported struct {
	Supported *bool `json:"supported"`
}

func descriptorFromRaw(model rawModel, raw json.RawMessage) ModelDescriptor {
	descriptor := ModelDescriptor{
		ID:           model.ID,
		CanonicalID:  model.CanonicalID,
		Name:         model.Name,
		Type:         model.Type,
		Family:       model.Family,
		LimitContext: model.Limit.Context,
		LimitInput:   model.Limit.Input,
		LimitOutput:  model.Limit.Output,
		Raw:          raw,
	}

	if model.Modalities != nil {
		if model.Modalities.Input != nil {
			descriptor.InputModalities = *model.Modalities.Input
			descriptor.InputModalitiesPresent = true
		}

		if model.Modalities.Output != nil {
			descriptor.OutputModalities = *model.Modalities.Output
			descriptor.OutputModalitiesPresent = true
		}
	}

	if model.Capabilities.Tools != nil {
		descriptor.ToolsSupported = model.Capabilities.Tools.Supported
	}

	if model.Capabilities.Reasoning != nil {
		descriptor.ReasoningSupported = model.Capabilities.Reasoning.Supported
	}

	descriptor.StructuredOutput = model.Capabilities.StructuredOutput

	return descriptor
}
