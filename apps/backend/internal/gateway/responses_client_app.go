package gateway

import (
	"net/http"
	"strings"

	"aihub/internal/generations"
)

const (
	// ClientAppHeader is the explicit AI Hub header used by callers to report the originating app.
	ClientAppHeader = "X-AIHub-App"
)

var clientAppHeaders = []string{
	ClientAppHeader,
	"X-Title",
	"Originator",
}

func clientAppFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	for _, header := range clientAppHeaders {
		if app := generations.SafeClientApp(r.Header.Get(header)); app != nil {
			return *app
		}
	}
	if app := generations.SafeClientApp(userAgentProductToken(r.UserAgent())); app != nil {
		return *app
	}
	return ""
}

func userAgentProductToken(userAgent string) string {
	fields := strings.Fields(strings.TrimSpace(userAgent))
	if len(fields) == 0 {
		return ""
	}
	product, _, _ := strings.Cut(fields[0], "/")
	return strings.TrimSpace(product)
}

