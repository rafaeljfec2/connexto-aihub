package routing

import (
	"encoding/json"
	"strings"
)

// OptimizationProfile represents the routing strategy.
type OptimizationProfile string

const (
	ProfileBalanced     OptimizationProfile = "balanced"
	ProfileEfficiency   OptimizationProfile = "efficiency"
	ProfileIntelligence OptimizationProfile = "intelligence"
)

// ReasoningEffort is the structural reasoning.effort field when present.
type ReasoningEffort string

const (
	ReasoningEffortLow    ReasoningEffort = "low"
	ReasoningEffortMedium ReasoningEffort = "medium"
	ReasoningEffortHigh   ReasoningEffort = "high"
	ReasoningEffortXHigh  ReasoningEffort = "xhigh"
)

// RequestUnderstanding holds deterministic features extracted from a responses payload.
type RequestUnderstanding struct {
	IsVirtual            bool
	VirtualTarget        string
	IsStreaming          bool
	EstimatedInputTokens int
	InputItemCount       int
	ReasoningEffort      ReasoningEffort
	OptimizationProfile  OptimizationProfile
	Requirements         RequirementStatus
}

type understandingEnvelope struct {
	Stream    *bool           `json:"stream"`
	Profile   *string         `json:"optimization_profile"`
	Input     json.RawMessage `json:"input"`
	Reasoning json.RawMessage `json:"reasoning"`
}

// UnderstandRequest performs deterministic structural feature extraction on the request.
func UnderstandRequest(body []byte, structural StructuralRequest) (RequestUnderstanding, error) {
	requirements, err := ProjectRequirements(body, structural)
	if err != nil {
		return RequestUnderstanding{}, err
	}

	var env understandingEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return RequestUnderstanding{}, err
	}

	isStreaming := false
	if env.Stream != nil {
		isStreaming = *env.Stream
	}

	isVirtual := structural.ModelPresent && IsVirtualModel(structural.ModelID)
	virtualTarget := ""
	if isVirtual {
		virtualTarget = structural.ModelID
	}

	profile := ProfileBalanced
	if env.Profile != nil {
		switch strings.ToLower(*env.Profile) {
		case "efficiency":
			profile = ProfileEfficiency
		case "intelligence":
			profile = ProfileIntelligence
		default:
			profile = ProfileBalanced
		}
	}

	// Heuristic estimation: ~4 chars per token from raw input or body
	estimatedTokens := 0
	if len(env.Input) > 0 {
		estimatedTokens = len(env.Input) / 4
	} else if len(body) > 0 {
		estimatedTokens = len(body) / 4
	}
	if estimatedTokens <= 0 && len(body) > 0 {
		estimatedTokens = 1
	}

	return RequestUnderstanding{
		IsVirtual:            isVirtual,
		VirtualTarget:        virtualTarget,
		IsStreaming:          isStreaming,
		EstimatedInputTokens: estimatedTokens,
		InputItemCount:       countInputItems(env.Input),
		ReasoningEffort:      parseReasoningEffort(env.Reasoning),
		OptimizationProfile:  profile,
		Requirements:         requirements,
	}, nil
}

func countInputItems(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	if raw[0] == '"' {
		return 1
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return 0
	}
	return len(items)
}

func parseReasoningEffort(raw json.RawMessage) ReasoningEffort {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	var object struct {
		Effort *string `json:"effort"`
	}
	if err := json.Unmarshal(raw, &object); err != nil || object.Effort == nil {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(*object.Effort)) {
	case "low":
		return ReasoningEffortLow
	case "medium":
		return ReasoningEffortMedium
	case "high":
		return ReasoningEffortHigh
	case "xhigh":
		return ReasoningEffortXHigh
	default:
		return ReasoningEffort(strings.ToLower(strings.TrimSpace(*object.Effort)))
	}
}

