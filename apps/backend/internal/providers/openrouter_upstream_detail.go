package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const (
	openRouterUpstreamBodyLimit       = 8 << 10
	InternalUpstreamErrorDetailHeader = "X-AIHub-Internal-Upstream-Error-Detail"
	openRouterFallbackErrorMessage    = "OpenRouter request failed"
)

var (
	openRouterBearerPattern = regexp.MustCompile(`(?i)Bearer\s+\S+`)
	openRouterKeyPattern    = regexp.MustCompile(`sk-or-[A-Za-z0-9_-]+`)
	openRouterSecretPattern = regexp.MustCompile(`sk-[A-Za-z0-9_-]+`)
)

type openRouterUpstreamError struct {
	Code     json.RawMessage          `json:"code"`
	Type     string                   `json:"type"`
	Message  string                   `json:"message"`
	Metadata *openRouterUpstreamMeta  `json:"metadata"`
}

type openRouterUpstreamMeta struct {
	ProviderCode json.RawMessage `json:"provider_code"`
	Raw          string          `json:"raw"`
	ProviderName string          `json:"provider_name"`
}

type openRouterUpstreamEnvelope struct {
	Error *openRouterUpstreamError `json:"error"`
}

// extractOpenRouterUpstreamDetail parses a bounded OpenRouter error JSON body into a
// single human-readable summary for redacted client and log surfaces.
func extractOpenRouterUpstreamDetail(raw []byte) string {
	raw = bytesTrim(raw)
	if len(raw) == 0 {
		return ""
	}

	var envelope openRouterUpstreamEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return truncateProxyUpstreamDetail(sanitizeProxyUpstreamSnippet(string(raw)))
	}
	if envelope.Error == nil {
		return ""
	}

	inner := nestedOpenRouterProviderMessage(envelope.Error.Metadata)
	outer := strings.TrimSpace(envelope.Error.Message)
	message := outer
	if inner != "" && (message == "" || isGenericOpenRouterMessage(message)) {
		message = inner
	}

	parts := make([]string, 0, 4)
	if message != "" {
		parts = append(parts, message)
	}
	if code := openRouterCodeLabel(envelope.Error.Code); code != "" {
		parts = append(parts, code)
	} else if typ := strings.TrimSpace(envelope.Error.Type); typ != "" {
		parts = append(parts, fmt.Sprintf("type=%s", typ))
	}
	if envelope.Error.Metadata != nil {
		if name := strings.TrimSpace(envelope.Error.Metadata.ProviderName); name != "" {
			parts = append(parts, fmt.Sprintf("provider=%s", name))
		}
		if len(envelope.Error.Metadata.ProviderCode) > 0 {
			var providerCode string
			if err := json.Unmarshal(envelope.Error.Metadata.ProviderCode, &providerCode); err == nil && strings.TrimSpace(providerCode) != "" {
				parts = append(parts, fmt.Sprintf("provider_code=%s", strings.TrimSpace(providerCode)))
			}
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return truncateProxyUpstreamDetail(strings.Join(parts, "; "))
}

func isGenericOpenRouterMessage(message string) bool {
	switch strings.ToLower(strings.TrimSpace(message)) {
	case "provider returned error", "providerreturned error":
		return true
	default:
		return false
	}
}

func openRouterCodeLabel(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil && strings.TrimSpace(asString) != "" {
		return fmt.Sprintf("code=%s", strings.TrimSpace(asString))
	}
	var asNumber float64
	if err := json.Unmarshal(raw, &asNumber); err == nil {
		return fmt.Sprintf("code=%s", strconv.FormatFloat(asNumber, 'f', -1, 64))
	}
	return ""
}

func nestedOpenRouterProviderMessage(meta *openRouterUpstreamMeta) string {
	if meta == nil {
		return ""
	}
	raw := strings.TrimSpace(meta.Raw)
	if raw == "" {
		return ""
	}
	var envelope struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return ""
	}
	if msg := strings.TrimSpace(envelope.Error.Message); msg != "" {
		return msg
	}
	return strings.TrimSpace(envelope.Message)
}

func redactOpenRouterSensitive(detail string) string {
	if detail == "" {
		return ""
	}
	detail = openRouterBearerPattern.ReplaceAllString(detail, "Bearer [redacted]")
	detail = openRouterKeyPattern.ReplaceAllString(detail, "sk-or-[redacted]")
	detail = openRouterSecretPattern.ReplaceAllString(detail, "sk-[redacted]")
	return detail
}

func buildOpenRouterSanitizedErrorBody(detail string) string {
	message := openRouterFallbackErrorMessage
	if detail != "" {
		if idx := strings.Index(detail, ";"); idx > 0 {
			message = strings.TrimSpace(detail[:idx])
		} else {
			message = detail
		}
	}
	payload := map[string]any{
		"error": map[string]any{
			"type":    "upstream_error",
			"message": message,
		},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return openRouterErrorBody
	}
	return string(encoded)
}

func sanitizeOpenRouterErrorResponse(response *http.Response) {
	if response == nil || response.StatusCode < http.StatusMultipleChoices {
		return
	}

	var raw []byte
	if response.Body != nil {
		var readErr error
		raw, readErr = io.ReadAll(io.LimitReader(response.Body, openRouterUpstreamBodyLimit))
		_ = response.Body.Close()
		if readErr != nil {
			raw = nil
		}
	}

	detail := redactOpenRouterSensitive(extractOpenRouterUpstreamDetail(raw))
	body := buildOpenRouterSanitizedErrorBody(detail)
	response.Header.Del(InternalUpstreamErrorDetailHeader)
	if detail != "" {
		response.Header.Set(InternalUpstreamErrorDetailHeader, detail)
	}
	response.Body = io.NopCloser(strings.NewReader(body))
	response.ContentLength = int64(len(body))
	response.Header.Set("Content-Type", "application/json")
}

func bytesTrim(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}
