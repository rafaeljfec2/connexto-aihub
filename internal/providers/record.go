package providers

import (
	"context"
	"errors"
	"time"
)

// Errors returned by the provider configuration store and service.
var (
	ErrNotFound = errors.New("provider configuration not found")
	ErrConflict = errors.New("provider configuration already exists")
)

// Record is the administrative view of one provider configuration. It is
// independent of the adapter type: several records may share a type (C2+).
// It never carries the credential, only whether one is stored.
type Record struct {
	ID                      string
	Name                    string
	Type                    string
	Endpoint                string
	Enabled                 bool
	HasCredential           bool
	HasManagementCredential bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// NewRecord is the data needed to create a configuration. Credential holds
// the sealed ciphertext; nil means no credential is stored yet.
type NewRecord struct {
	ID                   string
	Name                 string
	Type                 string
	Endpoint             string
	Enabled              bool
	Credential           []byte
	ManagementCredential []byte
}

// Changes carries the optional fields of an update. A nil field is unchanged.
// Credential holds the sealed ciphertext so that every change is written in a
// single statement.
type Changes struct {
	Name                 *string
	Endpoint             *string
	Credential           []byte
	ManagementCredential []byte
}

// Repository persists provider configurations. Implementations live outside
// this package so the domain never depends on a database driver.
type Repository interface {
	// Create inserts a new configuration and returns ErrConflict when the id
	// or name is already taken.
	Create(ctx context.Context, record NewRecord) (Record, error)
	// EnsureConfigured inserts the configuration when absent, otherwise fills
	// only the management fields that are still unset. Persisted values always
	// win over the supplied defaults.
	EnsureConfigured(ctx context.Context, record NewRecord) (Record, error)
	Get(ctx context.Context, id string) (Record, error)
	List(ctx context.Context) ([]Record, error)
	Update(ctx context.Context, id string, changes Changes) (Record, error)
	SetEnabled(ctx context.Context, id string, enabled bool) (Record, error)
	// SealedCredential returns the stored ciphertext, or nil when none exists.
	SealedCredential(ctx context.Context, id string) ([]byte, error)
	// SealedManagementCredential returns the separately protected administrative credential.
	SealedManagementCredential(ctx context.Context, id string) ([]byte, error)
}
