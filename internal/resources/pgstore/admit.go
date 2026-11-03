package pgstore

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"aihub/internal/resources"
)

type lockedPolicy struct {
	id         int64
	kind       resources.ScopeKind
	dailyLimit int
	count      int
}

const selectExistingAdmission = `
SELECT allowed, reason, period_start
  FROM resource_admissions
 WHERE admission_id = $1
 FOR UPDATE`

const selectApplicablePolicies = `
SELECT id, scope, daily_limit
  FROM resource_policies
 WHERE enabled = true
   AND (
        scope = 'global'
     OR (scope = 'provider' AND provider_id = $1)
     OR (scope = 'model' AND provider_id = $1 AND model_id = $2)
   )
 ORDER BY scope, provider_id NULLS FIRST, model_id NULLS FIRST, id
 FOR UPDATE`

const ensureCounter = `
INSERT INTO resource_period_counters (policy_id, period_start, admitted_count)
VALUES ($1, $2, 0)
ON CONFLICT DO NOTHING`

const lockCounter = `
SELECT admitted_count
  FROM resource_period_counters
 WHERE policy_id = $1
   AND period_start = $2
 FOR UPDATE`

const incrementCounter = `
UPDATE resource_period_counters
   SET admitted_count = admitted_count + 1
 WHERE policy_id = $1
   AND period_start = $2`

const insertAdmission = `
INSERT INTO resource_admissions (admission_id, provider_id, model_id, period_start, allowed, reason)
VALUES ($1, $2, $3, $4, $5, $6)`

// Admit evaluates applicable policies and increments counters atomically.
// Admission identifiers are retained for at least the current UTC period so
// internal transaction retries cannot double-charge (ADR-023).
func (s *Store) Admit(ctx context.Context, request resources.AdmissionRequest, now time.Time) (resources.AdmissionDecision, error) {
	periodStart := periodDate(now)
	periodStartTime, resetAt := resources.UsagePeriodBounds(now)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return resources.AdmissionDecision{}, fmtUnavailable(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if existing, ok, err := loadExistingAdmission(ctx, tx, request.ID); err != nil {
		return resources.AdmissionDecision{}, err
	} else if ok {
		if err := tx.Commit(ctx); err != nil {
			return resources.AdmissionDecision{}, fmtUnavailable(err)
		}
		return existing, nil
	}

	policies, err := lockApplicablePolicies(ctx, tx, request.ProviderID, request.ModelID)
	if err != nil {
		return resources.AdmissionDecision{}, err
	}

	if len(policies) == 0 {
		decision := resources.AdmissionDecision{
			AdmissionID: request.ID,
			Allowed:     true,
			PeriodStart: periodStartTime,
			ResetAt:     resetAt,
		}
		if err := persistAdmission(ctx, tx, request, periodStart, true, ""); err != nil {
			return resources.AdmissionDecision{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return resources.AdmissionDecision{}, fmtUnavailable(err)
		}
		return decision, nil
	}

	for index := range policies {
		if _, err := tx.Exec(ctx, ensureCounter, policies[index].id, periodStart); err != nil {
			return resources.AdmissionDecision{}, fmtUnavailable(err)
		}
		if err := tx.QueryRow(ctx, lockCounter, policies[index].id, periodStart).Scan(&policies[index].count); err != nil {
			return resources.AdmissionDecision{}, fmtUnavailable(err)
		}
		if policies[index].count >= policies[index].dailyLimit {
			decision := resources.AdmissionDecision{
				AdmissionID: request.ID,
				Allowed:     false,
				Reason:      resources.RejectionReasonLimitExceeded,
				PeriodStart: periodStartTime,
				ResetAt:     resetAt,
			}
			attachScopeCounts(&decision, policies)
			if err := persistAdmission(ctx, tx, request, periodStart, false, decision.Reason); err != nil {
				return resources.AdmissionDecision{}, err
			}
			if err := tx.Commit(ctx); err != nil {
				return resources.AdmissionDecision{}, fmtUnavailable(err)
			}
			return decision, nil
		}
	}

	for index := range policies {
		if _, err := tx.Exec(ctx, incrementCounter, policies[index].id, periodStart); err != nil {
			return resources.AdmissionDecision{}, fmtUnavailable(err)
		}
		policies[index].count++
	}

	decision := resources.AdmissionDecision{
		AdmissionID: request.ID,
		Allowed:     true,
		PeriodStart: periodStartTime,
		ResetAt:     resetAt,
	}
	attachScopeCounts(&decision, policies)

	if err := persistAdmission(ctx, tx, request, periodStart, true, ""); err != nil {
		return resources.AdmissionDecision{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return resources.AdmissionDecision{}, fmtUnavailable(err)
	}
	return decision, nil
}

func loadExistingAdmission(ctx context.Context, tx pgx.Tx, admissionID string) (resources.AdmissionDecision, bool, error) {
	var allowed bool
	var reason string
	var storedPeriod time.Time
	err := tx.QueryRow(ctx, selectExistingAdmission, admissionID).Scan(&allowed, &reason, &storedPeriod)
	if errors.Is(err, pgx.ErrNoRows) {
		return resources.AdmissionDecision{}, false, nil
	}
	if err != nil {
		return resources.AdmissionDecision{}, false, fmtUnavailable(err)
	}
	_, storedReset := resources.UsagePeriodBounds(storedPeriod)
	return resources.AdmissionDecision{
		AdmissionID: admissionID,
		Allowed:     allowed,
		Reason:      reason,
		PeriodStart: storedPeriod,
		ResetAt:     storedReset,
	}, true, nil
}

func lockApplicablePolicies(ctx context.Context, tx pgx.Tx, providerID, modelID string) ([]lockedPolicy, error) {
	rows, err := tx.Query(ctx, selectApplicablePolicies, providerID, modelID)
	if err != nil {
		return nil, fmtUnavailable(err)
	}
	defer rows.Close()

	var policies []lockedPolicy
	for rows.Next() {
		var policy lockedPolicy
		var scope string
		if err := rows.Scan(&policy.id, &scope, &policy.dailyLimit); err != nil {
			return nil, fmtUnavailable(err)
		}
		policy.kind = resources.ScopeKind(scope)
		policies = append(policies, policy)
	}
	if err := rows.Err(); err != nil {
		return nil, fmtUnavailable(err)
	}
	return policies, nil
}

func persistAdmission(ctx context.Context, tx pgx.Tx, request resources.AdmissionRequest, periodStart time.Time, allowed bool, reason string) error {
	if _, err := tx.Exec(ctx, insertAdmission, request.ID, request.ProviderID, request.ModelID, periodStart, allowed, reason); err != nil {
		return fmtUnavailable(err)
	}
	return nil
}

func attachScopeCounts(decision *resources.AdmissionDecision, policies []lockedPolicy) {
	for _, policy := range policies {
		switch policy.kind {
		case resources.ScopeGlobal:
			decision.GlobalCount = policy.count
		case resources.ScopeProvider:
			decision.ProviderCount = policy.count
		case resources.ScopeModel:
			decision.ModelCount = policy.count
		}
	}
}
