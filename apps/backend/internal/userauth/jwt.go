package userauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type accessClaims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Role  string `json:"role"`
	SID   string `json:"sid"`
	Exp   int64  `json:"exp"`
}

// SignAccessJWT builds an HS256 JWT for the access cookie.
func SignAccessJWT(secret []byte, userID, email, role, sessionID string, expires time.Time) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims := accessClaims{
		Sub: userID, Email: email, Role: role, SID: sessionID, Exp: expires.Unix(),
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	unsigned := header + "." + payload
	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write([]byte(unsigned)); err != nil {
		return "", err
	}
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return unsigned + "." + sig, nil
}

// VerifyAccessJWT validates the JWT signature and expiry.
func VerifyAccessJWT(secret []byte, token string) (accessClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return accessClaims{}, errors.New("malformed jwt")
	}

	unsigned := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return accessClaims{}, err
	}

	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write([]byte(unsigned)); err != nil {
		return accessClaims{}, err
	}
	if !hmac.Equal(mac.Sum(nil), sig) {
		return accessClaims{}, errors.New("invalid jwt signature")
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return accessClaims{}, err
	}

	var claims accessClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return accessClaims{}, err
	}

	if claims.Exp == 0 || time.Now().Unix() >= claims.Exp {
		return accessClaims{}, errors.New("jwt expired")
	}

	if claims.Sub == "" || claims.SID == "" {
		return accessClaims{}, fmt.Errorf("jwt missing subject or session")
	}

	return claims, nil
}
