package catalog

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// SnapshotVersion is the current on-disk schema version.
const SnapshotVersion = 1

// Snapshot is the complete persisted catalog.
type Snapshot struct {
	Version   int                         `json:"version"`
	SavedAt   time.Time                   `json:"saved_at"`
	Providers map[string]ProviderSnapshot `json:"providers"`
}

// SnapshotStore persists a complete catalog Snapshot. It mirrors the file
// snapshot contract: Load never returns an error, degrading to an empty
// snapshot with a diagnostic, so startup cannot fail because of persisted
// catalog state; Save may fail and the manager records the persistence
// failure without committing accepted state. Implementations: the versioned
// JSON snapshot file store (ADR-010) and the PostgreSQL store (ADR-020).
type SnapshotStore interface {
	Load(ctx context.Context) (Snapshot, LoadResult)
	Save(ctx context.Context, snapshot Snapshot) (SaveResult, error)
}

// ProviderSnapshot groups one provider's accepted models and sync state.
type ProviderSnapshot struct {
	Sync   ProviderSyncState `json:"sync"`
	Models []ModelEntry      `json:"models"`
}

func (s Snapshot) validate() error {
	if s.Version != SnapshotVersion {
		return fmt.Errorf("snapshot version %d, want %d", s.Version, SnapshotVersion)
	}

	for providerID, provider := range s.Providers {
		if providerID == "" {
			return errors.New("snapshot has an empty provider id")
		}

		if !validSyncOutcome(provider.Sync.Outcome) {
			return fmt.Errorf("provider %q has invalid outcome %q", providerID, provider.Sync.Outcome)
		}

		if !validSyncErrorClass(provider.Sync.ErrorClass) {
			return fmt.Errorf("provider %q has invalid error class %q", providerID, provider.Sync.ErrorClass)
		}

		seen := make(map[string]struct{}, len(provider.Models))

		for _, entry := range provider.Models {
			if entry.ModelID == "" {
				return fmt.Errorf("provider %q has a model with an empty id", providerID)
			}

			if _, duplicate := seen[entry.ModelID]; duplicate {
				return fmt.Errorf("provider %q has duplicate model %q", providerID, entry.ModelID)
			}

			seen[entry.ModelID] = struct{}{}

			if err := validateCapabilities(entry.Capabilities); err != nil {
				return fmt.Errorf("provider %q model %q: %w", providerID, entry.ModelID, err)
			}
		}
	}

	return nil
}

func validateCapabilities(capabilities map[string]CapabilityClaim) error {
	if len(capabilities) != len(capabilityVocabulary) {
		return fmt.Errorf("capability count %d, want %d", len(capabilities), len(capabilityVocabulary))
	}

	for name, claim := range capabilities {
		if !knownCapability(name) {
			return fmt.Errorf("unknown capability %q", name)
		}

		if !validCapabilityState(claim.State) {
			return fmt.Errorf("capability %q has invalid state %q", name, claim.State)
		}

		if claim.Provenance != ProvenanceProviderCatalog {
			return fmt.Errorf("capability %q has invalid provenance %q", name, claim.Provenance)
		}

		if !validCapabilityConfidence(claim.Confidence) {
			return fmt.Errorf("capability %q has invalid confidence %q", name, claim.Confidence)
		}

		if (claim.State == CapabilityUnknown) != (claim.Confidence == ConfidenceNone) {
			return fmt.Errorf("capability %q has inconsistent state %q and confidence %q", name, claim.State, claim.Confidence)
		}

		if claim.Verification != VerificationUnverified {
			return fmt.Errorf("capability %q has invalid verification %q", name, claim.Verification)
		}
	}

	return nil
}

func validCapabilityState(state CapabilityState) bool {
	switch state {
	case CapabilitySupported, CapabilityUnsupported, CapabilityUnknown:
		return true
	default:
		return false
	}
}

func validCapabilityConfidence(confidence CapabilityConfidence) bool {
	switch confidence {
	case ConfidenceExplicit, ConfidenceDerived, ConfidenceNone:
		return true
	default:
		return false
	}
}

func validSyncOutcome(outcome SyncOutcome) bool {
	switch outcome {
	case SyncOutcomeNever, SyncOutcomeSuccess, SyncOutcomeRejectedEmpty, SyncOutcomeFailed, SyncOutcomeSkipped:
		return true
	default:
		return false
	}
}

func validSyncErrorClass(class SyncErrorClass) bool {
	switch class {
	case SyncErrorNone, SyncErrorProviderDisabled, SyncErrorProviderUnhealthy,
		SyncErrorTransport, SyncErrorTimeout, SyncErrorHTTPStatus,
		SyncErrorInvalidPayload, SyncErrorEmptyCatalog, SyncErrorPersistence:
		return true
	default:
		return false
	}
}
