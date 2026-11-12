package userauth

import (
	"fmt"
	"strings"
)

func formatAccessCookie(token string, secure bool) string {
	return formatCookie(AccessCookieName, token, "/", int(AccessTTL.Seconds()), secure)
}

func formatRefreshCookie(token string, secure bool) string {
	return formatCookie(RefreshCookieName, token, "/api/ui", int(RefreshTTL.Seconds()), secure)
}

func clearAccessCookie(secure bool) string {
	return formatCookie(AccessCookieName, "", "/", 0, secure)
}

func clearRefreshCookie(secure bool) string {
	return formatCookie(RefreshCookieName, "", "/api/ui", 0, secure)
}

func formatCookie(name, value, path string, maxAge int, secure bool) string {
	parts := []string{
		fmt.Sprintf("%s=%s", name, value),
		fmt.Sprintf("Path=%s", path),
		"HttpOnly",
		"SameSite=Strict",
	}
	if maxAge >= 0 {
		parts = append(parts, fmt.Sprintf("Max-Age=%d", maxAge))
	}
	if secure {
		parts = append(parts, "Secure")
	}
	return strings.Join(parts, "; ")
}
