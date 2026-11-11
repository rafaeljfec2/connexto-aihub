package catalog_test

import (
	"context"
	"os"
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/config"
	"aihub/internal/providers"
)

// TestLiveCatalogSync is the opt-in real integration check for E03. It is
// skipped unless AIHUB_LIVE_SMOKE=1 and a credential are present; mock-based
// green tests never count as real provider success.
func TestLiveCatalogSync(t *testing.T) {
	if os.Getenv("AIHUB_LIVE_SMOKE") != "1" {
		t.Skip("set AIHUB_LIVE_SMOKE=1 to run the live catalog smoke test")
	}

	apiKey := os.Getenv(config.EnvProviderOpenCodeGoAPIKey)
	if apiKey == "" {
		t.Skip("AIHUB_PROVIDER_OPENCODEGO_API_KEY is not set")
	}

	baseURL := os.Getenv(config.EnvProviderOpenCodeGoBaseURL)
	if baseURL == "" {
		baseURL = "https://opencode.ai/inference/go"
	}

	client, err := providers.NewClient(config.ProviderOpenCodeGo{
		BaseURL: baseURL,
		APIKey:  apiKey,
	}, nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	descriptors, err := client.FetchModelDescriptors(ctx)
	if err != nil {
		t.Fatalf("FetchModelDescriptors() error = %v", err)
	}

	if len(descriptors) == 0 {
		t.Fatal("live catalog returned zero models")
	}

	var deepseek *providers.ModelDescriptor

	for index := range descriptors {
		if descriptors[index].ID == "deepseek-v4.1-flash" {
			deepseek = &descriptors[index]

			break
		}
	}

	if deepseek == nil {
		t.Fatal("deepseek-v4.1-flash is missing from the live catalog")
	}

	entry := catalog.NormalizeEntry(providers.OpenCodeGoID, *deepseek, time.Now())
	if entry.Capabilities["function_tools"].State != catalog.CapabilitySupported {
		t.Fatalf("function_tools = %q, want supported", entry.Capabilities["function_tools"].State)
	}

	if entry.Capabilities["streaming"].State != catalog.CapabilityUnknown {
		t.Fatalf("streaming = %q, want unknown", entry.Capabilities["streaming"].State)
	}
}
