package providers_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"aihub/internal/config"
	"aihub/internal/providers"
)

func newTestLogger() (*slog.Logger, *bytes.Buffer) {
	buffer := new(bytes.Buffer)
	logger := slog.New(slog.NewTextHandler(buffer, &slog.HandlerOptions{Level: slog.LevelDebug}))

	return logger, buffer
}

func TestManagerStartsWithDisabledStateWithoutCredential(t *testing.T) {
	logger, _ := newTestLogger()

	client := mustClient(t, config.ProviderOpenCodeGo{BaseURL: "https://example.com/go"})
	manager := providers.NewManager(logger, client)

	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned error for disabled provider: %v", err)
	}

	assertStatuses(t, manager, providers.Status{
		ID:      providers.OpenCodeGoID,
		Enabled: false,
		State:   providers.StateDisabled,
	})
}

func TestManagerReportsHealthyAfterConnectivityCheck(t *testing.T) {
	logger, buffer := newTestLogger()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("models path = %q, want /v1/models", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	manager := providers.NewManager(logger, mustClient(t, newEnabledClientConfig(upstream.URL)))

	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("Start() must not fail on reachable provider: %v", err)
	}

	assertStatuses(t, manager, providers.Status{
		ID:      providers.OpenCodeGoID,
		Enabled: true,
		State:   providers.StateHealthy,
	})

	if bytes.Contains(buffer.Bytes(), []byte(testCredential)) {
		t.Fatalf("provider logs leaked the credential: %q", buffer.String())
	}
}

func TestManagerReportsUnhealthyAndStillStarts(t *testing.T) {
	logger, _ := newTestLogger()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("models path = %q, want /v1/models", r.URL.Path)
		}

		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer upstream.Close()

	manager := providers.NewManager(logger, mustClient(t, newEnabledClientConfig(upstream.URL)))

	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("a failed connectivity check must not abort startup: %v", err)
	}

	assertStatuses(t, manager, providers.Status{
		ID:      providers.OpenCodeGoID,
		Enabled: true,
		State:   providers.StateUnhealthy,
	})
}

func TestManagerStopClosesIdleConnections(t *testing.T) {
	logger, _ := newTestLogger()

	manager := providers.NewManager(logger, mustClient(t, newEnabledClientConfig("https://example.com/go")))

	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	if err := manager.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}

	if err := manager.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() must be idempotent, got error: %v", err)
	}
}

func TestManagerNameIsStable(t *testing.T) {
	logger, _ := newTestLogger()

	manager := providers.NewManager(logger, mustClient(t, config.ProviderOpenCodeGo{BaseURL: "https://example.com/go"}))
	if got, want := manager.Name(), "providers"; got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
}

func TestProviderClientIsAModule(t *testing.T) {
	logger, _ := newTestLogger()

	var module any = providers.NewManager(logger, mustClient(t, config.ProviderOpenCodeGo{BaseURL: "https://example.com/go"}))
	if _, ok := module.(Interface); !ok {
		t.Fatalf("manager must satisfy the app module lifecycle contract")
	}
}

// Interface mirrors app.Module without importing the app package.
type Interface interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

func assertStatuses(t *testing.T, manager *providers.Manager, want providers.Status) {
	t.Helper()

	got := manager.Statuses()
	if len(got) != 1 {
		t.Fatalf("Statuses() = %+v, want a single entry", got)
	}

	if got[0] != want {
		t.Fatalf("Statuses()[0] = %+v, want %+v", got[0], want)
	}
}
