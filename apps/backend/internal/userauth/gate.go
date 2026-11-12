package userauth

import (
	"net/http"
)

// SessionInfo is returned by GET /api/ui/session in users mode.
type SessionInfo struct {
	Authenticated bool   `json:"authenticated"`
	Mode          string `json:"mode"`
	System        string `json:"system"`
	User          *User  `json:"user,omitempty"`
}

// Gate authorizes API calls via the access JWT cookie.
type Gate struct {
	secret       []byte
	store        *Store
	secureCookie bool
}

func NewGate(store *Store, secret []byte, secureCookie bool) Gate {
	return Gate{store: store, secret: secret, secureCookie: secureCookie}
}

// Authenticated implements access.OperatorGate.
func (g Gate) Authenticated(r *http.Request) bool {
	_, err := g.accessClaims(r)
	return err == nil
}

func (g Gate) accessClaims(r *http.Request) (accessClaims, error) {
	cookie, err := r.Cookie(AccessCookieName)
	if err != nil || cookie.Value == "" {
		return accessClaims{}, err
	}
	return VerifyAccessJWT(g.secret, cookie.Value)
}

// Session builds the session payload for the UI.
func (g Gate) Session(r *http.Request) SessionInfo {
	claims, err := g.accessClaims(r)
	if err != nil {
		return SessionInfo{Authenticated: false, Mode: ModeNone, System: ModeUsers}
	}

	user, uErr := g.store.FindUserByID(r.Context(), claims.Sub)
	if uErr != nil {
		return SessionInfo{Authenticated: false, Mode: ModeNone, System: ModeUsers}
	}

	return SessionInfo{
		Authenticated: true,
		Mode:          ModeUsers,
		System:        ModeUsers,
		User:          &user,
	}
}

// ClaimsFromRequest returns validated access claims when present.
func (g Gate) ClaimsFromRequest(r *http.Request) (accessClaims, error) {
	return g.accessClaims(r)
}
