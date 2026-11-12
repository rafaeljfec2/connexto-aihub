package userauth

import (
	"strings"
	"testing"
	"time"
)

func TestAccessJWTRoundTrip(t *testing.T) {
	secret := []byte(strings.Repeat("a", 32))
	expires := time.Now().Add(AccessTTL)
	token, err := SignAccessJWT(secret, "user-1", "u@example.com", RoleAdmin, "sess-1", expires)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	claims, err := VerifyAccessJWT(secret, token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.Sub != "user-1" || claims.Email != "u@example.com" || claims.SID != "sess-1" {
		t.Fatalf("claims=%+v", claims)
	}
}

func TestVerifyAccessJWTRejectsExpired(t *testing.T) {
	secret := []byte(strings.Repeat("b", 32))
	expires := time.Now().Add(-time.Minute)
	token, err := SignAccessJWT(secret, "user-1", "u@example.com", RoleAdmin, "sess-1", expires)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := VerifyAccessJWT(secret, token); err == nil {
		t.Fatal("expected expired jwt error")
	}
}
