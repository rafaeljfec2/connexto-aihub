package pgstore

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"aihub/internal/concurrency"
)

const uniqueViolation = "23505"

const policyColumns = `id, scope, provider_id, model_id, concurrency_limit, enabled, created_at, updated_at`

func (s *Store) CreatePolicy(ctx context.Context, input concurrency.NewPolicyInput) (concurrency.Policy, error) {
	providerID, modelID := nullableScopeIDs(input.Scope)
	row := s.pool.QueryRow(ctx, `
INSERT INTO concurrency_policies (scope, provider_id, model_id, concurrency_limit, enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING `+policyColumns,
		string(input.Scope.Kind), providerID, modelID, input.ConcurrencyLimit, input.Enabled)
	return mapPolicyWriteError(scanPolicy(row))
}

func (s *Store) UpdatePolicy(ctx context.Context, id int64, input concurrency.UpdatePolicyInput) (concurrency.Policy, error) {
	if input.ConcurrencyLimit == nil {
		return s.GetPolicy(ctx, id)
	}
	row := s.pool.QueryRow(ctx, `
UPDATE concurrency_policies
   SET concurrency_limit = $2,
       updated_at = now()
 WHERE id = $1
RETURNING `+policyColumns, id, *input.ConcurrencyLimit)
	return mapPolicyWriteError(scanPolicy(row))
}

func (s *Store) SetPolicyEnabled(ctx context.Context, id int64, enabled bool) (concurrency.Policy, error) {
	row := s.pool.QueryRow(ctx, `
UPDATE concurrency_policies
   SET enabled = $2,
       updated_at = now()
 WHERE id = $1
RETURNING `+policyColumns, id, enabled)
	return mapPolicyWriteError(scanPolicy(row))
}

func (s *Store) ConfigurePolicy(ctx context.Context, id int64, concurrencyLimit int, enabled bool) (concurrency.Policy, error) {
	row := s.pool.QueryRow(ctx, `
UPDATE concurrency_policies
   SET concurrency_limit = $2,
       enabled = $3,
       updated_at = now()
 WHERE id = $1
RETURNING `+policyColumns, id, concurrencyLimit, enabled)
	return mapPolicyWriteError(scanPolicy(row))
}

func (s *Store) GetPolicy(ctx context.Context, id int64) (concurrency.Policy, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+policyColumns+` FROM concurrency_policies WHERE id = $1`, id)
	policy, err := scanPolicy(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return concurrency.Policy{}, concurrency.ErrNotFound
	}
	return policy, err
}

func (s *Store) GetPolicyByScope(ctx context.Context, scope concurrency.Scope) (concurrency.Policy, error) {
	where, args := scopeLookupWhere(scope)
	row := s.pool.QueryRow(ctx, `SELECT `+policyColumns+` FROM concurrency_policies WHERE `+where, args...)
	policy, err := scanPolicy(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return concurrency.Policy{}, concurrency.ErrNotFound
	}
	return policy, err
}

func (s *Store) ListPolicies(ctx context.Context) ([]concurrency.Policy, error) {
	rows, err := s.pool.Query(ctx, `
SELECT `+policyColumns+`
  FROM concurrency_policies
 ORDER BY scope, provider_id NULLS FIRST, model_id NULLS FIRST, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var policies []concurrency.Policy
	for rows.Next() {
		policy, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	return policies, rows.Err()
}

func nullableScopeIDs(scope concurrency.Scope) (providerID, modelID any) {
	if scope.ProviderID != "" {
		providerID = scope.ProviderID
	}
	if scope.ModelID != "" {
		modelID = scope.ModelID
	}
	return providerID, modelID
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPolicy(row rowScanner) (concurrency.Policy, error) {
	var (
		policy concurrency.Policy
		scope  string
		providerID, modelID *string
	)
	if err := row.Scan(&policy.ID, &scope, &providerID, &modelID,
		&policy.ConcurrencyLimit, &policy.Enabled, &policy.CreatedAt, &policy.UpdatedAt); err != nil {
		return concurrency.Policy{}, err
	}
	policy.Scope = concurrency.Scope{Kind: concurrency.ScopeKind(scope)}
	if providerID != nil {
		policy.Scope.ProviderID = *providerID
	}
	if modelID != nil {
		policy.Scope.ModelID = *modelID
	}
	return policy, nil
}

func scopeLookupWhere(scope concurrency.Scope) (string, []any) {
	switch scope.Kind {
	case concurrency.ScopeGlobal:
		return `scope = 'global'`, nil
	case concurrency.ScopeProvider:
		return `scope = 'provider' AND provider_id = $1 AND model_id IS NULL`, []any{scope.ProviderID}
	default:
		return `scope = 'model' AND provider_id = $1 AND model_id = $2`, []any{scope.ProviderID, scope.ModelID}
	}
}

func mapPolicyWriteError(policy concurrency.Policy, err error) (concurrency.Policy, error) {
	if err == nil {
		return policy, nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return concurrency.Policy{}, concurrency.ErrConflict
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return concurrency.Policy{}, concurrency.ErrNotFound
	}
	return concurrency.Policy{}, err
}

func fmtUnavailable(err error) error {
	return errors.Join(concurrency.ErrUnavailable, err)
}
