package gateway

import (
	"net/http"
	"strings"
	"testing"

	"aihub/internal/routing"
)

func TestPrepareResponsesPayloadPreservesExplicitModel(t *testing.T) {
	original := []byte(`{"model":"keep-me","input":"hello"}`)
	prepared, err := prepareResponsesPayload(original, routing.StructuralRequest{
		ModelPresent: true,
		ModelID:      "keep-me",
	}, "other-model")
	if err != nil {
		t.Fatal(err)
	}
	if string(prepared) != string(original) {
		t.Fatalf("prepared = %s, want original preserved", prepared)
	}
}

func TestPrepareResponsesPayloadInjectsSelectedModel(t *testing.T) {
	original := []byte(`{"input":[{"type":"input_text","text":"hello"}]}`)
	prepared, err := prepareResponsesPayload(original, routing.StructuralRequest{}, "selected-model")
	if err != nil {
		t.Fatal(err)
	}
	if string(prepared) == string(original) {
		t.Fatal("expected model injection")
	}
	if !strings.Contains(string(prepared), "selected-model") {
		t.Fatalf("prepared = %s", prepared)
	}
}

func TestMapRoutingRejectionHTTPStatuses(t *testing.T) {
	cases := []struct {
		code             routing.RejectionCode
		explicitProvider string
		eligibility      routing.EligibilityResult
		status           int
		typ              string
	}{
		{routing.RejectRequirementUnmapped, "", routing.EligibilityResult{}, http.StatusBadRequest, "requirement_unmapped"},
		{routing.RejectProviderNotFound, "missing", routing.EligibilityResult{}, http.StatusNotFound, responseProviderNotFoundType},
		{routing.RejectModelNotFound, "", routing.EligibilityResult{}, http.StatusNotFound, "model_not_found"},
		{routing.RejectProviderModelConflict, "p1", routing.EligibilityResult{}, http.StatusConflict, "provider_model_conflict"},
		{routing.RejectRoutingAmbiguous, "", routing.EligibilityResult{}, http.StatusConflict, "routing_ambiguous"},
		{routing.RejectRoutingUnavailable, "", routing.EligibilityResult{}, http.StatusServiceUnavailable, "routing_unavailable"},
		{routing.RejectNoEligibleModel, "", routing.EligibilityResult{}, http.StatusNotFound, "no_eligible_model"},
		{
			routing.RejectNoEligibleModel,
			"openrouter",
			routing.EligibilityResult{
				Evaluated: []routing.CandidateEvaluation{{
					Target:  routing.Target{ProviderID: "openrouter", ModelID: "m1"},
					Reasons: []routing.ReasonCode{routing.ReasonCredentialMissing},
				}},
			},
			http.StatusServiceUnavailable,
			providerUnavailableType,
		},
	}

	for _, testCase := range cases {
		mapped := mapRoutingRejection(testCase.code, testCase.explicitProvider, testCase.eligibility)
		if mapped.status != testCase.status || mapped.typ != testCase.typ {
			t.Fatalf("%s mapped = %+v, want status=%d type=%q", testCase.code, mapped, testCase.status, testCase.typ)
		}
	}
}

func TestExplicitProviderOperationalUnavailable(t *testing.T) {
	eligibility := routing.EligibilityResult{
		Evaluated: []routing.CandidateEvaluation{
			{
				Target:  routing.Target{ProviderID: "openrouter", ModelID: "m1"},
				Reasons: []routing.ReasonCode{routing.ReasonCredentialMissing},
			},
		},
	}

	if !explicitProviderOperationalUnavailable("openrouter", eligibility) {
		t.Fatal("expected operational unavailable for credential_missing")
	}

	eligibility.Evaluated[0].Reasons = []routing.ReasonCode{routing.ReasonCapabilityUnsupported}
	if explicitProviderOperationalUnavailable("openrouter", eligibility) {
		t.Fatal("capability unsupported must not map as operational-only")
	}
}
