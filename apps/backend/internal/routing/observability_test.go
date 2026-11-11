package routing_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"aihub/internal/routing"
)

func TestLogDecisionSelectedContentFree(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buffer, &slog.HandlerOptions{Level: slog.LevelDebug}))

	target := routing.Target{ProviderID: "openrouter", ModelID: "model-x"}
	decision := routing.RoutingDecision{
		RequestID: "req-1",
		Selected:  &target,
		Origin:    routing.OriginExplicitModel,
		Criteria:  []string{"rule:explicit_model_unique_pair"},
		Eligibility: routing.EligibilityResult{
			Eligible: []routing.Target{target},
			Evaluated: []routing.CandidateEvaluation{{
				Target:   target,
				Eligible: true,
			}},
			Coverage: routing.CoverageEvidence{CompleteFreshCoverageForEligible: true},
		},
	}

	routing.LogDecision(context.Background(), logger, routing.DecisionLogInput{
		Decision:         decision,
		ExplicitProvider: "",
		ReachedExecution: true,
	})

	logged := buffer.String()
	if !bytes.Contains([]byte(logged), []byte("routing decision")) {
		t.Fatalf("log = %q", logged)
	}
	for _, forbidden := range []string{"hello", "input_text", "prompt", "instructions"} {
		if bytes.Contains([]byte(logged), []byte(forbidden)) {
			t.Fatalf("log leaked content %q in %q", forbidden, logged)
		}
	}
	for _, required := range []string{
		"intelligent_routing=true",
		"request_id=req-1",
		"reached_execution=true",
		"selection_origin=explicit_model",
		"provider_id=openrouter",
		"model_id=model-x",
		"eligible_count=1",
	} {
		if !bytes.Contains([]byte(logged), []byte(required)) {
			t.Fatalf("log missing %q in %q", required, logged)
		}
	}
}

func TestLogDecisionRejectedIncludesReasonCodes(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buffer, &slog.HandlerOptions{Level: slog.LevelDebug}))

	decision := routing.RoutingDecision{
		RequestID: "req-2",
		Rejection: routing.RejectNoEligibleModel,
		Criteria:  []string{"rejection:no_eligible_model"},
		Eligibility: routing.EligibilityResult{
			Evaluated: []routing.CandidateEvaluation{{
				Target:         routing.Target{ProviderID: "openrouter", ModelID: "m1"},
				Eligible:       false,
				Reasons:        []routing.ReasonCode{routing.ReasonCapabilityUnsupported},
				ExclusionClass: routing.ExclusionIncompatible,
			}},
		},
	}

	routing.LogDecision(context.Background(), logger, routing.DecisionLogInput{
		Decision:         decision,
		ExplicitProvider: "openrouter",
		ReachedExecution: false,
	})

	logged := buffer.String()
	if !bytes.Contains([]byte(logged), []byte("rejection=no_eligible_model")) {
		t.Fatalf("log = %q", logged)
	}
	if !bytes.Contains([]byte(logged), []byte("excluded_reason_codes")) ||
		!bytes.Contains([]byte(logged), []byte("capability_unsupported")) {
		t.Fatalf("log = %q", logged)
	}
	if !bytes.Contains([]byte(logged), []byte("reached_execution=false")) {
		t.Fatalf("log = %q", logged)
	}
}

func TestRejectFillsDefaultCriteria(t *testing.T) {
	decision := routing.RoutingDecision{RequestID: "req"}
	// invoke through SelectDeterministic path: unknown provider
	ctx := routing.RoutingContext{RequestID: "req", ExplicitProvider: "missing"}
	eligibility := routing.EligibilityResult{}
	decision = routing.SelectDeterministic(ctx, routing.Snapshot{}, eligibility)
	if len(decision.Criteria) == 0 {
		t.Fatal("expected default criteria on rejection")
	}
	if decision.Criteria[0] != "rejection:provider_not_found" {
		t.Fatalf("criteria = %v", decision.Criteria)
	}
}
