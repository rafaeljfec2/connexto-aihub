package db

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// PingTimeout bounds health checks and the module ping; larger than the
// shortest meaningful network failure window, smaller than any operator
// expectation.
const pingTimeout = 3 * time.Second

// errf wraps fmt.Errorf as a local helper to keep call sites terse.
func errf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

// RedactDSN masks credentials of a URL-form DSN for safe reporting.
func RedactDSN(dsn string) string {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "postgres://<unparsable-dsn>"
	}

	if parsed.User != nil {
		_, hasPassword := parsed.User.Password()
		if hasPassword {
			parsed.User = url.UserPassword(parsed.User.Username(), "xxxx")
		}
	}

	return parsed.String()
}

// SafeError reports an error whose message never contains credentials: any
// occurrence of the exact DSN is replaced by its redacted form, and any
// `password=` parameter fragment is masked. Parse errors of pgconn duplicate
// the whole DSN, so message-level scrubbing is mandatory (ADR-020).
func SafeError(dsn string, err error) error {
	if err == nil {
		return nil
	}

	message := err.Error()
	if dsn != "" {
		message = strings.ReplaceAll(message, dsn, RedactDSN(dsn))
	}

	message = maskPasswordParameters(message)

	return fmt.Errorf("%s", message)
}

// maskPasswordParameters replaces password values in `password=...` shapes
// (query form or keyword form) with the redact marker.
func maskPasswordParameters(message string) string {
	const marker = "<redacted>"
	var builder strings.Builder
	for offset := 0; offset < len(message); {
		index := strings.Index(message[offset:], "password=")
		if index < 0 {
			builder.WriteString(message[offset:])
			break
		}

		start := offset + index
		builder.WriteString(message[offset:start])
		builder.WriteString("password=")

		valueStart := start + len("password=")
		valueEnd := valueStart
		for valueEnd < len(message) && !strings.ContainsRune(" \t\n,&", rune(message[valueEnd])) {
			valueEnd++
		}

		builder.WriteString(marker)
		offset = valueEnd
	}

	return builder.String()
}
