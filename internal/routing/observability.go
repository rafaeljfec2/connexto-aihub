package routing

import (
	"context"
	"log/slog"
	"sort"
)

const maxExcludedReasonCodes = 32

// DecisionLogInput is content-free routing evidence for structured logs (contract §8).
type DecisionLogInput struct {
	Decision            RoutingDecision
	ExplicitProvider    string
	StructuralRejection string
	ReachedExecution    bool
}

// LogDecision emits a bounded routing_decision record. It must never include prompts,
// payloads, credentials, or provider response content.
func LogDecision(ctx context.Context, logger *slog.Logger, input DecisionLogInput) {
	if logger == nil {
		return
	}

	level := slog.LevelInfo
	rejection := string(input.Decision.Rejection)
	if rejection != "" || input.StructuralRejection != "" {
		level = slog.LevelWarn
	}

	attrs := []any{
		"intelligent_routing", true,
		"request_id", input.Decision.RequestID,
		"explicit_provider", input.ExplicitProvider,
		"reached_execution", input.ReachedExecution,
		"eligible_count", len(input.Decision.Eligibility.Eligible),
		"evaluated_count", len(input.Decision.Eligibility.Evaluated),
		"coverage_complete_fresh", input.Decision.Eligibility.Coverage.CompleteFreshCoverageForEligible,
		"requirement_unmapped", input.Decision.Eligibility.RequirementUnmapped,
	}

	if input.StructuralRejection != "" {
		attrs = append(attrs, "structural_rejection", input.StructuralRejection)
	}

	if rejection != "" {
		attrs = append(attrs, "rejection", rejection)
	}

	if len(input.Decision.Criteria) > 0 {
		attrs = append(attrs, "criteria", input.Decision.Criteria)
	}

	if len(input.Decision.Eligibility.Requirements.RequiredCapabilities) > 0 {
		attrs = append(attrs,
			"required_capabilities", input.Decision.Eligibility.Requirements.RequiredCapabilities)
	}

	excluded := excludedReasonCodes(input.Decision.Eligibility)
	if len(excluded) > 0 {
		attrs = append(attrs, "excluded_reason_codes", excluded)
	}

	if input.Decision.Selected != nil {
		attrs = append(attrs,
			"provider_id", input.Decision.Selected.ProviderID,
			"model_id", input.Decision.Selected.ModelID,
			"selection_origin", string(input.Decision.Origin),
		)
	}

	logger.Log(ctx, level, "routing decision", attrs...)
}

func excludedReasonCodes(result EligibilityResult) []string {
	if len(result.Evaluated) == 0 {
		return nil
	}

	seen := make(map[ReasonCode]struct{})
	for _, candidate := range result.Evaluated {
		if candidate.Eligible {
			continue
		}

		for _, reason := range candidate.Reasons {
			seen[reason] = struct{}{}
		}
	}

	if len(seen) == 0 {
		return nil
	}

	codes := make([]string, 0, len(seen))
	for reason := range seen {
		codes = append(codes, string(reason))
	}

	sort.Strings(codes)
	if len(codes) > maxExcludedReasonCodes {
		codes = codes[:maxExcludedReasonCodes]
	}

	return codes
}
