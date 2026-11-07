package providers

import (
	"encoding/json"
	"strings"
	"unicode"
)

const proxyUpstreamDetailMaxRunes = 500

type proxyUpstreamEnvelope struct {
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// extractProxyUpstreamDetail parses common Connexto Proxy JSON error bodies.
// code is the top-level "status" field when present (e.g. browser_unavailable).
func extractProxyUpstreamDetail(raw []byte) (detail string, code string) {
	s := strings.TrimSpace(sanitizeProxyUpstreamSnippet(string(raw)))
	if s == "" {
		return "", ""
	}
	var envelope proxyUpstreamEnvelope
	if err := json.Unmarshal(raw, &envelope); err == nil {
		code = strings.TrimSpace(envelope.Status)
		if envelope.Error != nil && strings.TrimSpace(envelope.Error.Message) != "" {
			return truncateProxyUpstreamDetail(strings.TrimSpace(envelope.Error.Message)), code
		}
		if strings.TrimSpace(envelope.Message) != "" {
			return truncateProxyUpstreamDetail(strings.TrimSpace(envelope.Message)), code
		}
	}
	return truncateProxyUpstreamDetail(s), code
}

func sanitizeProxyUpstreamSnippet(s string) string {
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

func truncateProxyUpstreamDetail(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= proxyUpstreamDetailMaxRunes {
		return string(runes)
	}
	return string(runes[:proxyUpstreamDetailMaxRunes]) + "…"
}
