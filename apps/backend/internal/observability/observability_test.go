package observability_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"aihub/internal/observability"
)

func TestNewCorrelationIDIsValidAndUnique(t *testing.T) {
	seen := make(map[string]struct{}, 100)

	for range 100 {
		id, err := observability.NewCorrelationID()
		if err != nil {
			t.Fatalf("NewCorrelationID() returned error: %v", err)
		}

		if !observability.IsValidCorrelationID(id) {
			t.Fatalf("generated id %q is not valid", id)
		}

		if _, exists := seen[id]; exists {
			t.Fatalf("generated duplicate id %q", id)
		}

		seen[id] = struct{}{}
	}
}

func TestIsValidCorrelationID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{name: "hex identifier", id: "8f14e45fceea167a", want: true},
		{name: "uuid style", id: "123e4567-e89b-12d3-a456-426614174000", want: true},
		{name: "too short", id: "abc", want: false},
		{name: "contains secret punctuation", id: "abc def ghi jkl", want: false},
		{name: "newline injection", id: "12345678\nforged", want: false},
		{name: "empty", id: "", want: false},
		{name: "too long", id: strings.Repeat("a", 129), want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := observability.IsValidCorrelationID(test.id); got != test.want {
				t.Fatalf("IsValidCorrelationID(%q) = %v, want %v", test.id, got, test.want)
			}
		})
	}
}

func TestCorrelationIDContextRoundTrip(t *testing.T) {
	ctx := observability.WithCorrelationID(context.Background(), "abc12345")

	if got, want := observability.CorrelationIDFrom(ctx), "abc12345"; got != want {
		t.Fatalf("CorrelationIDFrom() = %q, want %q", got, want)
	}
}

func TestCorrelationIDFromEmptyContext(t *testing.T) {
	if got := observability.CorrelationIDFrom(context.Background()); got != "" {
		t.Fatalf("CorrelationIDFrom() = %q, want empty string", got)
	}
}

func TestNewLoggerWritesJSON(t *testing.T) {
	var buffer bytes.Buffer
	logger := observability.NewLogger(&buffer, slog.LevelInfo, "json")

	logger.Info("hello", "key", "value")

	var record map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &record); err != nil {
		t.Fatalf("log output is not valid JSON: %v (%q)", err, buffer.String())
	}

	if record["msg"] != "hello" {
		t.Fatalf("msg = %v, want hello", record["msg"])
	}

	if record["key"] != "value" {
		t.Fatalf("key = %v, want value", record["key"])
	}
}

func TestNewLoggerHonoursLevel(t *testing.T) {
	var buffer bytes.Buffer
	logger := observability.NewLogger(&buffer, slog.LevelWarn, "json")

	logger.Info("suppressed")
	if buffer.Len() != 0 {
		t.Fatalf("info entry must be suppressed at warn level, got %q", buffer.String())
	}

	logger.Warn("emitted")
	if !strings.Contains(buffer.String(), "emitted") {
		t.Fatalf("warn entry must be emitted, got %q", buffer.String())
	}
}

func TestNewLoggerSupportsTextFormat(t *testing.T) {
	var buffer bytes.Buffer
	logger := observability.NewLogger(&buffer, slog.LevelInfo, "text")

	logger.Info("hello text")

	if !strings.Contains(buffer.String(), "msg=\"hello text\"") {
		t.Fatalf("text format output = %q", buffer.String())
	}
}
