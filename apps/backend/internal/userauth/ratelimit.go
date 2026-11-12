package userauth

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type fixedWindow struct {
	mu      sync.Mutex
	entries map[string]windowEntry
	limit   int
	window  time.Duration
}

type windowEntry struct {
	count   int
	resetAt time.Time
}

func newFixedWindow(limit int, window time.Duration) *fixedWindow {
	return &fixedWindow{
		entries: make(map[string]windowEntry),
		limit:   limit,
		window:  window,
	}
}

func (w *fixedWindow) allow(key string) bool {
	now := time.Now()
	w.mu.Lock()
	defer w.mu.Unlock()

	entry, ok := w.entries[key]
	if !ok || now.After(entry.resetAt) {
		w.entries[key] = windowEntry{count: 1, resetAt: now.Add(w.window)}
		return true
	}

	if entry.count >= w.limit {
		return false
	}

	entry.count++
	w.entries[key] = entry
	return true
}

// Limiter groups in-memory per-IP limits for auth endpoints.
type Limiter struct {
	login  *fixedWindow
	recovery *fixedWindow
}

// NewLimiter builds login (5/min) and recovery (3/hour) limiters.
func NewLimiter() *Limiter {
	return &Limiter{
		login:    newFixedWindow(5, time.Minute),
		recovery: newFixedWindow(3, time.Hour),
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (l *Limiter) AllowLogin(r *http.Request) bool {
	return l.login.allow(clientIP(r))
}

func (l *Limiter) AllowRecovery(r *http.Request) bool {
	return l.recovery.allow(clientIP(r))
}
