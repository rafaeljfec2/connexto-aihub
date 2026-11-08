package gateway

import (
	"log/slog"
	"net/http"

	"aihub/internal/providers"
)

// NewProviderQuotasHandler answers GET /api/ui/providers/quotas with official quota snapshots across all accounts.
// By default, it returns the latest cached snapshots immediately and triggers background refresh if stale or missing.
// If query parameter sync=true is supplied, it performs a blocking synchronous refresh across all accounts.
func NewProviderQuotasHandler(quotaService *providers.QuotaService, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "only GET is supported")
			return
		}

		if quotaService == nil {
			writeJSONError(w, http.StatusInternalServerError, "quota_service_unavailable", "quota service not configured")
			return
		}

		isSync := r.URL.Query().Get("sync") == "true" || r.URL.Query().Get("mode") == "sync"
		forceRefresh := r.URL.Query().Get("refresh") == "true"

		var response providers.ProviderQuotasResponse
		var err error

		if isSync {
			response, err = quotaService.QueryAllAccounts(r.Context())
		} else {
			response, err = quotaService.GetSnapshots(r.Context(), providers.GetSnapshotsOptions{ForceRefresh: forceRefresh})
		}

		if err != nil {
			if logger != nil {
				logger.Error("failed to query provider quotas", slog.Any("error", err))
			}
			writeJSONError(w, http.StatusInternalServerError, "quota_query_failed", "falha ao consultar franquias dos provedores")
			return
		}

		writeJSON(w, http.StatusOK, response)
	})
}
