package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestE07ProviderObserverCountsOnlyTransportAttempts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	observer := newE07ProviderObserver(http.DefaultTransport, "")
	client := &http.Client{Transport: observer}

	for range 2 {
		response, err := client.Get(server.URL)
		if err != nil {
			t.Fatal("provider transport request failed")
		}
		_ = response.Body.Close()
	}

	if got := observer.Snapshot().Calls; got != 2 {
		t.Fatalf("aggregate provider call count = %d, want 2", got)
	}
}

func TestE07ProviderObserverReportsSyntheticReplayStructureAndRestoresBytes(t *testing.T) {
	marker := "E07-SYNTHETIC-MARKER-7F2C"
	body := []byte(`{"model":"test-model","input":[{"role":"user","content":"` + marker + `"}]}`)
	var mu sync.Mutex
	var received []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			_ = r.Body.Close()
		}()
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error("upstream could not read forwarded payload")
			return
		}
		mu.Lock()
		received = payload
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	observer := newE07ProviderObserver(http.DefaultTransport, marker)
	observer.inspectReplay.Store(true)
	request, err := http.NewRequest(http.MethodPost, server.URL, bytes.NewReader(body))
	if err != nil {
		t.Fatal("could not build validation request")
	}
	response, err := (&http.Client{Transport: observer}).Do(request)
	if err != nil {
		t.Fatal("provider transport request failed")
	}
	_ = response.Body.Close()

	observation := observer.Snapshot().Replay
	if !observation.PriorContextPresent {
		t.Fatal("synthetic prior-turn context was not observed")
	}
	if observation.PreviousResponseIDPresent {
		t.Fatal("previous_response_id was observed")
	}
	mu.Lock()
	defer mu.Unlock()
	if !bytes.Equal(received, body) {
		t.Fatal("provider payload bytes changed during observation")
	}
}

func TestE07CodexEnvironmentIncludesOnlyGatewayCredential(t *testing.T) {
	parent := []string{
		"PATH=/usr/bin",
		"HOME=/home/validation",
		"LANG=C.UTF-8",
		"AIHUB_GATEWAY_API_KEY=hub-gateway-test-value",
		"AIHUB_PROVIDER_OPENCODEGO_API_KEY=provider-test-value",
		"OPENCODE_API_KEY=other-test-value",
	}
	got := environmentMap(e07CodexEnvironment(parent, "/tmp/codex-home", "gateway-test-value"))

	if got["AIHUB_CLIENT_KEY"] != "gateway-test-value" {
		t.Fatal("Codex environment is missing its configured gateway key")
	}
	if got["CODEX_HOME"] != "/tmp/codex-home" || got["HOME"] != "/tmp/codex-home" {
		t.Fatal("Codex environment is missing its isolated home")
	}
	for _, key := range []string{"AIHUB_GATEWAY_API_KEY", "AIHUB_PROVIDER_OPENCODEGO_API_KEY", "OPENCODE_API_KEY"} {
		if _, exists := got[key]; exists {
			t.Fatalf("Codex environment contains a Hub-only variable: %s", key)
		}
	}
	if _, exists := got["LANG"]; !exists {
		t.Fatal("Codex environment omitted a required locale variable")
	}
}

func environmentMap(values []string) map[string]string {
	result := make(map[string]string, len(values))
	for _, value := range values {
		key, item, ok := bytes.Cut([]byte(value), []byte("="))
		if ok {
			result[string(key)] = string(item)
		}
	}
	return result
}
