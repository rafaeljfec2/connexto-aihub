// Command bootstrap-primary-admin applies migrations and ensures the primary hub owner user exists.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"aihub/internal/config"
	"aihub/internal/db"
	"aihub/internal/userauth"
)

func main() {
	envPath := filepath.Join("..", "..", "..", ".env")
	if len(os.Args) > 1 {
		envPath = os.Args[1]
	}
	if err := config.LoadDotEnv(envPath); err != nil {
		fmt.Fprintf(os.Stderr, "load dotenv: %v\n", err)
		os.Exit(1)
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	if !cfg.UIUsersAuthEnabled() {
		fmt.Fprintf(os.Stderr, "postgres UI auth is not enabled (need %s and %s)\n", config.EnvDatabaseURL, config.EnvUISessionSecret)
		os.Exit(1)
	}

	pool, err := db.Open(db.Settings{
		URL:            cfg.Database.URL,
		MaxConns:       cfg.Database.MaxConns,
		MinConns:       cfg.Database.MinConns,
		ConnectTimeout: cfg.Database.ConnectTimeout,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, _, err := db.Migrate(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}

	store := userauth.NewStore(pool)
	svc := userauth.NewService(store, []byte(cfg.UI.SessionSecret), cfg.UI.SecureCookies, nil, nil)
	if err := svc.BootstrapPrimaryAdmin(ctx, cfg.Auth.BootstrapAdminEmail, cfg.BootstrapAdminPassword()); err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("primary admin ready for %s\n", cfg.Auth.BootstrapAdminEmail)
}
