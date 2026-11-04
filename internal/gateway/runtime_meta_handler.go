package gateway

import (
	"net/http"
)

// RuntimeMeta carries public execution context and data provenance without exposing secrets.
type RuntimeMeta struct {
	Environment   string `json:"environment"`
	DataOrigin    string `json:"data_origin"`
	Version       string `json:"version"`
	GatewayStatus string `json:"gateway_status"`
}

// RuntimeMetaProvider returns the current runtime context snapshot.
type RuntimeMetaProvider func() RuntimeMeta

// NewRuntimeMetaHandler serves GET /api/runtime/meta.
func NewRuntimeMetaHandler(provider RuntimeMetaProvider) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		var meta RuntimeMeta
		if provider != nil {
			meta = provider()
		}
		if meta.Environment == "" {
			meta.Environment = "development"
		}
		if meta.DataOrigin == "" {
			meta.DataOrigin = "live"
		}
		if meta.Version == "" {
			meta.Version = "PREVIEW 01"
		}
		if meta.GatewayStatus == "" {
			meta.GatewayStatus = "operational"
		}
		writeJSON(w, http.StatusOK, meta)
	})
}

