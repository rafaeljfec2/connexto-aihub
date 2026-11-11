package pgstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"aihub/internal/resources"
)

const uniqueViolation = "23505"

const policyColumns = `id, scope, provider_id, model_id, daily_limit, enabled, created_at, updated_at`

func (s *Store) CreatePolicy(ctx context.Context, input resources.NewPolicyInput) (resources.ResourcePolicy, error) {
	providerID, modelID := nullableScopeIDs(input.Scope)
	row := s.pool.QueryRow(ctx, `
INSERT INTO resource_policies (scope, provider_id, model_id, daily_limit, enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING `+policyColumns,
		string(input.Scope.Kind), providerID, modelID, input.DailyLimit, input.Enabled)

	return mapPolicyWriteError(scanPolicy(row))
}

func (s *Store) UpdatePolicy(ctx context.Context, id int64, input resources.UpdatePolicyInput) (resources.ResourcePolicy, error) {
	if input.DailyLimit == nil {
		return s.GetPolicy(ctx, id)
	}
	row := s.pool.QueryRow(ctx, `
UPDATE resource_policies
   SET daily_limit = $2,
       updated_at = now()
 WHERE id = $1
RETURNING `+policyColumns, id, *input.DailyLimit)

	return mapPolicyWriteError(scanPolicy(row))
}

func (s *Store) SetPolicyEnabled(ctx context.Context, id int64, enabled bool) (resources.ResourcePolicy, error) {
	row := s.pool.QueryRow(ctx, `
UPDATE resource_policies
   SET enabled = $2,
       updated_at = now()
 WHERE id = $1
RETURNING `+policyColumns, id, enabled)

	return mapPolicyWriteError(scanPolicy(row))
}

func (s *Store) ConfigurePolicy(ctx context.Context, id int64, dailyLimit int, enabled bool) (resources.ResourcePolicy, error) {
	row := s.pool.QueryRow(ctx, `
UPDATE resource_policies
   SET daily_limit = $2,
       enabled = $3,
       updated_at = now()
 WHERE id = $1
RETURNING `+policyColumns, id, dailyLimit, enabled)

	return mapPolicyWriteError(scanPolicy(row))
}

func (s *Store) GetPolicy(ctx context.Context, id int64) (resources.ResourcePolicy, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+policyColumns+` FROM resource_policies WHERE id = $1`, id)
	policy, err := scanPolicy(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return resources.ResourcePolicy{}, resources.ErrNotFound
	}
	return policy, err
}

func (s *Store) ListPolicies(ctx context.Context) ([]resources.ResourcePolicy, error) {
	rows, err := s.pool.Query(ctx, `
SELECT `+policyColumns+`
  FROM resource_policies
 ORDER BY scope, provider_id NULLS FIRST, model_id NULLS FIRST, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []resources.ResourcePolicy
	for rows.Next() {
		policy, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	return policies, rows.Err()
}

func (s *Store) GetPolicyByScope(ctx context.Context, scope resources.ResourceScope) (resources.ResourcePolicy, error) {
	where, args := scopeLookupWhere(scope)
	row := s.pool.QueryRow(ctx, `SELECT `+policyColumns+` FROM resource_policies WHERE `+where, args...)
	policy, err := scanPolicy(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return resources.ResourcePolicy{}, resources.ErrNotFound
	}
	return policy, err
}

func scopeLookupWhere(scope resources.ResourceScope) (string, []any) {
	switch scope.Kind {
	case resources.ScopeGlobal:
		return `scope = 'global'`, nil
	case resources.ScopeProvider:
		return `scope = 'provider' AND provider_id = $1 AND model_id IS NULL`, []any{scope.ProviderID}
	default:
		return `scope = 'model' AND provider_id = $1 AND model_id = $2`, []any{scope.ProviderID, scope.ModelID}
	}
}

func nullableScopeIDs(scope resources.ResourceScope) (providerID, modelID any) {
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

func scanPolicy(row rowScanner) (resources.ResourcePolicy, error) {
	var (
		policy     resources.ResourcePolicy
		scope      string
		providerID *string
		modelID    *string
	)
	if err := row.Scan(&policy.ID, &scope, &providerID, &modelID,
		&policy.DailyLimit, &policy.Enabled, &policy.CreatedAt, &policy.UpdatedAt); err != nil {
		return resources.ResourcePolicy{}, err
	}
	policy.Scope = resources.ResourceScope{Kind: resources.ScopeKind(scope)}
	if providerID != nil {
		policy.Scope.ProviderID = *providerID
	}
	if modelID != nil {
		policy.Scope.ModelID = *modelID
	}
	return policy, nil
}

func mapPolicyWriteError(policy resources.ResourcePolicy, err error) (resources.ResourcePolicy, error) {
	if err == nil {
		return policy, nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return resources.ResourcePolicy{}, resources.ErrConflict
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return resources.ResourcePolicy{}, resources.ErrNotFound
	}
	return resources.ResourcePolicy{}, err
}

func periodDate(now time.Time) time.Time {
	start, _ := resources.UsagePeriodBounds(now)
	return start
}

func stringsTrim(value string) string {
	return strings.TrimSpace(value)
}

func fmtUnavailable(err error) error {
	return fmt.Errorf("%w: %v", resources.ErrUnavailable, err)
}
