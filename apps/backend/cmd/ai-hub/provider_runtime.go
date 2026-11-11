package main

import (
	"context"
	"net/http"

	"aihub/internal/execution"
	"aihub/internal/providers"
)

func openResponsesFromRegistry(registry *providers.Registry) execution.OpenResponses {
	return func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
		return registry.OpenTargetResponses(ctx, providers.ResponsesTarget{
			ProviderID: target.ProviderID,
			ModelID:    target.ModelID,
		}, payload, contentType, sessionID)
	}
}
