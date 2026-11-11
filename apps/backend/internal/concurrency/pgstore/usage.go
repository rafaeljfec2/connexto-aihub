package pgstore

import (
	"context"
	"time"

	"aihub/internal/concurrency"
)

func (s *Store) ListUsage(ctx context.Context, query concurrency.UsageQuery) ([]concurrency.UsageSnapshot, error) {
	policies, err := s.ListPolicies(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var snapshots []concurrency.UsageSnapshot
	for _, policy := range policies {
		if query.ProviderID != "" && policy.Scope.ProviderID != "" && policy.Scope.ProviderID != query.ProviderID {
			continue
		}
		if query.ModelID != "" && policy.Scope.ModelID != "" && policy.Scope.ModelID != query.ModelID {
			continue
		}
		active, err := s.countActiveForPolicy(ctx, policy, now)
		if err != nil {
			return nil, err
		}
		available := 0
		if policy.Enabled {
			available = policy.ConcurrencyLimit - active
			if available < 0 {
				available = 0
			}
		}
		snapshots = append(snapshots, concurrency.UsageSnapshot{
			Policy: policy, ActiveCount: active, Available: available,
		})
	}
	return snapshots, nil
}

func (s *Store) countActiveForPolicy(ctx context.Context, policy concurrency.Policy, now time.Time) (int, error) {
	var count int
	var err error
	switch policy.Scope.Kind {
	case concurrency.ScopeGlobal:
		err = s.pool.QueryRow(ctx, countGlobalLeases, now).Scan(&count)
	case concurrency.ScopeProvider:
		err = s.pool.QueryRow(ctx, countProviderLeases, policy.Scope.ProviderID, now).Scan(&count)
	default:
		err = s.pool.QueryRow(ctx, countModelLeases, policy.Scope.ProviderID, policy.Scope.ModelID, now).Scan(&count)
	}
	if err != nil {
		return 0, err
	}
	return count, nil
}
