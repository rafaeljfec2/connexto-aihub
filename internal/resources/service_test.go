package resources_test

import (
	"context"
	"testing"
	"time"

	"aihub/internal/resources"
)

type fakeRepo struct {
	policies map[int64]resources.ResourcePolicy
	nextID   int64
	admitFn  func(context.Context, resources.AdmissionRequest, time.Time) (resources.AdmissionDecision, error)
}

func (f *fakeRepo) CreatePolicy(_ context.Context, input resources.NewPolicyInput) (resources.ResourcePolicy, error) {
	f.nextID++
	policy := resources.ResourcePolicy{
		ID: f.nextID, Scope: input.Scope, DailyLimit: input.DailyLimit, Enabled: input.Enabled,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	f.policies[policy.ID] = policy
	return policy, nil
}

func (f *fakeRepo) UpdatePolicy(_ context.Context, id int64, input resources.UpdatePolicyInput) (resources.ResourcePolicy, error) {
	policy := f.policies[id]
	if input.DailyLimit != nil {
		policy.DailyLimit = *input.DailyLimit
	}
	f.policies[id] = policy
	return policy, nil
}

func (f *fakeRepo) SetPolicyEnabled(_ context.Context, id int64, enabled bool) (resources.ResourcePolicy, error) {
	policy := f.policies[id]
	policy.Enabled = enabled
	f.policies[id] = policy
	return policy, nil
}

func (f *fakeRepo) GetPolicy(_ context.Context, id int64) (resources.ResourcePolicy, error) {
	policy, ok := f.policies[id]
	if !ok {
		return resources.ResourcePolicy{}, resources.ErrNotFound
	}
	return policy, nil
}

func (f *fakeRepo) ListPolicies(_ context.Context) ([]resources.ResourcePolicy, error) {
	out := make([]resources.ResourcePolicy, 0, len(f.policies))
	for _, policy := range f.policies {
		out = append(out, policy)
	}
	return out, nil
}

func (f *fakeRepo) GetPolicyByScope(_ context.Context, scope resources.ResourceScope) (resources.ResourcePolicy, error) {
	for _, policy := range f.policies {
		if policy.Scope.Kind == scope.Kind &&
			policy.Scope.ProviderID == scope.ProviderID &&
			policy.Scope.ModelID == scope.ModelID {
			return policy, nil
		}
	}
	return resources.ResourcePolicy{}, resources.ErrNotFound
}

func (f *fakeRepo) ListUsage(_ context.Context, query resources.UsageQuery) ([]resources.UsageSnapshot, error) {
	start, reset := resources.UsagePeriodBounds(query.PeriodStart)
	out := make([]resources.UsageSnapshot, 0, len(f.policies))
	for _, policy := range f.policies {
		if query.ProviderID != "" && policy.Scope.ProviderID != "" && policy.Scope.ProviderID != query.ProviderID {
			continue
		}
		if query.ModelID != "" && policy.Scope.ModelID != "" && policy.Scope.ModelID != query.ModelID {
			continue
		}
		out = append(out, resources.UsageSnapshot{
			Policy: policy, PeriodStart: start, ResetAt: reset, AdmittedCount: 0,
		})
	}
	return out, nil
}

func (f *fakeRepo) Admit(ctx context.Context, request resources.AdmissionRequest, now time.Time) (resources.AdmissionDecision, error) {
	if f.admitFn != nil {
		return f.admitFn(ctx, request, now)
	}
	start, reset := resources.UsagePeriodBounds(now)
	return resources.AdmissionDecision{AdmissionID: request.ID, Allowed: true, PeriodStart: start, ResetAt: reset}, nil
}

func TestUsagePeriodBoundsUsesUTCMidnight(t *testing.T) {
	instant := time.Date(2026, 11, 3, 20, 59, 0, 0, time.FixedZone("BRT", -3*3600))
	start, reset := resources.UsagePeriodBounds(instant)
	wantStart := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) {
		t.Fatalf("period start = %v, want %v", start, wantStart)
	}
	if !reset.Equal(wantStart.Add(24 * time.Hour)) {
		t.Fatalf("reset = %v, want next UTC day", reset)
	}
}

func TestAdmitRequiresProviderAndModel(t *testing.T) {
	service := resources.NewService(&fakeRepo{policies: map[int64]resources.ResourcePolicy{}})
	_, err := service.Admit(context.Background(), resources.AdmissionRequest{ProviderID: "p", ModelID: ""})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestAdmitReturnsLimitDecisionWithError(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepo{policies: map[int64]resources.ResourcePolicy{}}
	repo.admitFn = func(_ context.Context, request resources.AdmissionRequest, instant time.Time) (resources.AdmissionDecision, error) {
		start, reset := resources.UsagePeriodBounds(instant)
		return resources.AdmissionDecision{
			AdmissionID: request.ID, Allowed: false, Reason: resources.RejectionReasonLimitExceeded,
			PeriodStart: start, ResetAt: reset,
		}, nil
	}
	service := resources.NewService(repo, resources.WithClock(func() time.Time { return now }))
	decision, err := service.Admit(context.Background(), resources.AdmissionRequest{ID: "adm-1", ProviderID: "opencode-go", ModelID: "m1"})
	if !resources.IsLimitExceeded(err) {
		t.Fatalf("Admit() error = %v, want limit exceeded", err)
	}
	if decision.Allowed || decision.Reason == "" {
		t.Fatalf("decision = %+v", decision)
	}
}

func TestCreatePolicyValidatesScope(t *testing.T) {
	service := resources.NewService(&fakeRepo{policies: map[int64]resources.ResourcePolicy{}})
	_, err := service.CreatePolicy(context.Background(), resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeModel, ProviderID: "p"},
		DailyLimit: 10,
	})
	if err == nil {
		t.Fatal("expected invalid scope")
	}
}

func TestUpsertPolicyCreatesAndUpdates(t *testing.T) {
	repo := &fakeRepo{policies: map[int64]resources.ResourcePolicy{}}
	service := resources.NewService(repo)
	ctx := context.Background()

	created, err := service.UpsertPolicy(ctx, resources.UpsertPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 10, Enabled: true,
	})
	if err != nil || created.DailyLimit != 10 || !created.Enabled {
		t.Fatalf("create = %+v err %v", created, err)
	}

	updated, err := service.UpsertPolicy(ctx, resources.UpsertPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 20, Enabled: false,
	})
	if err != nil || updated.ID != created.ID || updated.DailyLimit != 20 || updated.Enabled {
		t.Fatalf("update = %+v err %v", updated, err)
	}
}

func TestUpsertPolicyRejectsInvalidLimit(t *testing.T) {
	service := resources.NewService(&fakeRepo{policies: map[int64]resources.ResourcePolicy{}})
	_, err := service.UpsertPolicy(context.Background(), resources.UpsertPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 0, Enabled: true,
	})
	if err == nil {
		t.Fatal("expected invalid limit")
	}
}
