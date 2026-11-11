package routing

import (
	"sync"
	"time"
)

const autoOverloadCooldown = 60 * time.Second

type autoRuntime struct {
	mu sync.Mutex

	quotaExcluded map[string]time.Time
	overloadUntil map[string]time.Time
	provisional   map[string]int
}

var autoRuntimeState = &autoRuntime{
	quotaExcluded: make(map[string]time.Time),
	overloadUntil: make(map[string]time.Time),
	provisional:   make(map[string]int),
}

func (s *autoRuntime) quotaExcludedUntil(t Target) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	until, ok := s.quotaExcluded[targetKey(t)]
	if !ok {
		return time.Time{}
	}
	return until
}

func (s *autoRuntime) overloadCooldownActive(t Target) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	until, ok := s.overloadUntil[targetKey(t)]
	if !ok {
		return false
	}
	return time.Now().UTC().Before(until)
}

// RecordAutoOverloadCooldown applies a 60s local cooldown for one OpenCode model.
func RecordAutoOverloadCooldown(t Target) {
	s := autoRuntimeState
	s.mu.Lock()
	defer s.mu.Unlock()
	s.overloadUntil[targetKey(t)] = time.Now().UTC().Add(autoOverloadCooldown)
}

// RecordAutoQuotaExhausted excludes one model until resetAt.
func RecordAutoQuotaExhausted(t Target, resetAt time.Time) {
	if resetAt.IsZero() {
		return
	}
	s := autoRuntimeState
	s.mu.Lock()
	defer s.mu.Unlock()
	s.quotaExcluded[targetKey(t)] = resetAt.UTC()
}

// ProvisionalDebit reserves one call against a pool (advisory, per instance).
func ProvisionalDebit(poolID AutoPoolID) {
	s := autoRuntimeState
	s.mu.Lock()
	defer s.mu.Unlock()
	s.provisional[string(poolID)]++
}

// ReleaseProvisionalDebit removes an advisory reservation.
func ReleaseProvisionalDebit(poolID AutoPoolID) {
	s := autoRuntimeState
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.provisional[string(poolID)] > 0 {
		s.provisional[string(poolID)]--
	}
}

func (s *autoRuntime) provisionalCount(poolID AutoPoolID) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.provisional[string(poolID)]
}
