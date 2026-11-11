package records

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCursorRoundTrips(t *testing.T) {
	cursor := Cursor{CompletedAt: at(30), ExecutionID: executionID(99)}

	encoded := encodeCursor(cursor)
	if strings.ContainsAny(encoded, "+/=") {
		t.Fatalf("cursor %q must be unpadded base64url", encoded)
	}

	decoded, err := decodeCursor(encoded)
	if err != nil {
		t.Fatalf("decodeCursor() returned error: %v", err)
	}

	if !decoded.CompletedAt.Equal(cursor.CompletedAt) || decoded.ExecutionID != cursor.ExecutionID {
		t.Fatalf("decoded cursor = %+v, want %+v", decoded, cursor)
	}
}

func TestDecodeCursorRejectsMalformedValues(t *testing.T) {
	valid := encodeCursor(Cursor{CompletedAt: at(1), ExecutionID: executionID(1)})

	oversized := base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("x", 65)))
	badTimestamp := base64.RawURLEncoding.EncodeToString([]byte("not-a-time\n" + executionID(1)))
	badID := base64.RawURLEncoding.EncodeToString([]byte(at(1).Format(time.RFC3339Nano) + "\nshort"))
	extraSeparator := base64.RawURLEncoding.EncodeToString([]byte(at(1).Format(time.RFC3339Nano) + "\n" + executionID(1) + "\nextra"))
	offsetTimestamp := base64.RawURLEncoding.EncodeToString([]byte("2026-10-31T12:00:01+01:00\n" + executionID(1)))

	tests := []struct {
		name  string
		value string
	}{
		{name: "invalid base64", value: "%%%"},
		{name: "empty", value: ""},
		{name: "oversized", value: oversized},
		{name: "bad timestamp", value: badTimestamp},
		{name: "bad execution id", value: badID},
		{name: "extra separator", value: extraSeparator},
		{name: "offset timestamp", value: offsetTimestamp},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeCursor(test.value); !errors.Is(err, ErrInvalidCursor) {
				t.Fatalf("decodeCursor() error = %v, want ErrInvalidCursor", err)
			}
		})
	}

	if _, err := decodeCursor(valid); err != nil {
		t.Fatalf("valid cursor rejected: %v", err)
	}
}
