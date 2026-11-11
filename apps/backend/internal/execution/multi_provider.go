package execution

import (
	"log/slog"
	"time"
)

// NewMultiProviderManager admits explicit provider IDs and delegates resolution
// to the configured opener while keeping execution lifecycle ownership shared.
func NewMultiProviderManager(logger *slog.Logger, opener OpenResponses, inactivity time.Duration, options ...Option) *Manager {
	return NewManager(logger, "", opener, inactivity, options...)
}
