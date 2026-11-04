package gateway

import (
	"log/slog"
	"net/http"

	"aihub/internal/providers"
)

// NewProviderQuotasHandler answers GET /api/ui/providers/quotas with official quota snapshots across all accounts.
func NewProviderQuotasHandler(quotaService *providers.QuotaService, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "only GET is supported")
			return
		}

		response, err := quotaService.QueryAllAccounts(r.Context())
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

