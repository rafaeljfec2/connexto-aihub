package providers

import (
	"fmt"
	"net"
	"net/url"
	"strings"

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
	if strings.EqualFold(parsed.Scheme, "http") && !isLoopbackHost(parsed.Hostname()) {
		return fmt.Errorf("%w: endpoint must use https unless it targets a loopback host", ErrInvalid)
	}
	if _, allowed := s.allowedHosts[strings.ToLower(parsed.Hostname())]; !allowed {
		return fmt.Errorf("%w: endpoint host is not in AIHUB_PROVIDER_ALLOWED_HOSTS", ErrInvalid)
	}
	return nil
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
