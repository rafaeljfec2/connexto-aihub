package catalog

import (
	"encoding/json"
	"strings"
	"time"

	"aihub/internal/providers"
)

// Normalized capability names. The vocabulary is closed in E03; extending it
// requires a SPEC revision.
const (
	capabilityTextGeneration   = "text_generation"
	capabilityStreaming        = "streaming"
	capabilityFunctionTools    = "function_tools"
	capabilityCustomTools      = "custom_tools"
	capabilityReasoning        = "reasoning"
	capabilityImageInput       = "image_input"
	capabilityAudioInput       = "audio_input"
	capabilityAudioOutput      = "audio_output"
	capabilityStructuredOutput = "structured_output"
)

var capabilityVocabulary = map[string]struct{}{
	capabilityTextGeneration:   {},
	capabilityStreaming:        {},
	capabilityFunctionTools:    {},
	capabilityCustomTools:      {},
	capabilityReasoning:        {},
	capabilityImageInput:       {},
	capabilityAudioInput:       {},
	capabilityAudioOutput:      {},
	capabilityStructuredOutput: {},
}

// KnownCapability reports whether a name belongs to the closed E03 vocabulary.
func KnownCapability(name string) bool {
	return knownCapability(name)
}

func knownCapability(name string) bool {
	_, ok := capabilityVocabulary[name]

	return ok
}

func supportedClaim(confidence CapabilityConfidence) CapabilityClaim {
	return CapabilityClaim{
		State:        CapabilitySupported,
		Provenance:   ProvenanceProviderCatalog,
		Confidence:   confidence,
		Verification: VerificationUnverified,
	}
}

func unsupportedClaim() CapabilityClaim {
	return CapabilityClaim{
		State:        CapabilityUnsupported,
		Provenance:   ProvenanceProviderCatalog,
		Confidence:   ConfidenceExplicit,
		Verification: VerificationUnverified,
	}
}

func unknownClaim() CapabilityClaim {
	return CapabilityClaim{
		State:        CapabilityUnknown,
		Provenance:   ProvenanceProviderCatalog,
		Confidence:   ConfidenceNone,
		Verification: VerificationUnverified,
	}
}

func booleanClaim(declared *bool) CapabilityClaim {
	if declared == nil {
		return unknownClaim()
	}

	if *declared {
		return supportedClaim(ConfidenceExplicit)
	}

	return unsupportedClaim()
}

// modalityPrefixClaim only produces a positive claim. The upstream contract
// does not guarantee that modality enumerations are exhaustive, so absence is
// unknown rather than unsupported.
func modalityPrefixClaim(modalities []string, present bool, prefix string) CapabilityClaim {
	if !present {
		return unknownClaim()
	}

	for _, modality := range modalities {
		if strings.HasPrefix(modality, prefix) {
			return supportedClaim(ConfidenceDerived)
		}
	}

	return unknownClaim()
}

func textGenerationClaim(modalities []string, present bool) CapabilityClaim {
	if !present {
		return unknownClaim()
	}

	for _, modality := range modalities {
		if modality == "text" {
			return supportedClaim(ConfidenceDerived)
		}
	}

	return unknownClaim()
}

func normalizeCapabilities(descriptor providers.ModelDescriptor) map[string]CapabilityClaim {
	return map[string]CapabilityClaim{
		capabilityTextGeneration:   textGenerationClaim(descriptor.OutputModalities, descriptor.OutputModalitiesPresent),
		capabilityStreaming:        unknownClaim(),
		capabilityFunctionTools:    booleanClaim(descriptor.ToolsSupported),
		capabilityCustomTools:      unknownClaim(),
		capabilityReasoning:        booleanClaim(descriptor.ReasoningSupported),
		capabilityImageInput:       modalityPrefixClaim(descriptor.InputModalities, descriptor.InputModalitiesPresent, "image/"),
		capabilityAudioInput:       modalityPrefixClaim(descriptor.InputModalities, descriptor.InputModalitiesPresent, "audio/"),
		capabilityAudioOutput:      modalityPrefixClaim(descriptor.OutputModalities, descriptor.OutputModalitiesPresent, "audio/"),
		capabilityStructuredOutput: booleanClaim(descriptor.StructuredOutput),
	}
}

// NormalizeEntry maps a provider descriptor into a catalog entry. Capabilities
// are always emitted with the full vocabulary so consumers see a stable shape.
func NormalizeEntry(providerID string, descriptor providers.ModelDescriptor, now time.Time) ModelEntry {
	stamp := now.UTC()
	limits := Limits{
		Context: cloneInt64(descriptor.LimitContext),
		Input:   cloneInt64(descriptor.LimitInput),
		Output:  cloneInt64(descriptor.LimitOutput),
	}

	entry := ModelEntry{
		ProviderID:       providerID,
		ModelID:          descriptor.ID,
		CanonicalID:      descriptor.CanonicalID,
		Name:             descriptor.Name,
		Type:             descriptor.Type,
		Family:           descriptor.Family,
		Limits:           limits,
		Capabilities:     normalizeCapabilities(descriptor),
		ProviderMetadata: cloneRaw(descriptor.Raw),
		FirstSeenAt:      stamp,
		LastSeenAt:       stamp,
	}

	if descriptor.InputModalitiesPresent || descriptor.OutputModalitiesPresent {
		entry.Modalities = &Modalities{
			Input:  cloneStrings(descriptor.InputModalities),
			Output: cloneStrings(descriptor.OutputModalities),
		}
	}

	return entry
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}

	cloned := make([]string, len(values))
	copy(cloned, values)

	return cloned
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}

	cloned := *value

	return &cloned
}

func cloneRaw(value json.RawMessage) json.RawMessage {
	if value == nil {
		return nil
	}

	cloned := make(json.RawMessage, len(value))
	copy(cloned, value)

	return cloned
}
