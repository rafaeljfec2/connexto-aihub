package gateway_test

import (
	"bytes"
	"log/slog"
	"sync"

	"aihub/internal/config"
)

// logBuffer is a concurrency-safe bytes buffer because the handler under test
// keeps logging from server goroutines while assertions read it.
type logBuffer struct {
	mu  sync.Mutex
	raw bytes.Buffer
}

func (b *logBuffer) Write(payload []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.raw.Write(payload)
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.raw.String()
}

func (b *logBuffer) contains(needle string) bool {
	return bytes.Contains([]byte(b.String()), []byte(needle))
}

// newTestLogger captures every record for redaction assertions.
func newTestLogger() (*slog.Logger, *logBuffer) {
	buffer := new(logBuffer)
	logger := slog.New(slog.NewTextHandler(buffer, &slog.HandlerOptions{Level: slog.LevelDebug}))

	return logger, buffer
}

// emptyProviderConfig builds a provider entry without a credential (D2).
func emptyProviderConfig() config.ProviderOpenCodeGo {
	return config.ProviderOpenCodeGo{BaseURL: "https://example.com/go"}
}
