package access

import (
	"errors"
	"net"
	"strings"
)

const maxAllowlistEntries = 64

// ParseAllowedHosts parses a comma-separated exact host allowlist.
func ParseAllowedHosts(value string) ([]string, error) {
	entries := strings.Split(value, ",")
	if len(entries) > maxAllowlistEntries {
		return nil, errors.New("host allowlist must contain at most 64 entries")
	}

	seen := make(map[string]struct{}, len(entries))
	hosts := make([]string, 0, len(entries))
	for _, entry := range entries {
		normalized, err := normalizeConfiguredHost(entry)
		if err != nil {
			return nil, err
		}

		if _, duplicate := seen[normalized]; duplicate {
			continue
		}

		seen[normalized] = struct{}{}
		hosts = append(hosts, normalized)
	}

	if len(hosts) == 0 {
		return nil, errors.New("host allowlist must not be empty")
	}

	return hosts, nil
}

func normalizeConfiguredHost(value string) (string, error) {
	entry := strings.TrimSpace(value)
	if entry == "" {
		return "", errors.New("host allowlist contains an empty entry")
	}

	if strings.HasPrefix(entry, "[") && strings.HasSuffix(entry, "]") {
		entry = entry[1 : len(entry)-1]
	}

	host, ok := normalizeHost(entry)
	if !ok {
		return "", errors.New("host allowlist contains an invalid host")
	}

	return host, nil
}

// normalizeRequestHost strips a valid port, brackets, case and a trailing dot.
func normalizeRequestHost(value string) (string, bool) {
	if value == "" || strings.TrimSpace(value) != value {
		return "", false
	}

	host := value
	if strings.HasPrefix(host, "[") {
		closing := strings.Index(host, "]")
		if closing < 0 {
			return "", false
		}

		remainder := host[closing+1:]
		if remainder != "" {
			if !strings.HasPrefix(remainder, ":") || !validPort(remainder[1:]) {
				return "", false
			}
		}

		host = host[1:closing]
	} else if strings.Count(host, ":") == 1 {
		hostPart, port, found := strings.Cut(host, ":")
		if !found || !validPort(port) {
			return "", false
		}
		host = hostPart
	}

	return normalizeHost(host)
}

func normalizeHost(value string) (string, bool) {
	if value == "" || strings.TrimSpace(value) != value {
		return "", false
	}

	for index := 0; index < len(value); index++ {
		char := value[index]
		if char == '@' || char == '/' || char == '\\' || char == '?' ||
			char == '#' || char == '*' || char == '%' || char <= ' ' || char == 0x7f {
			return "", false
		}
	}

	host := strings.ToLower(strings.TrimSuffix(value, "."))
	if host == "" {
		return "", false
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.String(), true
	}

	if strings.Contains(host, ":") || len(host) > 253 {
		return "", false
	}

	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", false
		}

		for index := 0; index < len(label); index++ {
			char := label[index]
			valid := char >= 'a' && char <= 'z' ||
				char >= '0' && char <= '9' || char == '-'
			if !valid {
				return "", false
			}
		}
	}

	return host, true
}

func validPort(value string) bool {
	if value == "" {
		return false
	}

	port := 0
	for index := 0; index < len(value); index++ {
		char := value[index]
		if char < '0' || char > '9' {
			return false
		}

		port = port*10 + int(char-'0')
		if port > 65535 {
			return false
		}
	}

	return true
}
