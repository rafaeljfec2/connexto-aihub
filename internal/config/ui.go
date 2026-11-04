package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"aihub/internal/ui"
)

const (
	EnvOperatorPassphrase = "AIHUB_OPERATOR_PASSPHRASE"
	EnvUISessionSecret    = "AIHUB_UI_SESSION_SECRET"
	EnvUISessionTTL       = "AIHUB_UI_SESSION_TTL"
	EnvUISecureCookies    = "AIHUB_UI_SECURE_COOKIES"
	EnvServeEmbeddedUI    = "AIHUB_SERVE_EMBEDDED_UI"
)

// UI groups operator session settings for the embedded frontend.
type UI struct {
	OperatorPassphrase string
	SessionSecret      string
	SessionTTL         time.Duration
	SecureCookies      bool
	ServeEmbedded      bool
}

func applyUIEnvironment(uiCfg *UI) error {
	if value, ok := os.LookupEnv(EnvOperatorPassphrase); ok {
		uiCfg.OperatorPassphrase = value
	}

	if value, ok := os.LookupEnv(EnvUISessionSecret); ok {
		uiCfg.SessionSecret = value
	}

	if value, ok := os.LookupEnv(EnvUISessionTTL); ok {
		ttl, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("ui.session_ttl (%s) must be a duration: %w", EnvUISessionTTL, err)
		}
		uiCfg.SessionTTL = ttl
	}

	if value, ok := os.LookupEnv(EnvUISecureCookies); ok {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("ui.secure_cookies (%s) must be a boolean: %w", EnvUISecureCookies, err)
		}
		uiCfg.SecureCookies = parsed
	}

	if value, ok := os.LookupEnv(EnvServeEmbeddedUI); ok {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("ui.serve_embedded (%s) must be a boolean: %w", EnvServeEmbeddedUI, err)
		}
		uiCfg.ServeEmbedded = parsed
	}

	return nil
}

func (u UI) validate(serverHost string) []error {
	var errs []error

	if u.OperatorPassphrase != "" && len(u.SessionSecret) < 32 {
		errs = append(errs, fmt.Errorf("%s must be at least 32 bytes when %s is set", EnvUISessionSecret, EnvOperatorPassphrase))
	}

	if u.SessionSecret != "" && u.OperatorPassphrase == "" {
		errs = append(errs, errors.New("operator passphrase must be set when ui session secret is configured"))
	}

	if u.ServeEmbedded && !isLoopbackServerHost(serverHost) && u.OperatorPassphrase == "" {
		errs = append(errs, errors.New("embedded ui on non-loopback listeners requires operator passphrase"))
	}

	return errs
}

// BuildOperatorGate constructs the runtime operator gate from validated config.
func (c Config) BuildOperatorGate() (ui.Gate, error) {
	return ui.NewGate(ui.Settings{
		Passphrase:     c.UI.OperatorPassphrase,
		SessionSecret:  []byte(c.UI.SessionSecret),
		SessionTTL:     c.UI.SessionTTL,
		SecureCookies:  c.UI.SecureCookies,
		LoopbackServer: isLoopbackServerHost(c.Server.Host),
	})
}
