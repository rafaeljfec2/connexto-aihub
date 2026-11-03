package pgstore

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"aihub/internal/concurrency"
)

type lockedConcurrencyPolicy struct {
	id    int64
	kind  concurrency.ScopeKind
	limit int
	count int
}

const selectExistingLease = `
SELECT execution_id
  FROM concurrency_leases
 WHERE execution_id = $1
   AND expires_at > $2
 FOR UPDATE`

const selectApplicableConcurrencyPolicies = `
SELECT id, scope, concurrency_limit
  FROM concurrency_policies
 WHERE enabled = true
   AND (
        scope = 'global'
     OR (scope = 'provider' AND provider_id = $1)
     OR (scope = 'model' AND provider_id = $1 AND model_id = $2)
   )
 ORDER BY scope, provider_id NULLS FIRST, model_id NULLS FIRST, id
 FOR UPDATE`

const insertLease = `
INSERT INTO concurrency_leases (execution_id, provider_id, model_id, acquired_at, expires_at)
VALUES ($1, $2, $3, $4, $5)`

const deleteLease = `
DELETE FROM concurrency_leases WHERE execution_id = $1`

const renewLease = `
UPDATE concurrency_leases
   SET expires_at = $2
 WHERE execution_id = $1
   AND expires_at > $3`

const countGlobalLeases = `
SELECT count(*)::int FROM concurrency_leases WHERE expires_at > $1`

const countProviderLeases = `
SELECT count(*)::int FROM concurrency_leases WHERE provider_id = $1 AND expires_at > $2`

const countModelLeases = `
SELECT count(*)::int FROM concurrency_leases WHERE provider_id = $1 AND model_id = $2 AND expires_at > $3`

const purgeExpiredLeases = `
DELETE FROM concurrency_leases WHERE expires_at <= $1
RETURNING execution_id, provider_id, model_id`

func (s *Store) Acquire(ctx context.Context, request concurrency.AcquireRequest, now time.Time, leaseTTL time.Duration) (concurrency.AcquireDecision, error) {
	expiresAt := now.Add(leaseTTL)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return concurrency.AcquireDecision{}, fmtUnavailable(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	expired, err := purgeExpiredLeasesTx(ctx, tx, now)
	if err != nil {
		return concurrency.AcquireDecision{}, err
	}

	var existingID string
	err = tx.QueryRow(ctx, selectExistingLease, request.ExecutionID, now).Scan(&existingID)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return concurrency.AcquireDecision{}, fmtUnavailable(err)
		}
		return concurrency.AcquireDecision{ExecutionID: request.ExecutionID, Allowed: true, ExpiredLeases: expired}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return concurrency.AcquireDecision{}, fmtUnavailable(err)
	}

	policies, err := lockApplicableConcurrencyPolicies(ctx, tx, request.ProviderID, request.ModelID)
	if err != nil {
		return concurrency.AcquireDecision{}, err
	}

	if len(policies) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return concurrency.AcquireDecision{}, fmtUnavailable(err)
		}
		return concurrency.AcquireDecision{ExecutionID: request.ExecutionID, Allowed: true, ExpiredLeases: expired}, nil
	}

	for index := range policies {
		count, err := activeLeaseCount(ctx, tx, policies[index], request.ProviderID, request.ModelID, now)
		if err != nil {
			return concurrency.AcquireDecision{}, err
		}
		policies[index].count = count
		if count >= policies[index].limit {
			if err := tx.Commit(ctx); err != nil {
				return concurrency.AcquireDecision{}, fmtUnavailable(err)
			}
			return concurrency.AcquireDecision{
				ExecutionID:   request.ExecutionID,
				Allowed:       false,
				Reason:        concurrency.RejectionReasonLimitExceeded,
				ExpiredLeases: expired,
			}, nil
		}
	}

	if _, err := tx.Exec(ctx, insertLease, request.ExecutionID, request.ProviderID, request.ModelID, now, expiresAt); err != nil {
		return concurrency.AcquireDecision{}, fmtUnavailable(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return concurrency.AcquireDecision{}, fmtUnavailable(err)
	}
	return concurrency.AcquireDecision{ExecutionID: request.ExecutionID, Allowed: true, ExpiredLeases: expired}, nil
}

func purgeExpiredLeasesTx(ctx context.Context, tx pgx.Tx, now time.Time) ([]concurrency.ExpiredLease, error) {
	rows, err := tx.Query(ctx, purgeExpiredLeases, now)
	if err != nil {
		return nil, fmtUnavailable(err)
	}
	defer rows.Close()

	var expired []concurrency.ExpiredLease
	for rows.Next() {
		var lease concurrency.ExpiredLease
		if err := rows.Scan(&lease.ExecutionID, &lease.ProviderID, &lease.ModelID); err != nil {
			return nil, fmtUnavailable(err)
		}
		expired = append(expired, lease)
	}
	return expired, rows.Err()
}

func (s *Store) Release(ctx context.Context, executionID string) error {
	_, err := s.pool.Exec(ctx, deleteLease, executionID)
	if err != nil {
		return fmtUnavailable(err)
	}
	return nil
}

func (s *Store) RenewLease(ctx context.Context, executionID string, now time.Time, leaseTTL time.Duration) error {
	grace := now.Add(-leaseTTL)
	expiresAt := now.Add(leaseTTL)
	tag, err := s.pool.Exec(ctx, renewLease, executionID, expiresAt, grace)
	if err != nil {
		return fmtUnavailable(err)
	}
	if tag.RowsAffected() == 0 {
		return concurrency.ErrNotFound
	}
	return nil
}

func lockApplicableConcurrencyPolicies(ctx context.Context, tx pgx.Tx, providerID, modelID string) ([]lockedConcurrencyPolicy, error) {
	rows, err := tx.Query(ctx, selectApplicableConcurrencyPolicies, providerID, modelID)
	if err != nil {
		return nil, fmtUnavailable(err)
	}
	defer rows.Close()

	var policies []lockedConcurrencyPolicy
	for rows.Next() {
		var policy lockedConcurrencyPolicy
		var scope string
		if err := rows.Scan(&policy.id, &scope, &policy.limit); err != nil {
			return nil, fmtUnavailable(err)
		}
		policy.kind = concurrency.ScopeKind(scope)
		policies = append(policies, policy)
	}
	return policies, rows.Err()
}

func activeLeaseCount(ctx context.Context, tx pgx.Tx, policy lockedConcurrencyPolicy, providerID, modelID string, now time.Time) (int, error) {
	var count int
	var err error
	switch policy.kind {
	case concurrency.ScopeGlobal:
		err = tx.QueryRow(ctx, countGlobalLeases, now).Scan(&count)
	case concurrency.ScopeProvider:
		err = tx.QueryRow(ctx, countProviderLeases, providerID, now).Scan(&count)
	default:
		err = tx.QueryRow(ctx, countModelLeases, providerID, modelID, now).Scan(&count)
	}
	if err != nil {
		return 0, fmtUnavailable(err)
	}
	return count, nil
}
