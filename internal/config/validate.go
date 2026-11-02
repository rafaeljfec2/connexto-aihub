package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

// Validate reports every invalid field in a single error.
func (c Config) Validate() error {
	var errs []error

	if err := validateHost(c.Server.Host); err != nil {
		errs = append(errs, err)
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errs = append(errs, fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port))
	}

	timeouts := []struct {
		name  string
		value time.Duration
	}{
		{"server.read_header_timeout", c.Server.ReadHeaderTimeout},
		{"server.read_timeout", c.Server.ReadTimeout},
		{"server.write_timeout", c.Server.WriteTimeout},
		{"server.idle_timeout", c.Server.IdleTimeout},
		{"server.shutdown_timeout", c.Server.ShutdownTimeout},
	}

	for _, timeout := range timeouts {
		if timeout.value <= 0 {
			errs = append(errs, fmt.Errorf("%s must be greater than zero", timeout.name))
		}
	}

	if c.Server.ReadHeaderTimeout > c.Server.ReadTimeout {
		errs = append(errs, errors.New("server.read_header_timeout must not exceed server.read_timeout"))
	}

	if !c.Log.Level.Valid() {
		errs = append(errs, fmt.Errorf("log.level must be one of debug, info, warn, error; got %q", c.Log.Level))
	}

	if !c.Log.Format.Valid() {
		errs = append(errs, fmt.Errorf("log.format must be one of json, text; got %q", c.Log.Format))
	}

	if err := ValidateBaseURL(c.Provider.OpenCodeGo.BaseURL); err != nil {
		errs = append(errs, err)
	}

	if c.Catalog.DataDir != "" && !filepath.IsAbs(c.Catalog.DataDir) {
		errs = append(errs, fmt.Errorf("%s must be an absolute path, got %q", EnvDataDir, c.Catalog.DataDir))
	}

	if c.Catalog.StaleAfter <= 0 {
		errs = append(errs, errors.New("catalog.stale_after must be greater than zero"))
	}

	if c.Catalog.SyncTimeout <= 0 {
		errs = append(errs, errors.New("catalog.sync_timeout must be greater than zero"))
	}

	errs = append(errs, c.Gateway.validate(c.Server.Host, c.Provider.OpenCodeGo.APIKey)...)
	errs = append(errs, c.Records.validate()...)
	errs = append(errs, validateDatabase(c)...)

	return errors.Join(errs...)
}

// ValidateBaseURL rejects provider base URLs that cannot be joined safely with
// the fixed endpoint paths, including URLs with embedded credentials.
func ValidateBaseURL(baseURL string) error {
	const field = "provider.opencode-go.base_url"

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("%s scheme must be http or https, got %q", field, parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("%s host must not be empty", field)
	}

	if parsed.User != nil {
		return errors.New(field + " must not embed credentials; use the dedicated environment variable")
	}

	if parsed.RawQuery != "" {
		return errors.New(field + " must not carry a query string")
	}

	if parsed.Fragment != "" {
		return errors.New(field + " must not carry a fragment")
	}

	return nil
}

func validateHost(host string) error {
	if host == "" {
		return errors.New("server.host must not be empty")
	}

	if strings.TrimSpace(host) != host {
		return fmt.Errorf("server.host must not contain surrounding spaces: %q", host)
	}

	if net.ParseIP(host) != nil || host == "localhost" {
		return nil
	}

	if len(host) > 253 {
		return fmt.Errorf("server.host is too long: %d characters", len(host))
	}

	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 {
			return fmt.Errorf("server.host %q is not a valid hostname", host)
		}

		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return fmt.Errorf("server.host %q has a label starting or ending with '-'", host)
		}

		for _, char := range label {
			isAllowed := char >= 'a' && char <= 'z' ||
				char >= 'A' && char <= 'Z' ||
				char >= '0' && char <= '9' ||
				char == '-'
			if !isAllowed {
				return fmt.Errorf("server.host %q contains invalid character %q", host, char)
			}
		}
	}

	return nil
}
