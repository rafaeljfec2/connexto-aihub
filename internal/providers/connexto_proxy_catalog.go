package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// FetchModelDescriptors parses claims from Connexto Proxy's catalog.
func (c *ConnextoProxyClient) FetchModelDescriptors(ctx context.Context) ([]ModelDescriptor, error) {
	response, err := c.ListModels(ctx)
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
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Data == nil {
		if err == nil {
			err = errors.New("catalog response has no data array")
		}
		return nil, &DecodeError{StatusCode: response.StatusCode, Err: err}
	}

	descriptors := make([]ModelDescriptor, 0, len(envelope.Data))
	seen := make(map[string]struct{}, len(envelope.Data))
	c.mu.RLock()
	logicalID := c.logicalProviderID
	c.mu.RUnlock()
	if logicalID == "" {
		logicalID = CursorID
	}

	for _, raw := range envelope.Data {
		descriptor, err := connextoProxyDescriptor(raw)
		if err != nil {
			return nil, &DecodeError{StatusCode: response.StatusCode, Err: err}
		}
		switch logicalID {
		case AntigravityID:
			if !connextoProxyModelBelongsToAntigravity(descriptor.ID, raw) {
				continue
			}
		default:
			if connextoProxyModelBelongsToAntigravity(descriptor.ID, raw) {
				continue
			}
		}
		if _, duplicate := seen[descriptor.ID]; duplicate {
			return nil, &DecodeError{StatusCode: response.StatusCode, Err: errors.New("catalog contains duplicate model IDs")}
		}
		seen[descriptor.ID] = struct{}{}
		descriptors = append(descriptors, descriptor)
	}
	return descriptors, nil
}

func rawJSONOwnedBy(raw json.RawMessage) string {
	var probe struct {
		OwnedBy string `json:"owned_by"`
	}
	_ = json.Unmarshal(raw, &probe)
	return probe.OwnedBy
}

func connextoProxyModelBelongsToAntigravity(modelID string, raw json.RawMessage) bool {
	return IsAntigravityCatalogModel(rawJSONOwnedBy(raw), modelID)
}

func connextoProxyDescriptor(raw json.RawMessage) (ModelDescriptor, error) {
	var model struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Object       string `json:"object"`
		OwnedBy      string `json:"owned_by"`
		Modalities   *struct {
			Input *[]string `json:"input"`
		} `json:"modalities"`
		Capabilities struct {
			Tools     *rawSupported `json:"tools"`
			Reasoning *rawSupported `json:"reasoning"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(raw, &model); err != nil {
		return ModelDescriptor{}, err
	}
	if model.ID == "" {
		return ModelDescriptor{}, errors.New("catalog model has no ID")
	}
	name := model.Name
	if name == "" {
		name = model.ID
	}

	antigravity := IsAntigravityCatalogModel(model.OwnedBy, model.ID)

	var toolsSupported, reasoningSupported bool
	if antigravity {
		toolsSupported = model.Capabilities.Tools != nil && model.Capabilities.Tools.Supported != nil && *model.Capabilities.Tools.Supported
		reasoningSupported = model.Capabilities.Reasoning != nil && model.Capabilities.Reasoning.Supported != nil && *model.Capabilities.Reasoning.Supported
	} else {
		// OpenAI-minimal proxy catalogs omit capabilities; Cursor agent models support tools and reasoning.
		toolsSupported = true
		reasoningSupported = true
		if model.Capabilities.Tools != nil && model.Capabilities.Tools.Supported != nil {
			toolsSupported = *model.Capabilities.Tools.Supported
		}
		if model.Capabilities.Reasoning != nil && model.Capabilities.Reasoning.Supported != nil {
			reasoningSupported = *model.Capabilities.Reasoning.Supported
		}
	}

	inputModalities := connextoProxyDefaultInputModalities()
	inputModalitiesPresent := true
	if antigravity {
		inputModalities = []string{"text"}
		inputModalitiesPresent = true
		if model.Modalities != nil && model.Modalities.Input != nil {
			inputModalities = NormalizeInputModalitiesForCatalog(*model.Modalities.Input)
		}
	} else {
		// SPEC E03 §5.4 A2: cursor sync uses vision defaults when upstream omits modalities.input.
		if model.Modalities != nil && model.Modalities.Input != nil {
			inputModalities = NormalizeInputModalitiesForCatalog(*model.Modalities.Input)
			inputModalitiesPresent = true
		}
	}

	limitCtx, limitOut := connextoProxyLimits(antigravity, model.ID)

	descriptor := ModelDescriptor{
		ID:                      model.ID,
		Name:                    name,
		Type:                    model.Object,
		InputModalities:         inputModalities,
		InputModalitiesPresent:  inputModalitiesPresent,
		OutputModalities:        []string{"text"},
		OutputModalitiesPresent: true,
		LimitContext:            limitCtx,
		LimitOutput:             limitOut,
		ToolsSupported:          boolPtr(toolsSupported),
		ReasoningSupported:      boolPtr(reasoningSupported),
		StructuredOutput:        boolPtr(true),
		Raw:                     bytes.Clone(raw),
	}
	return descriptor, nil
}

func connextoProxyLimits(antigravity bool, modelID string) (*int64, *int64) {
	if antigravity {
		switch {
		case strings.HasSuffix(modelID, "gemini-3.8-flash"):
			return int64Ptr(1048576), int64Ptr(65536)
		case strings.HasSuffix(modelID, "claude-sonnet-4-6"), strings.HasSuffix(modelID, "claude-opus-4-6-thinking"):
			return int64Ptr(200000), int64Ptr(64000)
		case strings.HasSuffix(modelID, "gpt-oss-120b-medium"):
			return int64Ptr(128000), int64Ptr(16384)
		default:
			return int64Ptr(128000), int64Ptr(16384)
		}
	}

	switch modelID {
	case "composer-2.5", "composer-2.5-fast":
		return int64Ptr(200000), int64Ptr(16384)
	case "grok-4.6", "grok-4.6-fast":
		return int64Ptr(256000), int64Ptr(8192)
	case "grok-4.5", "grok-4.5-fast":
		return int64Ptr(128000), int64Ptr(8192)
	default:
		return int64Ptr(128000), int64Ptr(8192)
	}
}

func connextoProxyDefaultInputModalities() []string {
	return DefaultVisionAgentInputModalities()
}

func (c *ConnextoProxyClient) filterModelsListResponse(response *http.Response) (*http.Response, error) {
	if response == nil || response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return response, nil
	}
	c.mu.RLock()
	logicalID := c.logicalProviderID
	c.mu.RUnlock()
	if logicalID == "" {
		logicalID = CursorID
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxModelsResponseBytes+1))
	if response.Body != nil {
		_ = response.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("read model list: %w", err)
	}
	if len(body) > maxModelsResponseBytes {
		return nil, errors.New("catalog payload exceeds 8 MiB")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		response.Body = io.NopCloser(strings.NewReader(string(body)))
		response.ContentLength = int64(len(body))
		return response, nil
	}
	rawData, ok := envelope["data"]
	if !ok {
		response.Body = io.NopCloser(strings.NewReader(string(body)))
		response.ContentLength = int64(len(body))
		return response, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(rawData, &items); err != nil {
		response.Body = io.NopCloser(strings.NewReader(string(body)))
		response.ContentLength = int64(len(body))
		return response, nil
	}
	filtered := make([]json.RawMessage, 0, len(items))
	for _, raw := range items {
		var idProbe struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		}
		_ = json.Unmarshal(raw, &idProbe)
		isAnt := IsAntigravityCatalogModel(idProbe.OwnedBy, idProbe.ID)
		switch logicalID {
		case AntigravityID:
			if isAnt {
				filtered = append(filtered, raw)
			}
		default:
			if !isAnt {
				filtered = append(filtered, raw)
			}
		}
	}
	dataBytes, err := json.Marshal(filtered)
	if err != nil {
		return nil, err
	}
	envelope["data"] = dataBytes
	out, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	response.Body = io.NopCloser(bytes.NewReader(out))
	response.ContentLength = int64(len(out))
	response.Header.Set("Content-Type", "application/json")
	return response, nil
}

