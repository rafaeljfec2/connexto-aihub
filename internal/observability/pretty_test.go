package observability_test

import (
	"bytes"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"aihub/internal/observability"
)

var timestampPattern = regexp.MustCompile(`^\d{2}:\d{2}:\d{2}\.\d{3} `)

func prettyLine(t *testing.T, log func(*slog.Logger)) string {
	t.Helper()

	var buffer bytes.Buffer
	log(observability.NewLogger(&buffer, slog.LevelDebug, "pretty"))

	return buffer.String()
}

func TestPrettyFormatIsOneReadableLinePerRecord(t *testing.T) {
	output := prettyLine(t, func(logger *slog.Logger) {
		logger.Info("ai hub started", "version", "0.1.0-dev", "port", 8080)
	})

	if strings.Count(output, "\n") != 1 || !strings.HasSuffix(output, "\n") {
		t.Fatalf("output must be exactly one line, got %q", output)
	}

	line := strings.TrimSuffix(output, "\n")
	if !timestampPattern.MatchString(line) {
		t.Fatalf("line does not start with a time: %q", line)
	}

	if !strings.Contains(line, "INFO  ai hub started") || !strings.Contains(line, "version=0.1.0-dev port=8080") {
		t.Fatalf("line = %q", line)
	}
}

func TestPrettyFormatNeverSplitsRecordsWithControlCharacters(t *testing.T) {
	output := prettyLine(t, func(logger *slog.Logger) {
		logger.Warn("line one\nforged INFO entry", "detail", "a\r\nb c")
	})

	if strings.Count(output, "\n") != 1 {
		t.Fatalf("a control character split the record: %q", output)
	}

	if !strings.Contains(output, `"line one\nforged INFO entry"`) || !strings.Contains(output, `detail="a\r\nb c"`) {
		t.Fatalf("values were not escaped: %q", output)
	}
}

func TestPrettyFormatRendersGroupsAndPreformattedAttributes(t *testing.T) {
	output := prettyLine(t, func(logger *slog.Logger) {
		logger.With("provider", "opencode-go").WithGroup("http").Error("request failed",
			"status", 502, slog.Group("upstream", "host", "opencode.ai"), "err", errors.New("timeout"))
	})

	for _, fragment := range []string{"ERROR request failed", "provider=opencode-go", "http.status=502",
		"http.upstream.host=opencode.ai", `http.err=timeout`} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("missing %q in %q", fragment, output)
		}
	}
}

func TestPrettyFormatHonoursLevelAndWritesNoColorToBuffers(t *testing.T) {
	output := prettyLine(t, func(logger *slog.Logger) {
		logger.Debug("visible at debug")
	})

	if !strings.Contains(output, "DEBUG visible at debug") {
		t.Fatalf("debug record missing: %q", output)
	}

	if strings.Contains(output, "\x1b[") {
		t.Fatalf("a non-terminal buffer received ANSI sequences: %q", output)
	}

	var buffer bytes.Buffer
	observability.NewLogger(&buffer, slog.LevelWarn, "pretty").Info("filtered")

	if buffer.Len() != 0 {
		t.Fatalf("info record written below the warn level: %q", buffer.String())
	}
}
