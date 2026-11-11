package access

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

// ParseAllowedOrigins parses a comma-separated exact browser-origin allowlist.
// An empty value is valid and allows only requests without an Origin header.
func ParseAllowedOrigins(value string) ([]string, error) {
	if value == "" {
		return nil, nil
	}

	entries := strings.Split(value, ",")
	if len(entries) > maxAllowlistEntries {
		return nil, errors.New("origin allowlist must contain at most 64 entries")
	}

	seen := make(map[string]struct{}, len(entries))
	origins := make([]string, 0, len(entries))
	for _, entry := range entries {
		normalized, err := canonicalOrigin(strings.TrimSpace(entry))
		if err != nil {
			return nil, err
		}

		if _, duplicate := seen[normalized]; duplicate {
			continue
		}

		seen[normalized] = struct{}{}
		origins = append(origins, normalized)
	}

	if len(origins) == 0 {
		return nil, errors.New("origin allowlist must not be empty when present")
	}

	return origins, nil
}

func canonicalOrigin(value string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value {
		return "", errors.New("origin must be an absolute http or https origin")
	}

	if strings.Contains(value, "#") {
		return "", errors.New("origin must not contain a fragment")
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", errors.New("origin is malformed")
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", errors.New("origin scheme must be http or https")
	}

	if parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.Path != "" {
		return "", errors.New("origin must not contain user information, path, query or fragment")
	}

	if strings.HasSuffix(parsed.Host, ":") {
		return "", errors.New("origin port must not be empty")
	}

	host, ok := normalizeHost(parsed.Hostname())
	if !ok {
		return "", errors.New("origin host is invalid")
	}

	port := parsed.Port()
	if port != "" {
		if !validPort(port) {
			return "", errors.New("origin port is invalid")
		}

		if isDefaultPort(scheme, port) {
			port = ""
		}
	}

	authority := host
	if net.ParseIP(host) != nil && strings.Contains(host, ":") {
		authority = "[" + host + "]"
	}

	if port != "" {
		authority += ":" + port
	}

	return scheme + "://" + authority, nil
}

func isDefaultPort(scheme, port string) bool {
	return scheme == "http" && port == "80" || scheme == "https" && port == "443"
}
