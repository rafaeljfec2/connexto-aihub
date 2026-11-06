package providers_test

import (
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
)

func TestNormalizeInputModalitiesForCatalogExpandsBareImage(t *testing.T) {
	got := providers.NormalizeInputModalitiesForCatalog([]string{"text", "image"})
	wantSubstrings := []string{"text", "image", "image/png"}
	for _, want := range wantSubstrings {
		found := false
		for _, modality := range got {
			if modality == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("NormalizeInputModalitiesForCatalog() = %v, missing %q", got, want)
		}
	}
}

func TestNormalizeInputModalitiesForCatalogPreservesExplicitMIMETypes(t *testing.T) {
	input := []string{"text", "image/jpeg", "image/png"}
	got := providers.NormalizeInputModalitiesForCatalog(input)
	if len(got) != len(input) {
		t.Fatalf("got = %v, want same length as input %v", got, input)
	}
}

func TestOpenRouterBareImageProjectsImageInputCapability(t *testing.T) {
	descriptor := providers.ModelDescriptor{
		ID:                     "anthropic/claude-sonnet-4.5",
		Name:                   "Claude Sonnet",
		InputModalities:        providers.NormalizeInputModalitiesForCatalog([]string{"text", "image"}),
		InputModalitiesPresent: true,
		OutputModalities:       []string{"text"},
		OutputModalitiesPresent: true,
	}
	entry := catalog.NormalizeEntry(providers.OpenRouterID, descriptor, time.Now())
	claim := entry.Capabilities["image_input"]
	if claim.State != catalog.CapabilitySupported {
		t.Fatalf("image_input state = %q, want supported", claim.State)
	}
}
