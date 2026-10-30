// Package observability provides structured logging and request correlation.
package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"regexp"
)

type correlationIDKey struct{}

// correlationIDPattern restricts identifiers to characters that are safe to
// propagate through headers and logs, preventing log injection.
var correlationIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)

// IsValidCorrelationID reports whether id is safe to propagate.
func IsValidCorrelationID(id string) bool {
	return correlationIDPattern.MatchString(id)
}

// NewCorrelationID returns a random 128-bit identifier encoded as hexadecimal.
func NewCorrelationID() (string, error) {
	var buffer [16]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return "", fmt.Errorf("generate correlation id: %w", err)
	}

	return hex.EncodeToString(buffer[:]), nil
}

// WithCorrelationID stores the identifier in the context.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey{}, id)
}

// CorrelationIDFrom returns the identifier stored in the context, if any.
func CorrelationIDFrom(ctx context.Context) string {
	id, ok := ctx.Value(correlationIDKey{}).(string)
	if !ok {
		return ""
	}

	return id
}

// NewLogger builds a slog logger that writes structured records to w.
// The format argument is expected to be validated by the config package.
func NewLogger(w io.Writer, level slog.Level, format string) *slog.Logger {
	options := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(w, options)
	} else {
		handler = slog.NewJSONHandler(w, options)
	}

	return slog.New(handler)
}
