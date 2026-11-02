package support

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"aihub/internal/config"
)

type E07StaticRoundTripper struct{}

func (E07StaticRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"data":[]}`))),
		Request:    request,
	}, nil
}

func E07StartAuthProbeHub(t *testing.T) (*E07Hub, *E07ProviderObserver, config.Config, error) {
	t.Helper()
	providerKey := "e07-synthetic-provider-boundary-key"
	gatewayKey, err := E07TemporaryGatewayKey(providerKey)
	if err != nil {
		return nil, nil, config.Config{}, errors.New("probe gateway credential could not be created")
	}
	cfg := config.Default()
	cfg.Gateway.APIKey = gatewayKey
	cfg.Provider.OpenCodeGo.APIKey = providerKey
	cfg.Catalog.DataDir = t.TempDir()
	observer := NewE07ProviderObserver(E07StaticRoundTripper{}, "")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	hub, err := StartE07Hub(cfg, observer, logger)
	if err != nil {
		return nil, nil, config.Config{}, errors.New("probe gateway could not start")
	}
	return hub, observer, cfg, nil
}
