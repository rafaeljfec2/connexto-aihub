package catalog_test

import (
	"reflect"
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
)

func TestNormalizeEntryBooleanCapabilities(t *testing.T) {
	now := testNow()

	cases := []struct {
		name             string
		tools            *bool
		reasoning        *bool
		structuredOutput *bool
		wantTools        catalog.CapabilityClaim
		wantReasoning    catalog.CapabilityClaim
		wantStructured   catalog.CapabilityClaim
	}{
		{
			name:             "all declared true",
			tools:            testBoolPtr(true),
			reasoning:        testBoolPtr(true),
			structuredOutput: testBoolPtr(true),
			wantTools:        testClaim(catalog.CapabilitySupported, catalog.ConfidenceExplicit),
			wantReasoning:    testClaim(catalog.CapabilitySupported, catalog.ConfidenceExplicit),
			wantStructured:   testClaim(catalog.CapabilitySupported, catalog.ConfidenceExplicit),
		},
		{
			name:             "all declared false",
			tools:            testBoolPtr(false),
			reasoning:        testBoolPtr(false),
			structuredOutput: testBoolPtr(false),
			wantTools:        testClaim(catalog.CapabilityUnsupported, catalog.ConfidenceExplicit),
			wantReasoning:    testClaim(catalog.CapabilityUnsupported, catalog.ConfidenceExplicit),
			wantStructured:   testClaim(catalog.CapabilityUnsupported, catalog.ConfidenceExplicit),
		},
		{
			name:             "all absent",
			tools:            nil,
			reasoning:        nil,
			structuredOutput: nil,
			wantTools:        testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantReasoning:    testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantStructured:   testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
		},
		{
			// Different values prove that no capability reads another field.
			name:             "mixed declarations",
			tools:            testBoolPtr(true),
			reasoning:        testBoolPtr(false),
			structuredOutput: nil,
			wantTools:        testClaim(catalog.CapabilitySupported, catalog.ConfidenceExplicit),
			wantReasoning:    testClaim(catalog.CapabilityUnsupported, catalog.ConfidenceExplicit),
			wantStructured:   testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			descriptor := testDescriptor()
			descriptor.ToolsSupported = testCase.tools
			descriptor.ReasoningSupported = testCase.reasoning
			descriptor.StructuredOutput = testCase.structuredOutput

			entry := catalog.NormalizeEntry(providers.OpenCodeGoID, descriptor, now)

			assertClaim(t, entry.Capabilities, "function_tools", testCase.wantTools)
			assertClaim(t, entry.Capabilities, "reasoning", testCase.wantReasoning)
			assertClaim(t, entry.Capabilities, "structured_output", testCase.wantStructured)
		})
	}
}

func TestNormalizeEntryModalityCapabilities(t *testing.T) {
	now := testNow()

	cases := []struct {
		name          string
		input         []string
		inputPresent  bool
		output        []string
		outputPresent bool
		wantText      catalog.CapabilityClaim
		wantImage     catalog.CapabilityClaim
		wantAudioIn   catalog.CapabilityClaim
		wantAudioOut  catalog.CapabilityClaim
	}{
		{
			// Image only on input and audio only on output prove direction.
			name:          "asymmetric input and output",
			input:         []string{"text", "image/png"},
			inputPresent:  true,
			output:        []string{"text", "audio/wav"},
			outputPresent: true,
			wantText:      testClaim(catalog.CapabilitySupported, catalog.ConfidenceDerived),
			wantImage:     testClaim(catalog.CapabilitySupported, catalog.ConfidenceDerived),
			wantAudioIn:   testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantAudioOut:  testClaim(catalog.CapabilitySupported, catalog.ConfidenceDerived),
		},
		{
			// Audio only on input and text only on output: audio_output must not
			// be derived from the input list.
			name:          "audio input with text output",
			input:         []string{"text", "audio/wav"},
			inputPresent:  true,
			output:        []string{"text"},
			outputPresent: true,
			wantText:      testClaim(catalog.CapabilitySupported, catalog.ConfidenceDerived),
			wantImage:     testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantAudioIn:   testClaim(catalog.CapabilitySupported, catalog.ConfidenceDerived),
			wantAudioOut:  testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
		},
		{
			name:          "text only is never unsupported",
			input:         []string{"text"},
			inputPresent:  true,
			output:        []string{"text"},
			outputPresent: true,
			wantText:      testClaim(catalog.CapabilitySupported, catalog.ConfidenceDerived),
			wantImage:     testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantAudioIn:   testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantAudioOut:  testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
		},
		{
			name:          "modalities absent",
			input:         nil,
			inputPresent:  false,
			output:        nil,
			outputPresent: false,
			wantText:      testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantImage:     testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantAudioIn:   testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantAudioOut:  testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
		},
		{
			name:          "output present without exact text token",
			input:         []string{"text"},
			inputPresent:  true,
			output:        []string{"image/png"},
			outputPresent: true,
			wantText:      testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantImage:     testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantAudioIn:   testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantAudioOut:  testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
		},
		{
			// Near miss: "text/plain" must not satisfy the exact "text" rule.
			name:          "text mime type is not text generation",
			input:         []string{"text"},
			inputPresent:  true,
			output:        []string{"text/plain"},
			outputPresent: true,
			wantText:      testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantImage:     testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantAudioIn:   testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantAudioOut:  testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
		},
		{
			// OpenRouter and peers declare bare "image" for vision-capable models.
			name:          "bare image token is supported",
			input:         []string{"text", "image"},
			inputPresent:  true,
			output:        []string{"text"},
			outputPresent: true,
			wantText:      testClaim(catalog.CapabilitySupported, catalog.ConfidenceDerived),
			wantImage:     testClaim(catalog.CapabilitySupported, catalog.ConfidenceDerived),
			wantAudioIn:   testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantAudioOut:  testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
		},
		{
			// Bare "audio" matches audio_input the same way as bare "image".
			name:          "bare audio token is supported",
			input:         []string{"text", "audio"},
			inputPresent:  true,
			output:        []string{"text"},
			outputPresent: true,
			wantText:      testClaim(catalog.CapabilitySupported, catalog.ConfidenceDerived),
			wantImage:     testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantAudioIn:   testClaim(catalog.CapabilitySupported, catalog.ConfidenceDerived),
			wantAudioOut:  testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
		},
		{
			// The rule is prefix, not substring: "x-image/png" is not image/*.
			name:          "image mime type must be a prefix",
			input:         []string{"text", "x-image/png"},
			inputPresent:  true,
			output:        []string{"text"},
			outputPresent: true,
			wantText:      testClaim(catalog.CapabilitySupported, catalog.ConfidenceDerived),
			wantImage:     testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantAudioIn:   testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
			wantAudioOut:  testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone),
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			descriptor := testDescriptor()
			descriptor.InputModalities = testCase.input
			descriptor.InputModalitiesPresent = testCase.inputPresent
			descriptor.OutputModalities = testCase.output
			descriptor.OutputModalitiesPresent = testCase.outputPresent

			entry := catalog.NormalizeEntry(providers.OpenCodeGoID, descriptor, now)

			assertClaim(t, entry.Capabilities, "text_generation", testCase.wantText)
			assertClaim(t, entry.Capabilities, "image_input", testCase.wantImage)
			assertClaim(t, entry.Capabilities, "audio_input", testCase.wantAudioIn)
			assertClaim(t, entry.Capabilities, "audio_output", testCase.wantAudioOut)

			for name, claim := range entry.Capabilities {
				if claim.State == catalog.CapabilityUnsupported {
					t.Fatalf("capability %q is unsupported; modality enumerations must never produce unsupported", name)
				}
			}
		})
	}
}

func TestNormalizeEntryAlwaysUnknownCapabilities(t *testing.T) {
	entry := catalog.NormalizeEntry(providers.OpenCodeGoID, testDescriptor(), testNow())
	want := testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone)

	assertClaim(t, entry.Capabilities, "streaming", want)
	assertClaim(t, entry.Capabilities, "custom_tools", want)
}

func TestNormalizeEntryEmitsCompleteVocabulary(t *testing.T) {
	entry := catalog.NormalizeEntry(providers.OpenCodeGoID, testDescriptor(), testNow())

	want := []string{
		"text_generation",
		"streaming",
		"function_tools",
		"custom_tools",
		"reasoning",
		"image_input",
		"audio_input",
		"audio_output",
		"structured_output",
	}

	if len(entry.Capabilities) != len(want) {
		t.Fatalf("capability count = %d, want %d", len(entry.Capabilities), len(want))
	}

	for _, name := range want {
		if _, ok := entry.Capabilities[name]; !ok {
			t.Fatalf("capability %q missing", name)
		}
	}
}

func TestNormalizeEntryPreservesFieldsAndLimits(t *testing.T) {
	now := testNow()
	descriptor := testDescriptor()

	entry := catalog.NormalizeEntry(providers.OpenCodeGoID, descriptor, now)

	if entry.ProviderID != providers.OpenCodeGoID {
		t.Fatalf("provider id = %q, want %q", entry.ProviderID, providers.OpenCodeGoID)
	}

	if entry.ModelID != descriptor.ID {
		t.Fatalf("model id = %q, want %q", entry.ModelID, descriptor.ID)
	}

	if entry.CanonicalID != descriptor.CanonicalID {
		t.Fatalf("canonical id = %q, want %q", entry.CanonicalID, descriptor.CanonicalID)
	}

	if entry.Name != descriptor.Name || entry.Type != descriptor.Type || entry.Family != descriptor.Family {
		t.Fatalf("display metadata was not preserved: %+v", entry)
	}

	if entry.Modalities == nil {
		t.Fatal("modalities = nil, want declared modalities")
	}

	if !reflect.DeepEqual(entry.Modalities.Input, descriptor.InputModalities) {
		t.Fatalf("input modalities = %v, want %v", entry.Modalities.Input, descriptor.InputModalities)
	}

	if !reflect.DeepEqual(entry.Modalities.Output, descriptor.OutputModalities) {
		t.Fatalf("output modalities = %v, want %v", entry.Modalities.Output, descriptor.OutputModalities)
	}

	if entry.Limits.Context == nil || *entry.Limits.Context != 1000000 {
		t.Fatalf("context limit = %v, want 1000000", entry.Limits.Context)
	}

	if entry.Limits.Input == nil || *entry.Limits.Input != 900000 {
		t.Fatalf("input limit = %v, want 900000", entry.Limits.Input)
	}

	if entry.Limits.Output == nil || *entry.Limits.Output != 384000 {
		t.Fatalf("output limit = %v, want 384000", entry.Limits.Output)
	}

	if !reflect.DeepEqual(entry.ProviderMetadata, descriptor.Raw) {
		t.Fatalf("provider metadata = %s, want %s", entry.ProviderMetadata, descriptor.Raw)
	}

	if !entry.FirstSeenAt.Equal(now) || !entry.LastSeenAt.Equal(now) {
		t.Fatalf("timestamps = %v/%v, want %v", entry.FirstSeenAt, entry.LastSeenAt, now)
	}

	if entry.MissingSince != nil {
		t.Fatalf("missing since = %v, want nil", entry.MissingSince)
	}
}

func TestNormalizeEntryOptionalFieldsAbsent(t *testing.T) {
	now := testNow()
	descriptor := providers.ModelDescriptor{ID: "minimal-model"}

	entry := catalog.NormalizeEntry(providers.OpenCodeGoID, descriptor, now)

	if entry.CanonicalID != "" {
		t.Fatalf("canonical id = %q, want empty", entry.CanonicalID)
	}

	if entry.Modalities != nil {
		t.Fatalf("modalities = %+v, want nil", entry.Modalities)
	}

	if entry.Limits.Context != nil || entry.Limits.Input != nil || entry.Limits.Output != nil {
		t.Fatalf("limits = %+v, want nil pointers", entry.Limits)
	}

	if entry.ProviderMetadata != nil {
		t.Fatalf("provider metadata = %s, want nil", entry.ProviderMetadata)
	}

	if len(entry.Capabilities) != 9 {
		t.Fatalf("capability count = %d, want 9", len(entry.Capabilities))
	}

	assertClaim(t, entry.Capabilities, "function_tools", testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone))
	assertClaim(t, entry.Capabilities, "image_input", testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone))
	assertClaim(t, entry.Capabilities, "text_generation", testClaim(catalog.CapabilityUnknown, catalog.ConfidenceNone))
}

func TestNormalizeEntryPreservesAbsentAndEmptyModalities(t *testing.T) {
	now := testNow()

	t.Run("both absent", func(t *testing.T) {
		descriptor := testDescriptor()
		descriptor.InputModalities = nil
		descriptor.InputModalitiesPresent = false
		descriptor.OutputModalities = nil
		descriptor.OutputModalitiesPresent = false

		entry := catalog.NormalizeEntry(providers.OpenCodeGoID, descriptor, now)

		if entry.Modalities != nil {
			t.Fatalf("modalities = %+v, want nil", entry.Modalities)
		}
	})

	t.Run("present but empty", func(t *testing.T) {
		descriptor := testDescriptor()
		descriptor.InputModalities = []string{}
		descriptor.InputModalitiesPresent = true
		descriptor.OutputModalities = nil
		descriptor.OutputModalitiesPresent = false

		entry := catalog.NormalizeEntry(providers.OpenCodeGoID, descriptor, now)

		if entry.Modalities == nil {
			t.Fatal("modalities = nil, want present")
		}

		if entry.Modalities.Input == nil || len(entry.Modalities.Input) != 0 {
			t.Fatalf("input modalities = %v, want empty non-nil slice", entry.Modalities.Input)
		}

		if entry.Modalities.Output != nil {
			t.Fatalf("output modalities = %v, want nil", entry.Modalities.Output)
		}
	})
}

func TestNormalizeEntryTimestampsAreUTC(t *testing.T) {
	zone := time.FixedZone("BRT", -3*60*60)
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, zone)

	entry := catalog.NormalizeEntry(providers.OpenCodeGoID, testDescriptor(), now)

	if entry.FirstSeenAt.Location() != time.UTC || entry.LastSeenAt.Location() != time.UTC {
		t.Fatalf("timestamps are not UTC: %v / %v", entry.FirstSeenAt.Location(), entry.LastSeenAt.Location())
	}

	if !entry.FirstSeenAt.Equal(now) || !entry.LastSeenAt.Equal(now) {
		t.Fatalf("timestamps = %v/%v, want %v", entry.FirstSeenAt, entry.LastSeenAt, now)
	}
}

func TestNormalizeEntryCopiesProviderData(t *testing.T) {
	descriptor := testDescriptor()
	originalRaw := string(descriptor.Raw)
	entry := catalog.NormalizeEntry("other-provider", descriptor, testNow())

	if entry.ProviderID != "other-provider" {
		t.Fatalf("provider id = %q, want %q", entry.ProviderID, "other-provider")
	}

	if entry.Modalities == nil {
		t.Fatal("modalities = nil, want declared modalities")
	}

	if entry.Limits.Context == nil || entry.Limits.Input == nil || entry.Limits.Output == nil {
		t.Fatalf("limits = %+v, want all three values", entry.Limits)
	}

	entry.Modalities.Input[0] = "mutated-input"
	entry.Modalities.Output[0] = "mutated-output"
	*entry.Limits.Context = 1
	*entry.Limits.Input = 2
	entry.ProviderMetadata[0] = 'X'

	if descriptor.InputModalities[0] != "text" {
		t.Fatalf("descriptor input modalities mutated: %v", descriptor.InputModalities)
	}

	if descriptor.OutputModalities[0] != "text" {
		t.Fatalf("descriptor output modalities mutated: %v", descriptor.OutputModalities)
	}

	if *descriptor.LimitContext != 1000000 || *descriptor.LimitInput != 900000 {
		t.Fatalf("descriptor limits mutated: context=%d input=%d", *descriptor.LimitContext, *descriptor.LimitInput)
	}

	if string(descriptor.Raw) != originalRaw {
		t.Fatalf("descriptor raw metadata mutated: %s", descriptor.Raw)
	}
}
