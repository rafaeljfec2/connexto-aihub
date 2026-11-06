package gateway_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
)

func TestOpenCodeGoRoutingNormalizesFlatMultimodalInput(t *testing.T) {
	var upstreamBody string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		upstreamBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"resp-test","status":"completed"}`)
	}))
	defer upstream.Close()

	logger, _ := newTestLogger()
	now := time.Now()
	reasoning := true
	descriptor := providers.ModelDescriptor{
		ID:                      "gpt-6-luna",
		InputModalities:         []string{"text", "image/png"},
		InputModalitiesPresent:  true,
		OutputModalities:        []string{"text"},
		OutputModalitiesPresent: true,
		ReasoningSupported:      &reasoning,
	}
	catalogManager := gatewayTestCatalog(t, now, map[string]catalog.SourceFunc{
		providers.OpenCodeGoID: func(_ context.Context) ([]providers.ModelDescriptor, error) {
			return []providers.ModelDescriptor{descriptor}, nil
		},
	})

	handler := newOpenCodeRoutingHandlerWithCatalog(t, upstream.URL, logger, time.Minute, catalogManager, "gpt-6-luna")
	gatewayServer := httptest.NewServer(handler)
	defer gatewayServer.Close()

	png := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	body := strings.Join([]string{
		`{"model":"gpt-6-luna","input":[`,
		`{"type":"input_text","text":"color?"},`,
		`{"type":"input_image","image_url":"data:image/png;base64,`, png, `"}`,
		`],"reasoning":{"effort":"low"}}`,
	}, "")

	resp, err := http.Post(gatewayServer.URL+"/v1/responses", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body = %s", resp.StatusCode, raw)
	}

	var envelope struct {
		Input []json.RawMessage `json:"input"`
	}
	if err := json.Unmarshal([]byte(upstreamBody), &envelope); err != nil {
		t.Fatalf("upstream json: %v body=%s", err, upstreamBody)
	}
	var first map[string]any
	if err := json.Unmarshal(envelope.Input[0], &first); err != nil {
		t.Fatal(err)
	}
	if first["type"] != "message" {
		t.Fatalf("first input type = %v, want message", first["type"])
	}
}
