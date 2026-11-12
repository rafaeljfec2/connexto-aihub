package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// DefaultPrimaryAdminEmail is provisioned when the users table is empty and a bootstrap password is available.
const DefaultPrimaryAdminEmail = "rafaeljfec2@gmail.com"

// Auth groups UI user authentication and optional SMTP settings.
type Auth struct {
	BootstrapAdminEmail    string
	BootstrapAdminPassword string
	SMTPHost               string
	SMTPPort               int
	SMTPUser               string
	SMTPPassword           string
	SMTPFrom               string
	PublicAppURL           string
}

func applyAuthEnvironment(auth *Auth) error {
	if value, ok := os.LookupEnv(EnvBootstrapAdminEmail); ok {
		auth.BootstrapAdminEmail = strings.TrimSpace(value)
	}
	if auth.BootstrapAdminEmail == "" {
		auth.BootstrapAdminEmail = DefaultPrimaryAdminEmail
	}
	if value, ok := os.LookupEnv(EnvBootstrapAdminPassword); ok {
		auth.BootstrapAdminPassword = value
	}

	if value, ok := os.LookupEnv(EnvSMTPHost); ok {
		auth.SMTPHost = value
	}
	if value, ok := os.LookupEnv(EnvSMTPPort); ok {
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s must be an integer: %w", EnvSMTPPort, err)
		}
		auth.SMTPPort = port
	}
	if value, ok := os.LookupEnv(EnvSMTPUser); ok {
		auth.SMTPUser = value
	}
	if value, ok := os.LookupEnv(EnvSMTPPassword); ok {
		auth.SMTPPassword = value
	}
	if value, ok := os.LookupEnv(EnvSMTPFrom); ok {
		auth.SMTPFrom = value
	}
	if value, ok := os.LookupEnv(EnvPublicAppURL); ok {
		auth.PublicAppURL = value
	}
	return nil
}

// UIUsersAuthEnabled reports email/password UI auth when PostgreSQL persistence is active.
func (c Config) UIUsersAuthEnabled() bool {
	return c.Catalog.Persistence == EnvCatalogPersistencePostgres &&
		c.Database.URL != "" &&
		len(c.UI.SessionSecret) >= 32
}

func validatePostgresUIAuth(c Config) []error {
	var errs []error
	if !c.UIUsersAuthEnabled() {
		return errs
	}
	if len(c.UI.SessionSecret) < 32 {
		errs = append(errs, fmt.Errorf("%s must be at least 32 bytes when %s is configured",
			EnvUISessionSecret, EnvDatabaseURL))
	}
	return errs
}

// BootstrapAdminPassword returns the password used to seed the first admin, without logging it.
func (c Config) BootstrapAdminPassword() string {
	if c.Auth.BootstrapAdminPassword != "" {
		return c.Auth.BootstrapAdminPassword
	}
	return c.UI.OperatorPassphrase
}

func combinedUIValidation(u UI, serverHost string, cfg Config) []error {
	if cfg.UIUsersAuthEnabled() {
		errs := validatePostgresUIAuth(cfg)
		if cfg.UI.ServeEmbedded && !isLoopbackServerHost(serverHost) && cfg.BootstrapAdminPassword() == "" {
			errs = append(errs, fmt.Errorf("first admin requires %s or %s on non-loopback deployments",
				EnvBootstrapAdminPassword, EnvOperatorPassphrase))
		}
		return errs
	}
	return u.validate(serverHost)
}
