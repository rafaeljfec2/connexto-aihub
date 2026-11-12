package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"aihub/internal/generations"
	"aihub/internal/records"
)

func TestClientAppFromRequestPrecedenceAndFallback(t *testing.T) {
	cases := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{
			name: "explicit X-AIHub-App wins",
			headers: map[string]string{
				"X-AIHub-App": "ai-hub-playground",
				"X-Title":     "OpenCode",
				"Originator":  "codex_cli_rs",
				"User-Agent":  "curl/8.5.0",
			},
			want: "ai-hub-playground",
		},
		{
			name: "X-Title beats Originator and User-Agent",
			headers: map[string]string{
				"X-Title":    "OpenCode",
				"Originator": "codex_cli_rs",
				"User-Agent": "curl/8.5.0",
			},
			want: "OpenCode",
		},
		{
			name: "Originator beats User-Agent",
			headers: map[string]string{
				"Originator": "codex_cli_rs",
				"User-Agent": "Mozilla/5.0",
			},
			want: "codex_cli_rs",
		},
		{
			name: "User-Agent product token fallback",
			headers: map[string]string{
				"User-Agent": "codex_cli_rs/0.14.0 (Linux x86_64)",
			},
			want: "codex_cli_rs",
		},
		{
			name: "empty headers return empty",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}
			if got := clientAppFromRequest(req); got != tc.want {
				t.Fatalf("clientAppFromRequest() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNewGenerationListDTOMapsClientApp(t *testing.T) {
	app := "codex_cli_rs"
	dto := newGenerationListDTO(generations.Generation{
		Record: records.Record{
			ExecutionID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ProviderID:  "cursor",
			Outcome:     "completed",
			Phase:       "forwarding",
		},
		Telemetry: generations.Telemetry{
			ClientApp: &app,
		},
	})
	if dto.Telemetry.ClientApp == nil || *dto.Telemetry.ClientApp != app {
		t.Fatalf("Telemetry.ClientApp = %v, want %q", dto.Telemetry.ClientApp, app)
	}
}

