package userauth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/db"
	"aihub/internal/db/dbtest"
	"aihub/internal/gateway"
	"aihub/internal/userauth"
)

func TestUsersLoginRefreshAndReuseRevokesFamily(t *testing.T) {
	pool := dbtest.Pool(t)
	if _, _, err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	store := userauth.NewStore(pool)
	secret := strings.Repeat("s", 32)
	hash, err := userauth.HashPassword("correct-horse")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	user, err := store.CreateUser(context.Background(), "admin@example.com", hash, userauth.RoleAdmin)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	svc := userauth.NewService(store, []byte(secret), false, nil, nil)
	gate := userauth.NewGate(store, []byte(secret), false)

	loginHandler := gateway.NewUIUsersLoginHandler(svc, nil)
	refreshHandler := gateway.NewUIRefreshHandler(svc)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/ui/login", strings.NewReader(`{"email":"admin@example.com","password":"correct-horse"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginReq.RemoteAddr = "127.0.0.1:1234"
	loginRec := httptest.NewRecorder()
	loginHandler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", loginRec.Code, loginRec.Body.String())
	}

	var loginBody struct {
		User userauth.User `json:"user"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginBody); err != nil || loginBody.User.ID != user.ID {
		t.Fatalf("login body: %+v err=%v", loginBody, err)
	}

	cookies := loginRec.Result().Cookies()
	var refresh string
	for _, c := range cookies {
		if c.Name == userauth.RefreshCookieName {
			refresh = c.Value
		}
	}
	if refresh == "" {
		t.Fatal("missing refresh cookie")
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/ui/refresh", nil)
	refreshReq.AddCookie(&http.Cookie{Name: userauth.RefreshCookieName, Value: refresh})
	refreshRec := httptest.NewRecorder()
	refreshHandler.ServeHTTP(refreshRec, refreshReq)
	if refreshRec.Code != http.StatusOK {
		t.Fatalf("refresh status=%d", refreshRec.Code)
	}

	oldRefresh := refresh
	for _, c := range refreshRec.Result().Cookies() {
		if c.Name == userauth.RefreshCookieName {
			refresh = c.Value
		}
	}

	reuseReq := httptest.NewRequest(http.MethodPost, "/api/ui/refresh", nil)
	reuseReq.AddCookie(&http.Cookie{Name: userauth.RefreshCookieName, Value: oldRefresh})
	reuseRec := httptest.NewRecorder()
	refreshHandler.ServeHTTP(reuseRec, reuseReq)
	if reuseRec.Code != http.StatusUnauthorized {
		t.Fatalf("reuse refresh status=%d want 401", reuseRec.Code)
	}

	validAfterReuse := httptest.NewRequest(http.MethodPost, "/api/ui/refresh", nil)
	validAfterReuse.AddCookie(&http.Cookie{Name: userauth.RefreshCookieName, Value: refresh})
	validRec := httptest.NewRecorder()
	refreshHandler.ServeHTTP(validRec, validAfterReuse)
	if validRec.Code != http.StatusUnauthorized {
		t.Fatalf("rotated refresh after family revoke status=%d want 401", validRec.Code)
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/api/ui/session", nil)
	for _, c := range refreshRec.Result().Cookies() {
		if c.Name == userauth.AccessCookieName {
			sessionReq.AddCookie(c)
		}
	}
	sessionHandler := gateway.NewUIUsersSessionHandler(gate)
	sessionRec := httptest.NewRecorder()
	sessionHandler.ServeHTTP(sessionRec, sessionReq)
	if sessionRec.Code != http.StatusOK {
		t.Fatalf("session status=%d", sessionRec.Code)
	}
}

func TestForgotPasswordAlwaysGeneric(t *testing.T) {
	pool := dbtest.Pool(t)
	if _, _, err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := userauth.NewStore(pool)
	svc := userauth.NewService(store, []byte(strings.Repeat("k", 32)), false, nil, nil)
	handler := gateway.NewUIForgotPasswordHandler(svc)

	for _, email := range []string{"missing@example.com", "x"} {
		req := httptest.NewRequest(http.MethodPost, "/api/ui/forgot-password", bytes.NewBufferString(`{"email":"`+email+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.0.0.2:4321"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), userauth.ForgotMessage()) {
			t.Fatalf("body=%s", rec.Body.String())
		}
	}
}

func TestResetPasswordSingleUse(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	if _, _, err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := userauth.NewStore(pool)
	hash, _ := userauth.HashPassword("old-password")
	user, err := store.CreateUser(ctx, "reset@example.com", hash, userauth.RoleAdmin)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	plain, digest, err := userauth.NewOpaqueToken()
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if err := store.InsertResetToken(ctx, user.ID, digest, time.Now().Add(userauth.ResetTTL)); err != nil {
		t.Fatalf("insert reset: %v", err)
	}

	svc := userauth.NewService(store, []byte(strings.Repeat("z", 32)), false, nil, nil)
	handler := gateway.NewUIResetPasswordHandler(svc)

	body := `{"token":"` + plain + `","password":"new-password-ok"}`
	req := httptest.NewRequest(http.MethodPost, "/api/ui/reset-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:9000"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("reset status=%d body=%s", rec.Code, rec.Body.String())
	}

	again := httptest.NewRequest(http.MethodPost, "/api/ui/reset-password", strings.NewReader(body))
	again.Header.Set("Content-Type", "application/json")
	againRec := httptest.NewRecorder()
	handler.ServeHTTP(againRec, again)
	if againRec.Code != http.StatusUnauthorized {
		t.Fatalf("second reset status=%d want 401", againRec.Code)
	}
}
