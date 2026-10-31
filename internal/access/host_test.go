package access_test

import (
	"strings"
	"testing"

	"aihub/internal/access"
)

func TestParseAllowedHosts(t *testing.T) {
	got, err := access.ParseAllowedHosts(" LocalHost. ,127.0.0.1,[::1],localhost,EXAMPLE.test. ")
	if err != nil {
		t.Fatalf("ParseAllowedHosts() returned error: %v", err)
	}

	want := []string{"localhost", "127.0.0.1", "::1", "example.test"}
	if len(got) != len(want) {
		t.Fatalf("ParseAllowedHosts() = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ParseAllowedHosts() = %v, want %v", got, want)
		}
	}
}

func TestParseAllowedHostsRejectsInvalidEntries(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "empty", value: ""},
		{name: "empty entry", value: "localhost,,127.0.0.1"},
		{name: "wildcard", value: "*.example.com"},
		{name: "port", value: "example.com:8080"},
		{name: "scheme", value: "http://example.com"},
		{name: "user info", value: "user@example.com"},
		{name: "path", value: "example.com/path"},
		{name: "query", value: "example.com?x=1"},
		{name: "too many entries", value: manyHosts(65)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := access.ParseAllowedHosts(test.value); err == nil {
				t.Fatalf("ParseAllowedHosts(%q) must fail", test.value)
			}
		})
	}
}

func manyHosts(count int) string {
	entries := make([]string, count)
	for i := range entries {
		entries[i] = "host" + strings.Repeat("a", 3) + string(rune('a'+i%26)) + ".example"
	}

	return strings.Join(entries, ",")
}
