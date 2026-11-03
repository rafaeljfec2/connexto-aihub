package pgstore

import (
	"context"
	"fmt"
	"strings"

	"aihub/internal/resources"
)

func (s *Store) ListUsage(ctx context.Context, query resources.UsageQuery) ([]resources.UsageSnapshot, error) {
	period := periodDate(query.PeriodStart)
	start, reset := resources.UsagePeriodBounds(query.PeriodStart)

	args := []any{period}
	filter := strings.Builder{}
	filter.WriteString(`
SELECT `)
	filter.WriteString(policyColumns)
	filter.WriteString(`,
       COALESCE(c.admitted_count, 0)
  FROM resource_policies p
  LEFT JOIN resource_period_counters c
    ON c.policy_id = p.id AND c.period_start = $1
 WHERE 1=1`)

	if query.ProviderID != "" {
		args = append(args, query.ProviderID)
		fmt.Fprintf(&filter, `
   AND (p.scope = 'global' OR p.provider_id = $%d)`, len(args))
	}
	if query.ModelID != "" {
		args = append(args, query.ModelID)
		fmt.Fprintf(&filter, `
   AND (p.scope = 'global' OR p.scope = 'provider' OR p.model_id = $%d)`, len(args))
	}

	filter.WriteString(`
 ORDER BY p.scope, p.provider_id NULLS FIRST, p.model_id NULLS FIRST, p.id`)

	rows, err := s.pool.Query(ctx, filter.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []resources.UsageSnapshot
	for rows.Next() {
		policy, admitted, err := scanPolicyUsage(rows)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, resources.UsageSnapshot{
			Policy:        policy,
			PeriodStart:   start,
			ResetAt:       reset,
			AdmittedCount: admitted,
		})
	}
	return snapshots, rows.Err()
}

func scanPolicyUsage(row rowScanner) (resources.ResourcePolicy, int, error) {
	var (
		policy     resources.ResourcePolicy
		scope      string
		providerID *string
		modelID    *string
		admitted   int
	)
	if err := row.Scan(&policy.ID, &scope, &providerID, &modelID,
		&policy.DailyLimit, &policy.Enabled, &policy.CreatedAt, &policy.UpdatedAt, &admitted); err != nil {
		return resources.ResourcePolicy{}, 0, err
	}
	policy.Scope = resources.ResourceScope{Kind: resources.ScopeKind(scope)}
	if providerID != nil {
		policy.Scope.ProviderID = *providerID
	}
	if modelID != nil {
		policy.Scope.ModelID = *modelID
	}
	return policy, admitted, nil
}
