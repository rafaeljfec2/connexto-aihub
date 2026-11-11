package providers_test

import (
	"bytes"
	"context"
	"sync"
	"time"

	"aihub/internal/providers"
)

// memoryRepository is an in-memory providers.Repository. It mirrors the
// PostgreSQL semantics the service depends on: ids and names are unique,
// EnsureConfigured never overwrites a persisted value.
type memoryRepository struct {
	mu               sync.Mutex
	records          map[string]providers.Record
	sealed           map[string][]byte
	sealedManagement map[string][]byte
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		records:          map[string]providers.Record{},
		sealed:           map[string][]byte{},
		sealedManagement: map[string][]byte{},
	}
}

func (r *memoryRepository) nameTaken(name, exceptID string) bool {
	for id, record := range r.records {
		if id != exceptID && record.Name == name {
			return true
		}
	}

	return false
}

func (r *memoryRepository) Create(_ context.Context, input providers.NewRecord) (providers.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.records[input.ID]; exists || r.nameTaken(input.Name, "") {
		return providers.Record{}, providers.ErrConflict
	}

	now := time.Now().UTC()
	record := providers.Record{
		ID: input.ID, Name: input.Name, Type: input.Type, Endpoint: input.Endpoint,
		Enabled: input.Enabled, HasCredential: input.Credential != nil,
		HasManagementCredential: input.ManagementCredential != nil,
		CreatedAt:               now, UpdatedAt: now,
	}
	r.records[input.ID] = record
	r.sealed[input.ID] = input.Credential
	r.sealedManagement[input.ID] = input.ManagementCredential

	return record, nil
}

func (r *memoryRepository) EnsureConfigured(_ context.Context, input providers.NewRecord) (providers.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, exists := r.records[input.ID]
	if !exists {
		now := time.Now().UTC()
		record = providers.Record{ID: input.ID, Name: input.Name, Type: input.Type,
			Endpoint: input.Endpoint, Enabled: input.Enabled, CreatedAt: now, UpdatedAt: now}
	}

	if r.sealed[input.ID] == nil && input.Credential != nil {
		r.sealed[input.ID] = input.Credential
	}
	if r.sealedManagement[input.ID] == nil && input.ManagementCredential != nil {
		r.sealedManagement[input.ID] = input.ManagementCredential
	}

	record.HasCredential = r.sealed[input.ID] != nil
	record.HasManagementCredential = r.sealedManagement[input.ID] != nil
	r.records[input.ID] = record

	return record, nil
}

func (r *memoryRepository) Get(_ context.Context, id string) (providers.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, ok := r.records[id]
	if !ok {
		return providers.Record{}, providers.ErrNotFound
	}

	return record, nil
}

func (r *memoryRepository) List(_ context.Context) ([]providers.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]providers.Record, 0, len(r.records))
	for _, record := range r.records {
		out = append(out, record)
	}

	return out, nil
}

func (r *memoryRepository) Update(_ context.Context, id string, changes providers.Changes) (providers.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, ok := r.records[id]
	if !ok {
		return providers.Record{}, providers.ErrNotFound
	}

	if changes.Name != nil {
		if r.nameTaken(*changes.Name, id) {
			return providers.Record{}, providers.ErrConflict
		}

		record.Name = *changes.Name
	}

	if changes.Endpoint != nil {
		record.Endpoint = *changes.Endpoint
	}

	if changes.Credential != nil {
		r.sealed[id] = changes.Credential
		record.HasCredential = true
	}
	if changes.ManagementCredential != nil {
		r.sealedManagement[id] = changes.ManagementCredential
		record.HasManagementCredential = true
	}

	record.UpdatedAt = time.Now().UTC()
	r.records[id] = record

	return record, nil
}

func (r *memoryRepository) SetEnabled(_ context.Context, id string, enabled bool) (providers.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, ok := r.records[id]
	if !ok {
		return providers.Record{}, providers.ErrNotFound
	}

	record.Enabled = enabled
	r.records[id] = record

	return record, nil
}

func (r *memoryRepository) SealedCredential(_ context.Context, id string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.records[id]; !ok {
		return nil, providers.ErrNotFound
	}

	return bytes.Clone(r.sealed[id]), nil
}

func (r *memoryRepository) SealedManagementCredential(_ context.Context, id string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.records[id]; !exists {
		return nil, providers.ErrNotFound
	}
	return r.sealedManagement[id], nil
}

// recordingAdapter captures every runtime configuration the service applies.
type recordingAdapter struct {
	mu      sync.Mutex
	applied []providers.Settings
}

func (a *recordingAdapter) Reconfigure(settings providers.Settings) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.applied = append(a.applied, settings)

	return nil
}

func (a *recordingAdapter) Enabled() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	return len(a.applied) > 0 && a.applied[len(a.applied)-1].Enabled
}

func (a *recordingAdapter) last() providers.Settings {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.applied[len(a.applied)-1]
}
