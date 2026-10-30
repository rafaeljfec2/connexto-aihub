package providers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aihub/internal/config"
	"aihub/internal/providers"
)

const testCredential = "oc_sk_fixture_secret_credential"

func newEnabledClientConfig(baseURL string) config.ProviderOpenCodeGo {
	return config.ProviderOpenCodeGo{BaseURL: baseURL, APIKey: testCredential}
}

func mustClient(t *testing.T, cfg config.ProviderOpenCodeGo) *providers.Client {
	t.Helper()

	client, err := providers.NewClient(cfg, nil)
	if err != nil {
		t.Fatalf("NewClient() returned error: %v", err)
	}

	return client
}

func TestNewClientRejectsInvalidBaseURL(t *testing.T) {
	cases := map[string]string{
		"empty":        "",
		"scheme only":  "https://",
		"not http(s)":  "ftp://example.com/go",
		"embedded key": "https://key@example.com/go",
	}

	for name, baseURL := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := providers.NewClient(config.ProviderOpenCodeGo{BaseURL: baseURL, APIKey: testCredential}, nil)
			if err == nil {
				t.Fatalf("NewClient(%q) accepted an invalid base URL", baseURL)
			}
		})
	}
}

func TestClientEnabledReflectsCredentialPresence(t *testing.T) {
	disabled := mustClient(t, config.ProviderOpenCodeGo{BaseURL: "https://example.com/go"})
	if disabled.Enabled() {
		t.Fatalf("client without credential must be disabled")
	}

	enabled := mustClient(t, newEnabledClientConfig("https://example.com/go"))
	if !enabled.Enabled() {
		t.Fatalf("client with credential must be enabled")
	}
}

func TestOpenResponsesPassesBodyVerbatim(t *testing.T) {
	const payload = `{"model":"deepseek-v4.1-flash","instructions":"be brief",` +
		`"input":[{"role":"user","content":[{"type":"input_text","text":"hi"}]}],` +
		`"tools":[{"type":"function","name":"read_file"},{"type":"custom","name":"apply_patch"}],` +
		`"tool_choice":"auto","parallel_tool_calls":true,"reasoning":{"effort":"high"},` +
		`"store":false,"stream":true,"include":["reasoning.encrypted_content"],` +
		`"prompt_cache_key":"pc_1","text":{"verbosity":"low"},"service_tier":"auto",` +
		`"client_metadata":{"agent":"codex"}}`

	var (
		gotMethod      string
		gotPath        string
		gotAuth        string
		gotSession     string
		gotContentType string
		gotBody        string
	)

	upstream := httptest.NewServer(sinkHandler(&upstreamCapture{
		method: &gotMethod, path: &gotPath, auth: &gotAuth,
		session: &gotSession, contentType: &gotContentType, body: &gotBody,
	}))
	defer upstream.Close()

	client := mustClient(t, newEnabledClientConfig(upstream.URL))

	response, err := client.OpenResponses(context.Background(), strings.NewReader(payload), "application/json", "ses_fixture")
	if err != nil {
		t.Fatalf("OpenResponses() returned error: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("upstream status = %d, want 200", response.StatusCode)
	}

	assertions := []struct {
		name, got, want string
	}{{"method", gotMethod, http.MethodPost},
		{"path", gotPath, "/openai/v1/responses"},
		{"authorization", gotAuth, "Bearer " + testCredential},
		{"session", gotSession, "ses_fixture"},
		{"content type", gotContentType, "application/json"},
		{"body", gotBody, payload},
	}
	for _, item := range assertions {
		if item.got != item.want {
			t.Errorf("%s = %q, want %q", item.name, item.got, item.want)
		}
	}

	if strings.Contains(strings.ToLower(gotBody), "previous_response_id") {
		t.Errorf("adapter must not inject previous_response_id; saw it in body %q", gotBody)
	}
}

func TestListModelsUsesVendorSpecificPath(t *testing.T) {
	var gotPath string

	upstream := httptest.NewServer(sinkHandler(&upstreamCapture{path: &gotPath}))
	defer upstream.Close()

	client := mustClient(t, newEnabledClientConfig(upstream.URL))

	response, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() returned error: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("upstream status = %d, want 200", response.StatusCode)
	}

	if gotPath != "/v1/models" {
		t.Fatalf("models path = %q, want /v1/models", gotPath)
	}
}

func TestClientTransportErrorNeverContainsCredential(t *testing.T) {
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closedURL := closed.URL
	closed.Close()

	client := mustClient(t, newEnabledClientConfig(closedURL))

	_, err := client.OpenResponses(context.Background(), strings.NewReader("{}"), "application/json", "ses_fixture")
	if err == nil {
		t.Fatalf("expected transport error for unreachable upstream")
	}

	if strings.Contains(err.Error(), testCredential) {
		t.Fatalf("error %v leaks the credential", err)
	}
}
