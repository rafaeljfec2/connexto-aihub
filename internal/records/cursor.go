package records

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

const maxCursorBytes = 64

// ErrInvalidCursor marks opaque values that do not match the cursor contract.
var ErrInvalidCursor = errors.New("invalid cursor")

// Cursor is the stable page boundary: completed_at plus execution_id.
type Cursor struct {
	CompletedAt time.Time
	ExecutionID string
}

// ParseCursor decodes an opaque, unpadded base64url cursor.
func ParseCursor(value string) (Cursor, error) {
	return decodeCursor(value)
}

func encodeCursor(cursor Cursor) string {
	payload := formatTimestamp(cursor.CompletedAt) + "\n" + cursor.ExecutionID

	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func decodeCursor(value string) (Cursor, error) {
	if value == "" {
		return Cursor{}, invalidCursor("cursor is empty")
	}

	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, invalidCursor("cursor is not unpadded base64url")
	}
	if len(payload) == 0 || len(payload) > maxCursorBytes {
		return Cursor{}, invalidCursor("cursor payload length is outside its bounds")
	}

	parts := strings.Split(string(payload), "\n")
	if len(parts) != 2 {
		return Cursor{}, invalidCursor("cursor payload must contain exactly one separator")
	}

	completedAt, err := parseTimestamp(parts[0])
	if err != nil {
		return Cursor{}, invalidCursor("cursor timestamp is invalid")
	}
	if !validExecutionID(parts[1]) {
		return Cursor{}, invalidCursor("cursor execution_id is invalid")
	}

	return Cursor{CompletedAt: completedAt, ExecutionID: parts[1]}, nil
}

func invalidCursor(message string) error {
	return errors.Join(ErrInvalidCursor, errors.New(message))
}
