package execution

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestClientCancellationClosesBodyAndJoinsReader(t *testing.T) {
	body := newBlockingBody("")
	providerContext := make(chan context.Context, 1)
	manager := newTestManager(time.Hour, func(ctx context.Context, _ Target, _ []byte, _, _ string) (*http.Response, error) {
		providerContext <- ctx
		return responseFor(body, 200), nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan Result, 1)
	go func() {
		result, _ := manager.Execute(ctx, validCommand(), &sinkProbe{})
		done <- result
	}()
	upstreamContext := <-providerContext
	awaitSignal(t, body.readStarted, "upstream body read")
	cancel()
	select {
	case <-upstreamContext.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("client cancellation did not reach the provider request context")
	}

	select {
	case result := <-done:
		if result.Outcome != OutcomeCanceled || result.Cause != CauseClient {
			t.Fatalf("result = %+v, want canceled(client)", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("execution did not finish after client cancellation")
	}
	if got := body.closeCount.Load(); got != 1 {
		t.Fatalf("body close count = %d, want 1", got)
	}
	if !manager.Drained() {
		t.Fatal("manager still has active execution work after reader join")
	}
}

func TestShutdownCancellationClosesBodyAndRetainsShutdownCause(t *testing.T) {
	body := newBlockingBody("")
	manager := newTestManager(time.Hour, func(context.Context, Target, []byte, string, string) (*http.Response, error) {
		return responseFor(body, 200), nil
	})
	done := make(chan Result, 1)
	go func() {
		result, _ := manager.Execute(context.Background(), validCommand(), &sinkProbe{})
		done <- result
	}()
	awaitSignal(t, body.readStarted, "upstream body read")
	manager.BeginDrain()
	manager.CancelActive()

	select {
	case result := <-done:
		if result.Outcome != OutcomeCanceled || result.Cause != CauseShutdown {
			t.Fatalf("result = %+v, want canceled(shutdown)", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("execution did not finish after shutdown cancellation")
	}
	if got := body.closeCount.Load(); got != 1 {
		t.Fatalf("body close count = %d, want 1", got)
	}
	if !manager.Drained() {
		t.Fatal("manager still has active execution work after shutdown join")
	}
}
