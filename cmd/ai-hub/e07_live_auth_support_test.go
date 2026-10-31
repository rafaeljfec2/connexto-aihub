package main

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"aihub/internal/config"
)

type e07StaticRoundTripper struct{}

func (e07StaticRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"data":[]}`))),
		Request:    request,
	}, nil
}

func e07StartAuthProbeHub(t *testing.T) (*e07Hub, *e07ProviderObserver, error) {
	t.Helper()
	providerKey := "e07-synthetic-provider-boundary-key"
	gatewayKey, err := e07TemporaryGatewayKey(providerKey)
	if err != nil {
		return nil, nil, errors.New("probe gateway credential could not be created")
	}
	cfg := config.Default()
	cfg.Gateway.APIKey = gatewayKey
	cfg.Provider.OpenCodeGo.APIKey = providerKey
	cfg.Catalog.DataDir = t.TempDir()
	observer := newE07ProviderObserver(e07StaticRoundTripper{}, "")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	hub, err := startE07Hub(cfg, observer, logger)
	if err != nil {
		return nil, nil, errors.New("probe gateway could not start")
	}
	return hub, observer, nil
}
