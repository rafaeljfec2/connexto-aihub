package server_test

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"aihub/internal/config"
	"aihub/internal/server"
)

func TestServerRejectsNonLoopbackWithoutPolicy(t *testing.T) {
	cfg := config.Default().Server
	cfg.Host = "0.0.0.0"
	cfg.Port = 0

	instance := server.New(cfg, http.NotFoundHandler(), testLogger(&bytes.Buffer{}))

	err := instance.Start(context.Background())
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = instance.Stop(ctx)

		t.Fatal("Start() must reject a non-loopback listener without explicit allowed hosts")
	}

	if !strings.Contains(err.Error(), "non-loopback") {
		t.Fatalf("error = %q, want a non-loopback refusal", err)
	}
}

func TestServerAcceptsNonLoopbackWithPolicy(t *testing.T) {
	cfg := config.Default().Server
	cfg.Host = "0.0.0.0"
	cfg.Port = 0

	instance := server.New(cfg, http.NotFoundHandler(), testLogger(&bytes.Buffer{}),
		server.WithNonLoopbackAllowed(true),
	)

	if err := instance.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	_, port, err := net.SplitHostPort(instance.Addr())
	if err != nil {
		t.Fatalf("split address: %v", err)
	}

	response, err := http.Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		t.Fatalf("GET returned error: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 from the injected handler", response.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := instance.Stop(ctx); err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}
}

func TestServerRejectsWildcardIPv6WithoutPolicy(t *testing.T) {
	cfg := config.Default().Server
	cfg.Host = "::"
	cfg.Port = 0

	instance := server.New(cfg, http.NotFoundHandler(), testLogger(&bytes.Buffer{}))

	err := instance.Start(context.Background())
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = instance.Stop(ctx)

		t.Fatal("Start() must reject an IPv6 wildcard listener without explicit allowed hosts")
	}

	if !strings.Contains(err.Error(), "non-loopback") {
		t.Skipf("IPv6 wildcard bind unavailable in this environment: %v", err)
	}
}
