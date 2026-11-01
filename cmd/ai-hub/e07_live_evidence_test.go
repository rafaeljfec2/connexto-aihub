package main

import (
	"testing"
	"time"
)

const (
	e07LiveRequestTimeout        = 2 * time.Minute
	e07ExpectedLiveProviderCalls = 8
)

type e07LiveScenarioEvidence struct {
	startedAt   time.Time
	callsBefore uint64
}

func e07StartLiveScenario(observer *e07ProviderObserver) e07LiveScenarioEvidence {
	return e07LiveScenarioEvidence{startedAt: time.Now().UTC(), callsBefore: e07ProviderCalls(observer)}
}

func e07FinishLiveScenario(t *testing.T, observer *e07ProviderObserver, evidence e07LiveScenarioEvidence, name, result string) uint64 {
	t.Helper()
	callsAfter := e07ProviderCalls(observer)
	endedAt := time.Now().UTC()
	t.Logf("E07 scenario id=%s started_at=%s ended_at=%s provider_calls_before=%d provider_calls_after=%d result=%s",
		name, evidence.startedAt.Format(time.RFC3339Nano), endedAt.Format(time.RFC3339Nano), evidence.callsBefore, callsAfter, result)
	if observer != nil && observer.BudgetSnapshot().Rejected > 0 {
		t.Fatalf("E07 provider-call budget rejected a request before transport: %s", name)
	}
	if e07ScenarioResultMustFail(name, result) {
		t.Errorf("E07 scenario failed: %s", name)
	}
	return callsAfter
}

func e07RunLiveScenario(t *testing.T, observer *e07ProviderObserver, name string, run func() string) string {
	t.Helper()
	evidence := e07StartLiveScenario(observer)
	t.Logf("E07 scenario id=%s started_at=%s provider_calls_before=%d", name,
		evidence.startedAt.Format(time.RFC3339Nano), evidence.callsBefore)
	result := run()
	integrityPass := observer == nil || observer.integrity == nil || observer.integrity.Check(t)
	if !integrityPass {
		result = "FAIL"
	}
	e07FinishLiveScenario(t, observer, evidence, name, result)
	if !integrityPass {
		t.Fatalf("E07 integrity checkpoint failed; remaining provider traffic stopped: %s", name)
	}
	return result
}

func e07RunLiveCleanupScenario(t *testing.T, observer *e07ProviderObserver, name string, run func() string) {
	t.Helper()
	evidence := e07StartLiveScenario(observer)
	t.Logf("E07 scenario id=%s started_at=%s provider_calls_before=%d", name,
		evidence.startedAt.Format(time.RFC3339Nano), evidence.callsBefore)
	result := run()
	callsAfter := e07ProviderCalls(observer)
	endedAt := time.Now().UTC()
	t.Logf("E07 scenario id=%s started_at=%s ended_at=%s provider_calls_before=%d provider_calls_after=%d result=%s",
		name, evidence.startedAt.Format(time.RFC3339Nano), endedAt.Format(time.RFC3339Nano), evidence.callsBefore, callsAfter, result)
	if result != "PASS" || (observer != nil && observer.BudgetSnapshot().Rejected > 0) {
		t.Error("E07 cleanup integrity scenario failed")
	}
}

func e07ProviderCalls(observer *e07ProviderObserver) uint64 {
	if observer == nil {
		return 0
	}
	return observer.Snapshot().Calls
}
