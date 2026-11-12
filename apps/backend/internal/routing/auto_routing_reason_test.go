package routing

import "testing"

func TestResolveAutoRoutingReason_Precedence(t *testing.T) {
	tests := []struct {
		reasons []string
		want    string
	}{
		{nil, "awrr_pool_select"},
		{[]string{"quota_unknown_all"}, "awrr_pool_select"},
		{[]string{"efficiency_fast_path"}, "efficiency_fast_path"},
		{[]string{"reasoning_preference_unmatched", "efficiency_fast_path"}, "reasoning_preference_unmatched"},
		{[]string{"reasoning_preferred"}, "reasoning_preferred"},
	}
	for _, tc := range tests {
		if got := resolveAutoRoutingReason(tc.reasons); got != tc.want {
			t.Fatalf("resolveAutoRoutingReason(%v) = %q, want %q", tc.reasons, got, tc.want)
		}
	}
}
