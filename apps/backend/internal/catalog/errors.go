package catalog

import "errors"

// Domain and infrastructure errors surfaced by the catalog.
var (
	ErrProviderNotFound = errors.New("provider not found")
	ErrSyncInProgress   = errors.New("sync already in progress")
	ErrProviderDisabled = errors.New("provider is disabled")
	ErrWritesDisabled   = errors.New("catalog writes are disabled")
	ErrStopped          = errors.New("catalog is stopped")
	ErrPersistence      = errors.New("catalog persistence failed")
)
