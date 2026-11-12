package gateway

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"aihub/internal/access"
	"aihub/internal/userauth"
)

type uiEmailLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type uiForgotPasswordRequest struct {
	Email string `json:"email"`
}

type uiResetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type uiLoginUserResponse struct {
	User userauth.User `json:"user"`
}

type uiForgotPasswordResponse struct {
	Message string `json:"message"`
}

// NewUIUsersSignUpHandler registers a new admin user and sets access and refresh cookies.
func NewUIUsersSignUpHandler(svc *userauth.Service, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 4<<10))
		if err != nil {
			access.WriteInternalError(w)
			return
		}

		var payload uiEmailLoginRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			access.WriteInvalidRequest(w, "request body must be JSON")
			return
		}

		result, err := svc.SignUp(r.Context(), r, payload.Email, payload.Password)
		switch err {
		case nil:
		case userauth.ErrRateLimited:
			access.WriteTooManyRequests(w)
			return
		case userauth.ErrSignupInvalid:
			access.WriteInvalidRequest(w, "invalid email or password")
			return
		case userauth.ErrEmailRegistered:
			access.WriteConflict(w, "email already registered")
			return
		default:
			access.WriteInternalError(w)
			return
		}

		w.Header().Add("Set-Cookie", result.AccessCookie)
		w.Header().Add("Set-Cookie", result.RefreshCookie)
		writeJSONNoStore(w, uiLoginUserResponse{User: result.User})
		if logger != nil {
			logger.Info("ui user signup succeeded", "user_id", result.User.ID)
		}
	})
}

// NewUIUsersLoginHandler authenticates email/password and sets access and refresh cookies.
func NewUIUsersLoginHandler(svc *userauth.Service, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 4<<10))
		if err != nil {
			access.WriteInternalError(w)
			return
		}

		var payload uiEmailLoginRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			access.WriteInvalidRequest(w, "request body must be JSON")
			return
		}

		result, err := svc.Login(r.Context(), r, payload.Email, payload.Password)
		switch err {
		case nil:
		case userauth.ErrRateLimited:
			access.WriteTooManyRequests(w)
			return
		case userauth.ErrInvalidCredentials:
			access.WriteUnauthorized(w, "invalid credentials")
			return
		default:
			access.WriteInternalError(w)
			return
		}

		w.Header().Add("Set-Cookie", result.AccessCookie)
		w.Header().Add("Set-Cookie", result.RefreshCookie)
		writeJSONNoStore(w, uiLoginUserResponse{User: result.User})
		if logger != nil {
			logger.Info("ui user login succeeded", "user_id", result.User.ID)
		}
	})
}

// NewUIRefreshHandler rotates the refresh cookie and issues a new access cookie.
func NewUIRefreshHandler(svc *userauth.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(userauth.RefreshCookieName)
		if err != nil || cookie.Value == "" {
			access.WriteUnauthorized(w, "refresh required")
			return
		}

		result, err := svc.Refresh(r.Context(), r, cookie.Value)
		if err != nil {
			access.WriteUnauthorized(w, "refresh invalid")
			return
		}

		w.Header().Add("Set-Cookie", result.AccessCookie)
		w.Header().Add("Set-Cookie", result.RefreshCookie)
		writeJSONNoStore(w, map[string]bool{"ok": true})
	})
}

// NewUIUsersLogoutHandler revokes refresh sessions and clears cookies.
func NewUIUsersLogoutHandler(svc *userauth.Service, gate userauth.Gate) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := gate.ClaimsFromRequest(r)
		if err == nil {
			_ = svc.Logout(r.Context(), claims)
		}
		for _, cleared := range svc.LogoutCookies() {
			w.Header().Add("Set-Cookie", cleared)
		}
		writeJSONNoStore(w, map[string]bool{"ok": true})
	})
}

// NewUIUsersSessionHandler returns the authenticated user for users mode.
func NewUIUsersSessionHandler(gate userauth.Gate) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSONNoStore(w, gate.Session(r))
	})
}

// NewUIForgotPasswordHandler triggers a reset email when SMTP is configured.
func NewUIForgotPasswordHandler(svc *userauth.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 4<<10))
		if err != nil {
			access.WriteInternalError(w)
			return
		}

		var payload uiForgotPasswordRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			access.WriteInvalidRequest(w, "request body must be JSON")
			return
		}

		if err := svc.ForgotPassword(r.Context(), r, payload.Email); err == userauth.ErrRateLimited {
			access.WriteTooManyRequests(w)
			return
		} else if err != nil {
			access.WriteInternalError(w)
			return
		}

		writeJSONNoStore(w, uiForgotPasswordResponse{Message: userauth.ForgotMessage()})
	})
}

// NewUIResetPasswordHandler consumes a reset token and updates the password.
func NewUIResetPasswordHandler(svc *userauth.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 4<<10))
		if err != nil {
			access.WriteInternalError(w)
			return
		}

		var payload uiResetPasswordRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			access.WriteInvalidRequest(w, "request body must be JSON")
			return
		}

		if err := svc.ResetPassword(r.Context(), r, payload.Token, payload.Password); err != nil {
			switch err {
			case userauth.ErrRateLimited:
				access.WriteTooManyRequests(w)
			case userauth.ErrResetInvalid:
				access.WriteUnauthorized(w, "reset invalid")
			default:
				access.WriteInternalError(w)
			}
			return
		}

		for _, cleared := range svc.LogoutCookies() {
			w.Header().Add("Set-Cookie", cleared)
		}
		writeJSONNoStore(w, map[string]bool{"ok": true})
	})
}

func writeJSONNoStore(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(payload)
}
