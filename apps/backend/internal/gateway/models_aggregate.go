package gateway

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"aihub/internal/providers"
)

type openAIModelListEnvelope struct {
	Object string            `json:"object"`
	Data   []json.RawMessage `json:"data"`
}

func writeAggregatedRegistryModels(
	writer http.ResponseWriter,
	request *http.Request,
	registry *providers.Registry,
	logger *slog.Logger,
) {
	providerIDs := registry.ProviderIDs()
	merged := make([]json.RawMessage, 0)
	attempted := 0

	for _, providerID := range providerIDs {
		adapter, ok := registry.Resolve(providerID)
		if !ok {
			continue
		}
		models, ok := adapter.(providers.ModelListAdapter)
		if !ok || !adapter.Enabled() {
			continue
		}
		attempted++
		items, err := fetchProviderModelListItems(request.Context(), models, providerID, logger)
		if err != nil {
			logger.WarnContext(request.Context(), "provider model list skipped during aggregation",
				"provider", providerID, "error", err)
			continue
		}
		merged = append(merged, items...)
	}

	if attempted == 0 {
		writeJSONError(writer, http.StatusServiceUnavailable, providerUnavailableType, responseProviderUnavailable)
		return
	}
	if len(merged) == 0 {
		writeJSONError(writer, http.StatusBadGateway, upstreamUnavailableType, upstreamUnavailableMsg)
		return
	}

	virtualAuto := json.RawMessage(`{"id":"aihub/auto","object":"model","created":1700000000,"owned_by":"connexto"}`)
	merged = append([]json.RawMessage{virtualAuto}, merged...)

	writeJSON(writer, http.StatusOK, openAIModelListEnvelope{
		Object: "list",
		Data:   merged,
	})
}

func fetchProviderModelListItems(
	ctx context.Context,
	models providers.ModelListAdapter,
	providerID string,
	logger *slog.Logger,
) ([]json.RawMessage, error) {
	response, err := models.ListModels(ctx)
	if err != nil || response == nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, upstreamStatusError{status: response.StatusCode}
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	items, err := parseOpenAIModelListData(body)
	if err != nil {
		logger.WarnContext(ctx, "provider model list payload unparsable", "provider", providerID, "error", err)
		return nil, err
	}
	return items, nil
}

func parseOpenAIModelListData(body []byte) ([]json.RawMessage, error) {
	var envelope openAIModelListEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Data) == 0 {
		return nil, nil
	}
	return envelope.Data, nil
}

type upstreamStatusError struct {
	status int
}

func (e upstreamStatusError) Error() string {
	return http.StatusText(e.status)
}
