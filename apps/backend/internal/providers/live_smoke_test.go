package providers_test

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"aihub/internal/config"
	"aihub/internal/providers"
)

// TestLiveOpenCodeGoSmoke runs against the real OpenCode Go upstream. It never
// runs automatically: set AIHUB_LIVE_SMOKE=1 and provide
// AIHUB_PROVIDER_OPENCODEGO_API_KEY. A green mock suite does not qualify as
// real integration success (D6).
func TestLiveOpenCodeGoSmoke(t *testing.T) {
	if os.Getenv("AIHUB_LIVE_SMOKE") != "1" {
		t.Skip("skipped: set AIHUB_LIVE_SMOKE=1 to run the real integration smoke test")
	}

	credential := os.Getenv(config.EnvProviderOpenCodeGoAPIKey)
	if credential == "" {
		t.Skip("skipped: no AIHUB_PROVIDER_OPENCODEGO_API_KEY available")
	}

	baseURL := os.Getenv(config.EnvProviderOpenCodeGoBaseURL)
	if baseURL == "" {
		baseURL = "https://opencode.ai/inference/go"
	}

	client, err := providers.NewClient(config.ProviderOpenCodeGo{
		BaseURL: baseURL,
		APIKey:  credential,
	}, nil)
	if err != nil {
		t.Fatalf("NewClient() returned error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	models, err := client.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels() returned error: %v", err)
	}

	modelsBody, err := io.ReadAll(models.Body)
	if err != nil {
		t.Fatalf("read models body: %v", err)
	}

	if err := models.Body.Close(); err != nil {
		t.Fatalf("close models body: %v", err)
	}

	if models.StatusCode != 200 {
		t.Fatalf("models status = %d", models.StatusCode)
	}

	if !strings.Contains(string(modelsBody), "deepseek-v4.1-flash") {
		t.Fatalf("catalog does not list deepseek-v4.1-flash: %s", truncate(modelsBody))
	}

	responsesBody := io.NopCloser(strings.NewReader(`{"model":"deepseek-v4.1-flash",` +
		`"input":[{"role":"user","content":[{"type":"input_text","text":"Reply with exactly: pong"}]}],` +
		`"stream":true,"store":false}`))

	stream, err := client.OpenResponses(ctx, responsesBody, "application/json", "aihub-live-smoke-e02")
	if err != nil {
		t.Fatalf("OpenResponses() returned error: %v", err)
	}

	defer func() { _ = stream.Body.Close() }()

	streamBody, err := io.ReadAll(stream.Body)
	if err != nil {
		t.Fatalf("read stream body: %v", err)
	}

	if stream.StatusCode != 200 {
		t.Fatalf("responses status = %d, body: %s", stream.StatusCode, truncate(streamBody))
	}

	if !strings.Contains(string(streamBody), "response.completed") {
		t.Fatalf("stream did not reach response.completed: %s", truncate(streamBody))
	}

	t.Logf("live smoke OK: %d stream bytes", len(streamBody))
}

func truncate(body []byte) string {
	const limit = 400

	if len(body) <= limit {
		return string(body)
	}

	return string(body[:limit]) + "..."
}
