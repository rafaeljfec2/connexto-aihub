package ui

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	CookieName           = "aihub_ui_session"
	PrincipalID          = "ui-operator"
	defaultSessionTTL    = 24 * time.Hour
	minSessionSecretSize = 32
)

// Settings configures operator UI authentication.
type Settings struct {
	Passphrase     string
	SessionSecret  []byte
	SessionTTL     time.Duration
	SecureCookies  bool
	LoopbackServer bool
}

// Gate validates operator UI settings.
type Gate struct {
	passphraseDigest [32]byte
	passphraseSet    bool
	sessionSecret    []byte
	sessionTTL       time.Duration
	secureCookies    bool
	loopbackServer   bool
}

// NewGate builds an operator gate from settings.
func NewGate(settings Settings) (Gate, error) {
	ttl := settings.SessionTTL
	if ttl <= 0 {
		ttl = defaultSessionTTL
	}

	gate := Gate{
		sessionTTL:     ttl,
		secureCookies:  settings.SecureCookies,
		loopbackServer: settings.LoopbackServer,
	}

	if settings.Passphrase != "" {
		if len(settings.SessionSecret) < minSessionSecretSize {
			return Gate{}, errors.New("ui session secret must be at least 32 bytes when operator passphrase is set")
		}
		gate.passphraseSet = true
		gate.passphraseDigest = sha256.Sum256([]byte(settings.Passphrase))
		gate.sessionSecret = append([]byte(nil), settings.SessionSecret...)
	}

	return gate, nil
}

// PassphraseRequired reports whether UI login is mandatory.
func (g Gate) PassphraseRequired() bool {
	return g.passphraseSet
}

// LoopbackTrustEnabled reports whether loopback clients may access APIs without a session cookie.
func (g Gate) LoopbackTrustEnabled() bool {
	return g.loopbackServer && !g.passphraseSet
}

// Authenticated reports whether the request is an authenticated operator via cookie or loopback trust.
func (g Gate) Authenticated(r *http.Request) bool {
	if g.verifySessionCookie(r) {
		return true
	}
	return g.LoopbackTrustEnabled() && clientLoopback(r)
}

// Login verifies the passphrase and returns a Set-Cookie header value.
func (g Gate) Login(passphrase string) (string, error) {
	if !g.passphraseSet {
		return "", errors.New("operator passphrase is not configured")
	}

	digest := sha256.Sum256([]byte(passphrase))
	if subtle.ConstantTimeCompare(g.passphraseDigest[:], digest[:]) != 1 {
		return "", errors.New("invalid passphrase")
	}

	token, err := g.signSession(time.Now().Add(g.sessionTTL))
	if err != nil {
		return "", err
	}

	return formatCookie(token, g.secureCookies), nil
}

// LogoutCookie returns a Set-Cookie header that clears the session.
func (g Gate) LogoutCookie() string {
	return CookieName + "=; Path=/; Max-Age=0; HttpOnly; SameSite=Strict"
}

// SessionInfo describes the current operator session for the UI.
type SessionInfo struct {
	Authenticated bool   `json:"authenticated"`
	Mode          string `json:"mode"`
	System        string `json:"system"`
}

// Session returns session metadata for the request.
func (g Gate) Session(r *http.Request) SessionInfo {
	const systemOperator = "operator"
	if g.verifySessionCookie(r) {
		return SessionInfo{Authenticated: true, Mode: "passphrase", System: systemOperator}
	}
	if g.LoopbackTrustEnabled() && clientLoopback(r) {
		return SessionInfo{Authenticated: true, Mode: "loopback_trust", System: systemOperator}
	}
	return SessionInfo{Authenticated: false, Mode: "none", System: systemOperator}
}

func (g Gate) verifySessionCookie(r *http.Request) bool {
	if !g.passphraseSet {
		return false
	}

	cookie, err := r.Cookie(CookieName)
	if err != nil || cookie.Value == "" {
		return false
	}

	expiry, err := g.verifySessionToken(cookie.Value)
	if err != nil {
		return false
	}

	return time.Now().Before(expiry)
}

func (g Gate) signSession(expiry time.Time) (string, error) {
	payload := strconv.FormatInt(expiry.Unix(), 10)
	mac := hmac.New(sha256.New, g.sessionSecret)
	if _, err := mac.Write([]byte(payload)); err != nil {
		return "", err
	}
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func (g Gate) verifySessionToken(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return time.Time{}, errors.New("malformed session token")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return time.Time{}, err
	}

	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, err
	}

	mac := hmac.New(sha256.New, g.sessionSecret)
	if _, err := mac.Write(payloadBytes); err != nil {
		return time.Time{}, err
	}

	if subtle.ConstantTimeCompare(mac.Sum(nil), sigBytes) != 1 {
		return time.Time{}, errors.New("invalid session signature")
	}

	unix, err := strconv.ParseInt(string(payloadBytes), 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(unix, 0), nil
}

func formatCookie(token string, secure bool) string {
	parts := []string{
		fmt.Sprintf("%s=%s", CookieName, token),
		"Path=/",
		"HttpOnly",
		"SameSite=Strict",
		fmt.Sprintf("Max-Age=%d", int(defaultSessionTTL.Seconds())),
	}
	if secure {
		parts = append(parts, "Secure")
	}
	return strings.Join(parts, "; ")
}

func clientLoopback(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
