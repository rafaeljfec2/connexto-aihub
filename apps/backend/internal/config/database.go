package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Database environment variables.
const (
	EnvDatabaseURL            = "AIHUB_DATABASE_URL"
	EnvDatabaseMaxConns       = "AIHUB_DATABASE_MAX_CONNS"
	EnvDatabaseMinConns       = "AIHUB_DATABASE_MIN_CONNS"
	EnvDatabaseConnectTimeout = "AIHUB_DATABASE_CONNECT_TIMEOUT"
	EnvCatalogPersistence     = "AIHUB_CATALOG_PERSISTENCE"
)

// Environment values for the catalog persistence mode.
const (
	EnvCatalogPersistenceFile     = "file"
	EnvCatalogPersistencePostgres = "postgres"
)

const (
	minDatabaseMaxConns       = 1
	maxDatabaseMaxConns       = 64
	minDatabaseMinConns       = 0
	maxDatabaseMinConns       = 32
	minDatabaseConnectTimeout = 100 * time.Millisecond
	maxDatabaseConnectTimeout = 60 * time.Second
)

// Database groups the PostgreSQL settings. An empty URL means "no database";
// PostgreSQL is opt-in per ADR-020.
type Database struct {
	URL            string
	MaxConns       int
	MinConns       int
	ConnectTimeout time.Duration
}

func defaultDatabase() Database {
	return Database{
		MaxConns:       4,
		MinConns:       0,
		ConnectTimeout: 5 * time.Second,
	}
}

// applyDatabaseEnvironment reads the database environment variables onto cfg.
func applyDatabaseEnvironment(cfg *Config) error {
	if value, ok := os.LookupEnv(EnvDatabaseURL); ok {
		cfg.Database.URL = value
	}

	integers := []struct {
		field  string
		env    string
		target *int
	}{
		{"database.max_conns", EnvDatabaseMaxConns, &cfg.Database.MaxConns},
		{"database.min_conns", EnvDatabaseMinConns, &cfg.Database.MinConns},
	}

	for _, item := range integers {
		value, ok := os.LookupEnv(item.env)
		if !ok {
			continue
		}

		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s (%s) must be an integer: %w", item.field, item.env, err)
		}

		*item.target = parsed
	}

	if value, ok := os.LookupEnv(EnvDatabaseConnectTimeout); ok {
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("database.connect_timeout (%s): %w", EnvDatabaseConnectTimeout, err)
		}

		cfg.Database.ConnectTimeout = parsed
	}

	if value, ok := os.LookupEnv(EnvCatalogPersistence); ok {
		mode, err := parsePersistenceMode(value)
		if err != nil {
			return err
		}

		cfg.Catalog.Persistence = mode
		cfg.Catalog.persistenceSet = true
	}

	return nil
}

func parsePersistenceMode(value string) (string, error) {
	switch strings.ToLower(value) {
	case EnvCatalogPersistenceFile:
		return EnvCatalogPersistenceFile, nil
	case EnvCatalogPersistencePostgres:
		return EnvCatalogPersistencePostgres, nil
	default:
		return "", fmt.Errorf("catalog.persistence (%s) must be %q or %q, got %q",
			EnvCatalogPersistence, EnvCatalogPersistenceFile, EnvCatalogPersistencePostgres, value)
	}
}

// resolvePersistenceDefaults derives the persistence mode from the configured
// database URL when the variable was not set explicitly (ADR-020).
func resolvePersistenceDefaults(cfg *Config) {
	if cfg.Catalog.persistenceSet {
		return
	}

	if cfg.Database.URL != "" {
		cfg.Catalog.Persistence = EnvCatalogPersistencePostgres

		return
	}

	cfg.Catalog.Persistence = EnvCatalogPersistenceFile
}

// validateDatabase reports invalid database and persistence-mode settings.
func validateDatabase(c Config) []error {
	var errs []error

	if err := validateDatabaseURL(c.Database.URL); err != nil {
		errs = append(errs, err)
	}

	switch c.Catalog.Persistence {
	case EnvCatalogPersistenceFile:
	case EnvCatalogPersistencePostgres:
		if c.Database.URL == "" {
			errs = append(errs, fmt.Errorf("%s=%q requires %s to be set",
				EnvCatalogPersistence, EnvCatalogPersistencePostgres, EnvDatabaseURL))
		}
	case "":
		errs = append(errs, fmt.Errorf("catalog.persistence was not resolved"))
	default:
		errs = append(errs, fmt.Errorf("catalog.persistence must be %q or %q, got %q",
			EnvCatalogPersistenceFile, EnvCatalogPersistencePostgres, c.Catalog.Persistence))
	}

	bounds := []struct {
		name  string
		value int
		min   int
		max   int
	}{
		{"database.max_conns", c.Database.MaxConns, minDatabaseMaxConns, maxDatabaseMaxConns},
		{"database.min_conns", c.Database.MinConns, minDatabaseMinConns, maxDatabaseMinConns},
	}

	for _, bound := range bounds {
		if bound.value < bound.min || bound.value > bound.max {
			errs = append(errs, fmt.Errorf("%s must be between %d and %d, got %d",
				bound.name, bound.min, bound.max, bound.value))
		}
	}

	if c.Database.MinConns >= c.Database.MaxConns {
		errs = append(errs, fmt.Errorf("database.min_conns (%d) must be smaller than database.max_conns (%d)",
			c.Database.MinConns, c.Database.MaxConns))
	}

	if c.Database.ConnectTimeout < minDatabaseConnectTimeout || c.Database.ConnectTimeout > maxDatabaseConnectTimeout {
		errs = append(errs, fmt.Errorf("database.connect_timeout must be between %s and %s, got %s",
			minDatabaseConnectTimeout, maxDatabaseConnectTimeout, c.Database.ConnectTimeout))
	}

	return errs
}

// validateDatabaseURL performs the stdlib-level DSN validation: URL form only
// with a postgres scheme and a non-empty host. Option-level validation happens
// when the pool is created (ADR-020).
func validateDatabaseURL(rawURL string) error {
	if rawURL == "" {
		return nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%s: %w", EnvDatabaseURL, err)
	}

	switch strings.ToLower(parsed.Scheme) {
	case "postgres", "postgresql":
	default:
		return fmt.Errorf("%s scheme must be postgres or postgresql, got %q",
			EnvDatabaseURL, parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("%s host must not be empty", EnvDatabaseURL)
	}

	return nil
}
