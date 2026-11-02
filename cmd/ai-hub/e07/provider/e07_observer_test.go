package provider

import (
	e07support "aihub/cmd/ai-hub/e07/support"
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

	observer := e07support.NewE07ProviderObserver(http.DefaultTransport, "")
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

	observer := e07support.NewE07ProviderObserver(http.DefaultTransport, marker)
	observer.InspectReplay.Store(true)
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
