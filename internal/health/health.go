// Package health exposes liveness and readiness endpoints.
package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
)

// Check verifies a dependency. A non-nil error marks the check as unavailable.
type Check func(context.Context) error

// Check and report statuses.
const (
	StatusUp   = "up"
	StatusDown = "down"

	checkOK    = "ok"
	checkError = "error"
)

// Report is the JSON body returned by health endpoints.
type Report struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// Registry holds the readiness checks registered by the application.
type Registry struct {
	mu     sync.RWMutex
	checks map[string]Check
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{checks: make(map[string]Check)}
}

// Register adds a named readiness check. Registering the same name twice
// replaces the previous check.
func (r *Registry) Register(name string, check Check) error {
	if name == "" {
		return errors.New("health: check name must not be empty")
	}

	if check == nil {
		return errors.New("health: check must not be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.checks[name] = check
	return nil
}

// Report runs every registered check. It never exposes error details, only the
// resulting status of each check.
func (r *Registry) Report(ctx context.Context) Report {
	r.mu.RLock()
	checks := make(map[string]Check, len(r.checks))
	for name, check := range r.checks {
		checks[name] = check
	}
	r.mu.RUnlock()

	report := Report{Status: StatusUp}

	if len(checks) > 0 {
		report.Checks = make(map[string]string, len(checks))
	}

	for name, check := range checks {
		status := checkOK
		if err := check(ctx); err != nil {
			status = checkError
			report.Status = StatusDown
		}
		report.Checks[name] = status
	}

	return report
}

// Live returns the liveness handler. It reports that the process is running and
// does not evaluate dependencies.
func Live() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, Report{Status: StatusUp})
	})
}

// Ready returns the readiness handler backed by the registry.
func (r *Registry) Ready() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		report := r.Report(req.Context())

		status := http.StatusOK
		if report.Status != StatusUp {
			status = http.StatusServiceUnavailable
		}

		writeJSON(w, status, report)
	})
}

func writeJSON(w http.ResponseWriter, status int, body Report) {
	payload, err := json.Marshal(body)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}
