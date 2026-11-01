package main

import (
	"errors"
	"net/http"
	"sync"
)

const e07ProviderCallLimit = 9

var errE07ProviderBudgetExhausted = errors.New("E07 provider-call budget exhausted")

type e07ProviderBudgetSnapshot struct {
	Limit     int
	Attempted int
	Admitted  int
	Rejected  int
	Forwarded int
}

type e07ProviderBudget struct {
	mu       sync.Mutex
	snapshot e07ProviderBudgetSnapshot
}

func newE07ProviderBudget(limit int) (*e07ProviderBudget, error) {
	if limit < 1 || limit > e07ProviderCallLimit {
		return nil, errors.New("E07 provider-call budget is outside the approved range")
	}
	return &e07ProviderBudget{snapshot: e07ProviderBudgetSnapshot{Limit: limit}}, nil
}

func (budget *e07ProviderBudget) reserve() bool {
	budget.mu.Lock()
	defer budget.mu.Unlock()

	budget.snapshot.Attempted++
	if budget.snapshot.Admitted >= budget.snapshot.Limit {
		budget.snapshot.Rejected++
		return false
	}
	budget.snapshot.Admitted++
	return true
}

func (budget *e07ProviderBudget) markForwarded() {
	budget.mu.Lock()
	budget.snapshot.Forwarded++
	budget.mu.Unlock()
}

func (budget *e07ProviderBudget) Snapshot() e07ProviderBudgetSnapshot {
	budget.mu.Lock()
	defer budget.mu.Unlock()
	return budget.snapshot
}

func newE07ProviderTransport() (*http.Transport, error) {
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("default HTTP transport cannot be safely isolated")
	}
	transport := defaultTransport.Clone()
	transport.DisableKeepAlives = true
	return transport, nil
}
