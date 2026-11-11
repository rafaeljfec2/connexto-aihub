// Package dbtest supports integration tests that need a real PostgreSQL
// server (ADR-020). Tests opt in by exporting AIHUB_TEST_DATABASE_URL; every
// other test run stays green without a database. The helper gives each calling
// package its own throwaway database, created from the URL database as an
// admin connection, so packages and tests never share state even when the
// test binaries run in parallel.
package dbtest

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/db"
)

// testDatabaseURLEnv opt-in switch for database-backed tests.
const testDatabaseURLEnv = "AIHUB_TEST_DATABASE_URL"

// Pool connects the calling test to a freshly created, disposable database.
// The database name is derived from the calling package and test name (for
// example internal/catalog receives aihub_it_catalog_testfoo) so parallel
// tests are isolated by construction. Both the database and the pool are
// dropped when the test finishes. The schema is left empty: callers decide
// whether and when to migrate.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv(testDatabaseURLEnv)
	if dsn == "" {
		t.Skipf("skipping: integration test requires %s pointing to a PostgreSQL instance", testDatabaseURLEnv)
	}

	name := databaseName(t)
	admin, err := connectAdmin(dsn)
	if err != nil {
		t.Fatalf("connect test database administrator: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := recreateDatabase(ctx, admin, name); err != nil {
		admin.Close()
		t.Fatalf("recreate test database %s: %v", name, err)
	}

	pool, err := db.Open(db.Settings{
		URL:            databaseURL(dsn, name),
		MaxConns:       2,
		MinConns:       0,
		ConnectTimeout: 5 * time.Second,
	})
	if err != nil {
		admin.Close()
		t.Fatalf("open test pool for %s: %v", name, err)
	}

	t.Cleanup(func() {
		pool.Close()

		dropCtx, dropCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dropCancel()

		if err := dropDatabase(dropCtx, admin, name); err != nil {
			t.Logf("drop test database %s: %v", name, err)
		}

		admin.Close()
	})

	return pool
}

// databaseName derives a unique, identifier-safe database name from the
// calling test's package directory and test name, so sibling tests inside a
// package are isolated from each other as well.
func databaseName(t *testing.T) string {
	t.Helper()

	packageFile, ok := poolCallerFile()
	if !ok {
		return sanitizeDatabaseName("aihub_it_" + t.Name())
	}

	pkg := strings.ToLower(filepath.Base(filepath.Dir(packageFile)))

	return sanitizeDatabaseName("aihub_it_" + pkg + "_" + t.Name())
}

// poolCallerFile returns the file of the caller that invoked Pool.
func poolCallerFile() (string, bool) {
	// 0 = this function, 1 = databaseName, 2 = Pool, 3 = the test.
	_, file, _, ok := runtime.Caller(3)

	return file, ok
}

// sanitizeDatabaseName keeps only lowercase identifier-safe characters.
func sanitizeDatabaseName(name string) string {
	sanitized := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		default:
			return '_'
		}
	}, name)

	// PostgreSQL identifiers max out at 63 bytes.
	if len(sanitized) > 63 {
		return sanitized[:63]
	}

	return sanitized
}

// connectAdmin connects to the PostgreSQL server named in the URL. Database
// creation and dropping work regardless of which database the connection
// initially attaches to.
func connectAdmin(dsn string) (*pgxpool.Pool, error) {
	return pgxpool.New(context.Background(), dsn)
}

// recreateDatabase drops and recreates the named database for a clean start.
func recreateDatabase(ctx context.Context, admin *pgxpool.Pool, name string) error {
	conn, err := admin.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire administrative connection: %w", err)
	}

	defer conn.Release()

	if err := dropDatabase(ctx, admin, name); err != nil {
		return err
	}

	if _, err := conn.Conn().Exec(ctx,
		fmt.Sprintf("CREATE DATABASE %s", quotedIdentifier(name))); err != nil {
		return fmt.Errorf("create database: %w", err)
	}

	return nil
}

func dropDatabase(ctx context.Context, admin *pgxpool.Pool, name string) error {
	conn, err := admin.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire administrative connection: %w", err)
	}

	defer conn.Release()

	_, err = conn.Conn().Exec(ctx,
		fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quotedIdentifier(name)))

	if err != nil && !isCatalogLookupFailure(err) {
		return fmt.Errorf("drop database: %w", err)
	}

	return nil
}

// isCatalogLookupFailure reports the benign race where another drop removed
// the database between the IF EXISTS check and the execution.
func isCatalogLookupFailure(err error) bool {
	return strings.Contains(err.Error(), "does not exist")
}

// quotedIdentifier safely quotes a PostgreSQL identifier.
func quotedIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// databaseURL rewrites the path database of a URL-form DSN.
func databaseURL(dsn, name string) string {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}

	parsed.Path = "/" + name

	return parsed.String()
}
