package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRuntimeMetaHandler(t *testing.T) {
	meta := RuntimeMeta{
		Environment:   "development",
		DataOrigin:    "live",
		Version:       "PREVIEW 01",
		GatewayStatus: "operational",
	}

	handler := NewRuntimeMetaHandler(func() RuntimeMeta {
		return meta
	})

	req := httptest.NewRequest(http.MethodGet, "/api/runtime/meta", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res RuntimeMeta
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Environment != "development" {
		t.Errorf("expected environment 'development', got '%s'", res.Environment)
	}
	if res.DataOrigin != "live" {
		t.Errorf("expected data_origin 'live', got '%s'", res.DataOrigin)
	}
	if res.Version != "PREVIEW 01" {
		t.Errorf("expected version 'PREVIEW 01', got '%s'", res.Version)
	}
	if res.GatewayStatus != "operational" {
		t.Errorf("expected gateway_status 'operational', got '%s'", res.GatewayStatus)
	}
}

