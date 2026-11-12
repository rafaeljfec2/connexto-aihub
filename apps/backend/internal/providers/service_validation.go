package providers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"aihub/internal/config"
)

func (s *Service) validateIdentity(id, name, providerType string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("%w: id must be lowercase letters, digits and hyphens", ErrInvalid)
	}
	if err := validateName(strings.TrimSpace(name)); err != nil {
		return err
	}
	if !idPattern.MatchString(providerType) {
		return fmt.Errorf("%w: type must be lowercase letters, digits and hyphens", ErrInvalid)
	}
	return nil
}

func validateName(name string) error {
	if name == "" || len(name) > maxNameLength {
		return fmt.Errorf("%w: name must be 1 to %d characters", ErrInvalid, maxNameLength)
	}
	return nil
}

func (s *Service) validateEndpoint(endpoint string) error {
	if err := config.ValidateBaseURL(endpoint); err != nil {
		return fmt.Errorf("%w: endpoint is not a valid base URL", ErrInvalid)
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("%w: endpoint is not a valid base URL", ErrInvalid)
	}
	if strings.EqualFold(parsed.Scheme, "http") && !isLoopbackHost(parsed.Hostname()) && !s.isAllowedHTTPAddress(parsed.Host) {
		return fmt.Errorf("%w: endpoint must use https unless it targets a loopback host", ErrInvalid)
	}
	if _, allowed := s.allowedHosts[strings.ToLower(parsed.Hostname())]; !allowed {
		return fmt.Errorf("%w: endpoint host is not in AIHUB_PROVIDER_ALLOWED_HOSTS", ErrInvalid)
	}
	return nil
}

func (s *Service) isAllowedHTTPAddress(host string) bool {
	if len(s.allowedHTTPAddresses) == 0 {
		return false
	}
	_, allowed := s.allowedHTTPAddresses[strings.ToLower(strings.TrimSpace(host))]
	return allowed
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validateCredential(credential string) error {
	trimmed := strings.TrimSpace(credential)
	if trimmed == "" || len(trimmed) > maxCredentialLength {
		return fmt.Errorf("%w: credential must be 1 to %d characters", ErrInvalid, maxCredentialLength)
	}
	return nil
}

func (s *Service) generateUniqueID(ctx context.Context, name, providerType string) (string, error) {
	base := slugifyProviderName(name)
	if base == "" {
		base = slugifyProviderName(providerType)
	}
	if base == "" {
		base = "provider"
	}
	if len(base) > 54 {
		base = strings.TrimRight(base[:54], "-")
	}

	candidate := base
	for suffix := 2; suffix <= 999; suffix++ {
		_, err := s.repo.Get(ctx, candidate)
		if errors.Is(err, ErrNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
		candidate = base + "-" + strconv.Itoa(suffix)
	}
	return "", fmt.Errorf("%w: could not allocate unique provider id", ErrConflict)
}

func slugifyProviderName(value string) string {
	var b strings.Builder
	lastHyphen := true
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if unicode.IsSpace(r) || r == '-' || r == '_' || r == '.' || r == '/' {
			if !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
