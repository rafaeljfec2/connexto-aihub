package routing

import (
	"strings"
	"time"
)

// AutoEvidenceClass distinguishes smoke, homologation, Gate 0, and production readiness.
type AutoEvidenceClass string

const (
	EvidenceSmokeHomologation AutoEvidenceClass = "smoke_homologation"
	EvidenceGate0Schema       AutoEvidenceClass = "gate0_json_schema"
	EvidenceProductionRelease AutoEvidenceClass = "production_release"
)

// AutoModelEvidence records homologation and contract proof for one allowlist model.
type AutoModelEvidence struct {
	Target                Target
	SmokeHomologated      bool
	StrictSchemaNonStream bool
	StrictSchemaStream    bool
	Gate0Approved         bool
	ProductionTrafficOK   bool
	QuotaBlockedUntil     *time.Time
	Notes                 string
}

// autoModelEvidenceTable is checked-in discovery state; no live homologation.
var autoModelEvidenceTable = map[string]AutoModelEvidence{
	targetKey(Target{ProviderID: "openrouter", ModelID: "anthropic/claude-haiku-4.5"}): {
		Target: Target{ProviderID: "openrouter", ModelID: "anthropic/claude-haiku-4.5"},
		SmokeHomologated: true, StrictSchemaNonStream: true, Gate0Approved: true, ProductionTrafficOK: false,
		Notes: "gate0_battery_2026-11-12",
	},
	targetKey(Target{ProviderID: "opencode-go", ModelID: "glm-5.3-flash"}): {
		Target: Target{ProviderID: "opencode-go", ModelID: "glm-5.3-flash"},
		SmokeHomologated: true, StrictSchemaNonStream: true, Gate0Approved: true, ProductionTrafficOK: false,
		Notes: "gate0_battery_2026-11-12",
	},
	targetKey(Target{ProviderID: "opencode-go", ModelID: "deepseek-v4.1-flash"}): {
		Target: Target{ProviderID: "opencode-go", ModelID: "deepseek-v4.1-flash"},
		SmokeHomologated: true, StrictSchemaNonStream: true, Gate0Approved: true, ProductionTrafficOK: false,
		Notes: "gate0_battery_2026-11-12",
	},
	targetKey(Target{ProviderID: "opencode-go", ModelID: "claude-haiku-5-5"}): {
		Target: Target{ProviderID: "opencode-go", ModelID: "claude-haiku-5-5"},
		SmokeHomologated: true, StrictSchemaNonStream: true, Gate0Approved: true, ProductionTrafficOK: false,
		Notes: "gate0_battery_2026-11-12",
	},
	targetKey(Target{ProviderID: "opencode-go", ModelID: "gpt-6-luna"}): {
		Target: Target{ProviderID: "opencode-go", ModelID: "gpt-6-luna"},
		SmokeHomologated: true, StrictSchemaNonStream: true, Gate0Approved: true, ProductionTrafficOK: false,
		Notes: "gate0_battery_2026-11-12",
	},
	targetKey(Target{ProviderID: "cursor", ModelID: "composer-2.5"}): {
		Target: Target{ProviderID: "cursor", ModelID: "composer-2.5"},
		SmokeHomologated: true, StrictSchemaNonStream: true, Gate0Approved: true, ProductionTrafficOK: false,
		Notes: "strict_schema_proxy_enforced_non_stream;gate0_battery_2026-11-12",
	},
	targetKey(Target{ProviderID: "antigravity", ModelID: "antigravity:gemini-3.8-flash"}): {
		Target: Target{ProviderID: "antigravity", ModelID: "antigravity:gemini-3.8-flash"},
		SmokeHomologated: true, StrictSchemaNonStream: true, Gate0Approved: true, ProductionTrafficOK: false,
		Notes: "agy_runtime_overhead~15300_tokens;gate0_battery_2026-11-12",
	},
	targetKey(Target{ProviderID: "antigravity", ModelID: "antigravity:claude-opus-4-6-thinking"}): {
		Target: Target{ProviderID: "antigravity", ModelID: "antigravity:claude-opus-4-6-thinking"},
		SmokeHomologated: false, Gate0Approved: false, ProductionTrafficOK: false,
		Notes: "removed_from_auto_allowlist_2026-11-12;retest_http_429",
	},
	targetKey(Target{ProviderID: "antigravity", ModelID: "antigravity:claude-sonnet-4-6"}): {
		Target: Target{ProviderID: "antigravity", ModelID: "antigravity:claude-sonnet-4-6"},
		SmokeHomologated: false, Gate0Approved: false, ProductionTrafficOK: false,
		Notes: "removed_from_auto_allowlist_2026-11-12;retest_http_429",
	},
	targetKey(Target{ProviderID: "antigravity", ModelID: "antigravity:gpt-oss-120b-medium"}): {
		Target: Target{ProviderID: "antigravity", ModelID: "antigravity:gpt-oss-120b-medium"},
		SmokeHomologated: false, Gate0Approved: false, ProductionTrafficOK: false,
		Notes: "removed_from_auto_allowlist_2026-11-12;retest_http_429",
	},
}

func targetKey(t Target) string {
	return t.ProviderID + "/" + t.ModelID
}

// AutoGate0Satisfied is true only when every AutoMVPAllowlist model is Gate-0 approved.
func AutoGate0Satisfied() bool {
	for _, entry := range AutoMVPAllowlist {
		ev, ok := autoModelEvidenceTable[targetKey(entry)]
		if !ok || !ev.Gate0Approved {
			return false
		}
	}
	return true
}

// LookupAutoModelEvidence returns the checked-in evidence row for an allowlist target.
func LookupAutoModelEvidence(t Target) (AutoModelEvidence, bool) {
	ev, ok := autoModelEvidenceTable[targetKey(t)]
	return ev, ok
}

func requiresStructuredOutput(req RequirementStatus) bool {
	for _, cap := range req.RequiredCapabilities {
		if cap == "structured_output" {
			return true
		}
	}
	return false
}

// FilterAutoBandBCandidates keeps allowlist targets that pass contract proofs for this request.
func FilterAutoBandBCandidates(understanding RequestUnderstanding, allowlisted []Target) ([]Target, []string) {
	if len(allowlisted) == 0 {
		return nil, nil
	}
	out := make([]Target, 0, len(allowlisted))
	reasons := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	for _, t := range allowlisted {
		ok, code := PassesAutoContractBand(understanding, t)
		if ok {
			out = append(out, t)
			continue
		}
		if code == "" {
			continue
		}
		if _, dup := seen[code]; dup {
			continue
		}
		seen[code] = struct{}{}
		reasons = append(reasons, code)
	}
	return out, reasons
}

// PassesAutoContractBand applies Band B contract proofs for the request shape.
func PassesAutoContractBand(understanding RequestUnderstanding, t Target) (bool, string) {
	ev, ok := LookupAutoModelEvidence(t)
	if !ok {
		return false, "allowlist_miss"
	}
	if ev.QuotaBlockedUntil != nil && time.Now().UTC().Before(*ev.QuotaBlockedUntil) {
		return false, "quota_exhausted"
	}
	if autoRuntimeState.quotaExcludedUntil(t).After(time.Now().UTC()) {
		return false, "quota_exhausted"
	}
	if autoRuntimeState.overloadCooldownActive(t) {
		return false, "cooldown_overload"
	}
	if !ev.SmokeHomologated {
		return false, "capability_unverified"
	}
	if requiresStructuredOutput(understanding.Requirements) {
		if understanding.IsStreaming {
			if !ev.StrictSchemaStream {
				return false, "capability_unverified"
			}
		} else if !ev.StrictSchemaNonStream {
			return false, "capability_unverified"
		}
	}
	if strings.Contains(ev.Notes, "blocked_upstream") {
		return false, "quota_exhausted"
	}
	return true, ""
}
