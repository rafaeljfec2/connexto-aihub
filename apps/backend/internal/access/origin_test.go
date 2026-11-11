package access_test

import (
	"strings"
	"testing"

	"aihub/internal/access"
)

func TestParseAllowedOrigins(t *testing.T) {
	got, err := access.ParseAllowedOrigins("HTTPS://Example.COM:443, http://example.com:80,https://example.com:8443,https://example.com:443")
	if err != nil {
		t.Fatalf("ParseAllowedOrigins() returned error: %v", err)
	}

	want := []string{"https://example.com", "http://example.com", "https://example.com:8443"}
	if len(got) != len(want) {
		t.Fatalf("ParseAllowedOrigins() = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ParseAllowedOrigins() = %v, want %v", got, want)
		}
	}
}

func TestParseAllowedOriginsAllowsEmpty(t *testing.T) {
	got, err := access.ParseAllowedOrigins("")
	if err != nil {
		t.Fatalf("ParseAllowedOrigins() returned error: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("ParseAllowedOrigins() = %v, want empty", got)
	}
}

func TestParseAllowedOriginsRejectsInvalidEntries(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "null", value: "null"},
		{name: "wildcard", value: "*"},
		{name: "path", value: "https://example.com/path"},
		{name: "user info", value: "https://user@example.com"},
		{name: "query", value: "https://example.com?x=1"},
		{name: "fragment", value: "https://example.com#x"},
		{name: "empty fragment", value: "https://example.com#"},
		{name: "empty query", value: "https://example.com?"},
		{name: "empty query and fragment", value: "https://example.com?#"},
		{name: "empty port", value: "https://example.com:"},
		{name: "ftp", value: "ftp://example.com"},
		{name: "missing host", value: "https://"},
		{name: "too many entries", value: manyOrigins(65)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := access.ParseAllowedOrigins(test.value); err == nil {
				t.Fatalf("ParseAllowedOrigins(%q) must fail", test.value)
			}
		})
	}
}

func manyOrigins(count int) string {
	entries := make([]string, count)
	for i := range entries {
		entries[i] = "https://host" + strings.Repeat("a", 3) + string(rune('a'+i%26)) + ".example"
	}

	return strings.Join(entries, ",")
}
