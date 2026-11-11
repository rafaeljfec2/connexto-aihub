package gateway

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"aihub/internal/access"
	"aihub/internal/ui"
)

type uiLoginRequest struct {
	Passphrase string `json:"passphrase"`
}

// NewUILoginHandler authenticates the operator passphrase and sets the session cookie.
func NewUILoginHandler(gate ui.Gate, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			access.WriteMethodNotAllowed(w)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 4<<10))
		if err != nil {
			access.WriteInternalError(w)
			return
		}

		var payload uiLoginRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			access.WriteInvalidRequest(w, "request body must be JSON")
			return
		}

		cookie, err := gate.Login(payload.Passphrase)
		if err != nil {
			access.WriteUnauthorized(w, "invalid operator credentials")
			return
		}

		w.Header().Set("Set-Cookie", cookie)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		if logger != nil {
			logger.Info("operator ui login succeeded")
		}
	})
}

// NewUILogoutHandler clears the operator session cookie.
func NewUILogoutHandler(gate ui.Gate) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			access.WriteMethodNotAllowed(w)
			return
		}

		if !gate.Authenticated(r) {
			access.WriteUnauthorized(w, "operator session required")
			return
		}

		w.Header().Set("Set-Cookie", gate.LogoutCookie())
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
}

// NewUISessionHandler reports whether the caller has an operator session.
func NewUISessionHandler(gate ui.Gate) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			access.WriteMethodNotAllowed(w)
			return
		}

		info := gate.Session(r)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(info)
	})
}
