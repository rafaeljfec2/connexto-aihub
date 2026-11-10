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

// RequestUnderstanding holds deterministic features extracted from a responses payload.
type RequestUnderstanding struct {
	IsVirtual            bool
	VirtualTarget        string
	IsStreaming          bool
	EstimatedInputTokens int
	OptimizationProfile  OptimizationProfile
	Requirements         RequirementStatus
}

type understandingEnvelope struct {
	Stream  *bool           `json:"stream"`
	Profile *string         `json:"optimization_profile"`
	Input   json.RawMessage `json:"input"`
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
		OptimizationProfile:  profile,
		Requirements:         requirements,
	}, nil
}

