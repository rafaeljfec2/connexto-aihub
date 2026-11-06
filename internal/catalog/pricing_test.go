package catalog_test

import (
	"encoding/json"
	"os"
	"testing"

	"aihub/internal/catalog"
	"aihub/internal/providers"
)

func TestExtractProviderReportedPricingOpenCodeGo(t *testing.T) {
	fixture, err := os.ReadFile("../providers/testdata/models.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(fixture, &envelope); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	var deepseekRaw json.RawMessage
	for _, raw := range envelope.Data {
		var id struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &id); err != nil {
			t.Fatalf("decode id: %v", err)
		}
		if id.ID == "deepseek-v4.1-flash" {
			deepseekRaw = raw
			break
		}
	}
	if deepseekRaw == nil {
		t.Fatal("deepseek-v4.1-flash not found in fixture")
	}

	pricing, ok := catalog.ExtractProviderReportedPricing(providers.OpenCodeGoID, deepseekRaw)
	if !ok {
		t.Fatal("expected pricing from opencode-go fixture")
	}
	if pricing.Complete != true || pricing.Unit != "per_1m_tokens" || pricing.Currency != "USD" {
		t.Fatalf("pricing = %+v", pricing)
	}
	if pricing.Input == nil || *pricing.Input != 0.15 {
		t.Fatalf("input = %v, want 0.15", pricing.Input)
	}
	if pricing.Output == nil || *pricing.Output != 0.6 {
		t.Fatalf("output = %v, want 0.6", pricing.Output)
	}
	if pricing.CacheRead == nil || *pricing.CacheRead != 0.003 {
		t.Fatalf("cache_read = %v, want 0.003", pricing.CacheRead)
	}
}

func TestExtractProviderReportedPricingOpenCodeGoFreeModel(t *testing.T) {
	raw := json.RawMessage(`{"cost":{"text":{"input":{"price":0},"output":{"price":0}}}}`)
	pricing, ok := catalog.ExtractProviderReportedPricing(providers.OpenCodeGoID, raw)
	if !ok {
		t.Fatal("expected zero pricing for free model")
	}
	if pricing.Input == nil || *pricing.Input != 0 || pricing.Output == nil || *pricing.Output != 0 {
		t.Fatalf("pricing = %+v", pricing)
	}
}

func TestExtractProviderReportedPricingOpenRouter(t *testing.T) {
	raw := json.RawMessage(`{
		"id":"anthropic/claude-3.5-sonnet",
		"pricing":{
			"prompt":"0.000003",
			"completion":"0.000015",
			"input_cache_read":"0.0000003",
			"input_cache_write":"0.00000375"
		}
	}`)
	pricing, ok := catalog.ExtractProviderReportedPricing(providers.OpenRouterID, raw)
	if !ok {
		t.Fatal("expected openrouter pricing")
	}
	if pricing.Input == nil || *pricing.Input != 3 {
		t.Fatalf("input per 1M = %v, want 3", pricing.Input)
	}
	if pricing.Output == nil || *pricing.Output != 15 {
		t.Fatalf("output per 1M = %v, want 15", pricing.Output)
	}
	if pricing.CacheRead == nil || *pricing.CacheRead != 0.3 {
		t.Fatalf("cache_read per 1M = %v, want 0.3", pricing.CacheRead)
	}
	if pricing.CacheWrite == nil || *pricing.CacheWrite != 3.75 {
		t.Fatalf("cache_write per 1M = %v, want 3.75", pricing.CacheWrite)
	}
}

func TestExtractProviderReportedPricingMissingMetadata(t *testing.T) {
	_, ok := catalog.ExtractProviderReportedPricing(providers.CursorID, json.RawMessage(`{"id":"composer-2.5"}`))
	if ok {
		t.Fatal("cursor metadata without pricing should not parse")
	}
	_, ok = catalog.ExtractProviderReportedPricing(providers.OpenRouterID, json.RawMessage(`{"id":"x"}`))
	if ok {
		t.Fatal("openrouter without pricing should not parse")
	}
}
