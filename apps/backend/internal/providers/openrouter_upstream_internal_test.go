package providers

import (
	"strings"
	"testing"
)

func TestExtractOpenRouterNestedProviderMessage(t *testing.T) {
	raw := []byte(`{"error":{"message":"Provider returned error","code":400,"metadata":{"raw":"{\"error\":{\"message\":\"messages.1.content.0: Invalid data in redacted_thinking block\"}}","provider_name":"Google"}}}`)
	detail := extractOpenRouterUpstreamDetail(raw)
	if !strings.Contains(detail, "Invalid data in redacted_thinking block") {
		t.Fatalf("detail = %q", detail)
	}
	if !strings.Contains(detail, "provider=Google") {
		t.Fatalf("detail = %q", detail)
	}
	if strings.Contains(detail, "previous_errors") {
		t.Fatal("detail should not contain raw metadata blob")
	}
}
