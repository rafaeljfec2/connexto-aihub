package access_test

import (
	"regexp"
	"strings"
	"testing"

	"aihub/internal/access"
)

func TestNewSessionID(t *testing.T) {
	first, err := access.NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID() returned error: %v", err)
	}

	second, err := access.NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID() returned error: %v", err)
	}

	if first == second {
		t.Fatal("two generated session IDs must differ")
	}

	pattern := regexp.MustCompile(`^aihub-ses-[0-9a-f]{32}$`)
	if !pattern.MatchString(first) {
		t.Fatalf("session ID = %q, want aihub-ses- followed by 32 lowercase hex characters", first)
	}
}

func TestValidSessionID(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "single character", value: "a", want: true},
		{name: "maximum length", value: strings.Repeat("a", 128), want: true},
		{name: "allowed punctuation", value: "ses_context-1.2:3", want: true},
		{name: "empty", value: "", want: false},
		{name: "too long", value: strings.Repeat("a", 129), want: false},
		{name: "space", value: "ses context", want: false},
		{name: "slash", value: "ses/context", want: false},
		{name: "backslash", value: `ses\context`, want: false},
		{name: "comma", value: "ses,context", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := access.ValidSessionID(test.value); got != test.want {
				t.Fatalf("ValidSessionID(%q) = %v, want %v", test.value, got, test.want)
			}
		})
	}
}
