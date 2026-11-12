package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
	"aihub/internal/routing"
)

// ResponsesRouting wires the E08 pipeline into Responses admission.
type ResponsesRouting struct {
	Catalog         *catalog.Manager
	Statuses        func() []providers.Status
	ProviderService *providers.Service
	StaleAfter      time.Duration
	// QuotaSnapshots supplies cached quota reads only (no synchronous upstream refresh).
	QuotaSnapshots func(ctx context.Context) []providers.ProviderQuotaSnapshot
	QuotaStaleTTL  time.Duration
}

type routingHTTPError struct {
	status  int
	typ     string
	message string
}

func (e routingHTTPError) Error() string {
	return fmt.Sprintf("%s: %s", e.typ, e.message)
}

func routeResponses(
	reqCtx context.Context,
	deps *ResponsesRouting,
	registry *providers.Registry,
	requestID string,
	explicitProvider string,
	payload []byte,
) (routing.Target, []byte, routing.RoutingDecision, error) {
	decision := routing.RoutingDecision{RequestID: requestID}

	if deps == nil || deps.Catalog == nil {
		return routing.Target{}, nil, decision, fmt.Errorf("responses routing is not configured")
	}

	structural, structuralErr := routing.ValidateStructure(payload)
	if structuralErr != nil {
		return routing.Target{}, nil, decision, routingHTTPError{
			status:  http.StatusBadRequest,
			typ:     string(routing.StructuralInvalidRequest),
			message: structuralErr.Error(),
		}
	}

	understanding, err := routing.UnderstandRequest(payload, structural)
	if err != nil {
		return routing.Target{}, nil, decision, routingHTTPError{
			status:  http.StatusBadRequest,
			typ:     invalidRequestType,
			message: "the request body is invalid",
		}
	}

	staleAfter := deps.StaleAfter
	if staleAfter <= 0 {
		staleAfter = 24 * time.Hour
	}

	statuses := deps.Statuses
	if statuses == nil {
		statuses = func() []providers.Status { return nil }
	}

	snapshot := routing.CaptureSnapshot(routing.SnapshotInput{
		Now:        time.Now().UTC(),
		StaleAfter: staleAfter,
		Admin:      buildAdminFactsForRouting(reqCtx, registry, deps.ProviderService),
		Status:     statuses(),
		Registry:   registry,
		Catalog:    deps.Catalog,
	})

	quotaTTL := deps.QuotaStaleTTL
	if quotaTTL <= 0 {
		quotaTTL = 2 * time.Minute
	}
	if deps.QuotaSnapshots != nil {
		snapshot.AutoQuota = routing.BuildAutoQuotaView(deps.QuotaSnapshots(reqCtx), snapshot.CapturedAt, quotaTTL)
	}

	routeCtx := routing.RoutingContext{
		RequestID:        requestID,
		ExplicitProvider: explicitProvider,
		Structural:       structural,
		Understanding:    understanding,
	}

	eligibility := routing.EvaluateEligibility(routeCtx, snapshot, understanding.Requirements)
	decision = routing.SelectDeterministic(routeCtx, snapshot, eligibility)
	if decision.Rejection != "" {
		if status, gate := routing.MapAutoGate0HTTPStatus(decision.Criteria); gate {
			return routing.Target{}, nil, decision, routingHTTPError{
				status:  status,
				typ:     string(decision.Rejection),
				message: "auto routing is blocked until Gate 0 is satisfied for all ten models",
			}
		}
		return routing.Target{}, nil, decision, mapRoutingRejection(decision.Rejection, explicitProvider, eligibility)
	}

	if decision.Selected == nil {
		return routing.Target{}, nil, decision, mapRoutingRejection(routing.RejectRoutingUnavailable, explicitProvider, eligibility)
	}

	prepared, err := prepareResponsesPayload(payload, structural, decision.Selected.ModelID)
	if err != nil {
		return routing.Target{}, nil, decision, routingHTTPError{
			status:  http.StatusBadRequest,
			typ:     invalidRequestType,
			message: "the request body is invalid",
		}
	}

	prepared, err = adaptResponsesPayloadForProvider(decision.Selected.ProviderID, prepared)
	if err != nil {
		return routing.Target{}, nil, decision, routingHTTPError{
			status:  http.StatusBadRequest,
			typ:     invalidRequestType,
			message: "the request body is invalid",
		}
	}

	return *decision.Selected, prepared, decision, nil
}

func prepareResponsesPayload(original []byte, structural routing.StructuralRequest, selectedModelID string) ([]byte, error) {
	if (structural.ModelPresent && !routing.IsVirtualModel(structural.ModelID)) || selectedModelID == "" {
		return original, nil
	}

	topLevel := map[string]json.RawMessage{}
	if err := json.Unmarshal(original, &topLevel); err != nil {
		return nil, err
	}

	modelRaw, err := json.Marshal(selectedModelID)
	if err != nil {
		return nil, err
	}

	topLevel["model"] = modelRaw
	return json.Marshal(topLevel)
}

func buildAdminFactsForRouting(ctx context.Context, registry *providers.Registry, service *providers.Service) []routing.AdminProviderFacts {
	if registry == nil {
		return nil
	}

	ids := registry.ProviderIDs()
	facts := make([]routing.AdminProviderFacts, 0, len(ids))
	for _, id := range ids {
		adapter, bound := registry.Resolve(id)
		fact := routing.AdminProviderFacts{ID: id, Known: bound}
		if !bound {
			facts = append(facts, fact)
			continue
		}

		fact.Known = true
		fact.Enabled = true
		hasCred := adapter.Enabled()
		if service != nil {
			if record, err := service.Get(ctx, id); err == nil {
				fact.Type = record.Type
				switch record.Type {
				case providers.TypeAntigravity:
					mgmt, _ := service.ManagementCredentialPresent(ctx, record.ID)
					hasCred = mgmt || providers.AntigravityGatewayKeyFromEnvironment() || adapter.Enabled()
				default:
					hasCred = record.HasCredential
				}
			}
		}
		fact.HasCredential = hasCred
		facts = append(facts, fact)
	}

	return facts
}

func mapRoutingRejection(code routing.RejectionCode, explicitProvider string, eligibility routing.EligibilityResult) routingHTTPError {
	if explicitProvider != "" && code == routing.RejectNoEligibleModel && explicitProviderOperationalUnavailable(explicitProvider, eligibility) {
		return routingHTTPError{
			status:  http.StatusServiceUnavailable,
			typ:     providerUnavailableType,
			message: responseProviderUnavailable,
		}
	}

	switch code {
	case routing.RejectRequirementUnmapped:
		return routingHTTPError{status: http.StatusBadRequest, typ: string(code), message: "the request requires capabilities that are not mapped for routing"}
	case routing.RejectProviderNotFound:
		return routingHTTPError{status: http.StatusNotFound, typ: responseProviderNotFoundType, message: responseProviderNotFoundMsg}
	case routing.RejectModelNotFound:
		return routingHTTPError{status: http.StatusNotFound, typ: string(code), message: "the requested model is not available"}
	case routing.RejectProviderModelConflict:
		return routingHTTPError{status: http.StatusConflict, typ: string(code), message: "the requested provider and model conflict"}
	case routing.RejectRoutingAmbiguous:
		return routingHTTPError{status: http.StatusConflict, typ: string(code), message: "routing could not select a unique model"}
	case routing.RejectRoutingUnavailable:
		return routingHTTPError{status: http.StatusServiceUnavailable, typ: string(code), message: "routing evidence is insufficient to select a model"}
	case routing.RejectNoEligibleModel:
		return routingHTTPError{status: http.StatusNotFound, typ: string(code), message: "no eligible model satisfies the request"}
	default:
		return routingHTTPError{status: http.StatusServiceUnavailable, typ: string(code), message: "routing could not select a model"}
	}
}

func explicitProviderOperationalUnavailable(providerID string, eligibility routing.EligibilityResult) bool {
	found := false

	for _, candidate := range eligibility.Evaluated {
		if candidate.Target.ProviderID != providerID {
			continue
		}

		found = true
		if candidate.Eligible {
			return false
		}

		if !providerOperationalReasonsOnly(candidate.Reasons) {
			return false
		}
	}

	return found
}

func providerOperationalReasonsOnly(reasons []routing.ReasonCode) bool {
	if len(reasons) == 0 {
		return false
	}

	for _, reason := range reasons {
		switch reason {
		case routing.ReasonProviderUnknown,
			routing.ReasonProviderDisabled,
			routing.ReasonCredentialMissing,
			routing.ReasonProviderUnbound,
			routing.ReasonResponsesUnsupported,
			routing.ReasonProviderUnavailable:
			continue
		default:
			return false
		}
	}

	return true
}

func writeRoutingFailure(w http.ResponseWriter, err error) bool {
	var failure routingHTTPError
	if !errors.As(err, &failure) {
		return false
	}

	writeJSONError(w, failure.status, failure.typ, failure.message)
	return true
}
