package providers

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

const proxyAntigravityAuthUpstreamDetailMaxRunes = 500

// ProxyAntigravityAuthFailedError is returned when the Connexto Proxy Antigravity
// auth call fails (non-2xx or unusable response). Detail is safe for client-facing
// messages (sanitized, length-capped). Use errors.Is(err, ErrProxyAntigravityAuthFailed).
type ProxyAntigravityAuthFailedError struct {
	Status int
	Detail string
}

func (e *ProxyAntigravityAuthFailedError) Error() string {
	if e == nil {
		return ErrProxyAntigravityAuthFailed.Error()
	}
	if e.Detail != "" {
		return fmt.Sprintf("%v: upstream status %d: %s", ErrProxyAntigravityAuthFailed, e.Status, e.Detail)
	}
	return fmt.Sprintf("%v: upstream status %d", ErrProxyAntigravityAuthFailed, e.Status)
}

// Is implements errors.Is so callers can use errors.Is(err, ErrProxyAntigravityAuthFailed).
func (e *ProxyAntigravityAuthFailedError) Is(target error) bool {
	return target == ErrProxyAntigravityAuthFailed
}

func newProxyAntigravityAuthFailed(status int, body []byte) error {
	return &ProxyAntigravityAuthFailedError{
		Status: status,
		Detail: extractProxyAntigravityAuthUpstreamDetail(body),
	}
}

func extractProxyAntigravityAuthUpstreamDetail(raw []byte) string {
	s := strings.TrimSpace(sanitizeProxyAntigravityAuthUpstreamSnippet(string(raw)))
	if s == "" {
		return ""
	}
	var envelope struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil {
		if envelope.Error != nil && strings.TrimSpace(envelope.Error.Message) != "" {
			return truncateProxyAntigravityAuthDetail(strings.TrimSpace(envelope.Error.Message))
		}
		if strings.TrimSpace(envelope.Message) != "" {
			return truncateProxyAntigravityAuthDetail(strings.TrimSpace(envelope.Message))
		}
	}
	return truncateProxyAntigravityAuthDetail(s)
}

func sanitizeProxyAntigravityAuthUpstreamSnippet(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			b.WriteRune(' ')
		case unicode.IsControl(r):
			continue
		default:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func truncateProxyAntigravityAuthDetail(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= proxyAntigravityAuthUpstreamDetailMaxRunes {
		return string(runes)
	}
	return string(runes[:proxyAntigravityAuthUpstreamDetailMaxRunes]) + "…"
}
