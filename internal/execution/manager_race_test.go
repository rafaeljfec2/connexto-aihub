package execution

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTerminalArbiterPublishesOneImmutableWinner(t *testing.T) {
	events := []terminalEvent{
		{outcome: OutcomeCompleted, phase: PhaseForwarding},
		{outcome: OutcomeFailed, phase: PhaseForwarding, cause: CauseUpstreamRead},
		{outcome: OutcomeCanceled, phase: PhaseForwarding, cause: CauseClient},
		{outcome: OutcomeCanceled, phase: PhaseOpening, cause: CauseShutdown},
		{outcome: OutcomeTimedOut, phase: PhaseOpening, cause: CauseOpening},
		{outcome: OutcomeTimedOut, phase: PhaseForwarding, cause: CauseInactivity},
	}

	for range 32 {
		manager := newTestManager(time.Second, nil)
		work := newExecution(manager, "execution-race-test", validCommand(), context.Background())
		start := make(chan struct{})
		winners := make(chan terminalEvent, len(events))
		var group sync.WaitGroup
		for _, event := range events {
			event := event
			group.Add(1)
			go func() {
				defer group.Done()
				<-start
				if work.reserve(event) {
					winners <- event
				}
			}()
		}
		close(start)
		group.Wait()
		close(winners)
		if len(winners) != 1 {
			t.Fatalf("terminal events reserved %d winners, want exactly one", len(winners))
		}
		winner := <-winners

		result := work.finish()
		if result.Outcome != winner.outcome || result.Phase != winner.phase || result.Cause != winner.cause {
			t.Fatalf("result = %+v, want first event %+v", result, winner)
		}
		for _, loser := range events {
			if work.reserve(loser) {
				t.Fatalf("terminal event %+v mutated a published result", loser)
			}
		}
		if work.event != winner {
			t.Fatalf("stored terminal event = %+v, want immutable %+v", work.event, winner)
		}
	}
}

func TestConcurrentRegistrationRacesDrainGateWithoutLateProviderCalls(t *testing.T) {
	const requestCount = 64
	var providerCalls atomic.Int32
	manager := newTestManager(time.Second, func(context.Context, Target, []byte, string, string) (*http.Response, error) {
		providerCalls.Add(1)
		return responseFor(newEOFBody(""), 200), nil
	})
	start := make(chan struct{})
	results := make(chan error, requestCount)
	var group sync.WaitGroup
	for range requestCount {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			_, err := manager.Execute(context.Background(), validCommand(), &sinkProbe{})
			results <- err
		}()
	}
	close(start)
	manager.BeginDrain()
	group.Wait()
	close(results)

	var admitted int32
	for err := range results {
		if err == nil {
			admitted++
		}
		if err != nil && !errors.Is(err, ErrUnavailable) {
			t.Fatalf("Execute() returned unexpected admission error: %v", err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Wait(ctx); err != nil {
		t.Fatalf("Wait() returned error after drain: %v", err)
	}
	if got := providerCalls.Load(); got != admitted {
		t.Fatalf("provider calls = %d, want one for each admitted execution (%d)", got, admitted)
	}
	if !manager.Drained() {
		t.Fatal("manager reported active work after drain completed")
	}
}

func TestResponseCommitSerializesWithTerminalArbitration(t *testing.T) {
	body := newBlockingBody("")
	sink := &gatedCommitSink{started: make(chan struct{}), release: make(chan struct{})}
	manager := newTestManager(time.Hour, func(context.Context, Target, []byte, string, string) (*http.Response, error) {
		return responseFor(body, http.StatusOK), nil
	})
	done := make(chan Result, 1)
	go func() {
		result, _ := manager.Execute(context.Background(), validCommand(), sink)
		done <- result
	}()
	awaitSignal(t, sink.started, "response metadata commitment")

	manager.mu.Lock()
	var work *execution
	for _, candidate := range manager.active {
		work = candidate
	}
	manager.mu.Unlock()
	if work == nil {
		t.Fatal("execution was not registered while committing response metadata")
	}
	if work.mu.TryLock() {
		work.mu.Unlock()
		t.Fatal("terminal arbiter was unlocked during response metadata commitment")
	}

	manager.BeginDrain()
	cancelDone := make(chan struct{})
	go func() {
		manager.CancelActive()
		close(cancelDone)
	}()
	close(sink.release)
	awaitSignal(t, cancelDone, "shutdown cancellation after response commitment")
	select {
	case result := <-done:
		if result.Outcome != OutcomeCanceled || result.Cause != CauseShutdown || !result.Committed {
			t.Fatalf("result = %+v, want committed canceled(shutdown)", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("execution did not finish after shutdown cancellation")
	}
	if body.closeCount.Load() != 1 || !manager.Drained() {
		t.Fatalf("body closes = %d, manager drained = %t", body.closeCount.Load(), manager.Drained())
	}
}

type gatedCommitSink struct {
	started   chan struct{}
	release   chan struct{}
	committed atomic.Bool
}

func (s *gatedCommitSink) Commit(int, string) {
	close(s.started)
	<-s.release
	s.committed.Store(true)
}

func (*gatedCommitSink) WriteChunk([]byte) error { return nil }
