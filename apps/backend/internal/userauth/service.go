package userauth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrRateLimited        = errors.New("rate limited")
	ErrRefreshInvalid     = errors.New("refresh invalid")
	ErrResetInvalid       = errors.New("reset invalid")
	ErrSignupInvalid      = errors.New("signup invalid")
	ErrEmailRegistered    = errors.New("email already registered")
)

// Service implements users-mode authentication flows.
type Service struct {
	store        *Store
	secret       []byte
	secureCookie bool
	limiter      *Limiter
	mailer       *Mailer
	logger       *slog.Logger
}

// NewService constructs a users auth service.
func NewService(store *Store, secret []byte, secureCookie bool, mailer *Mailer, logger *slog.Logger) *Service {
	return &Service{
		store: store, secret: secret, secureCookie: secureCookie,
		limiter: NewLimiter(), mailer: mailer, logger: logger,
	}
}

type LoginResult struct {
	User            User
	AccessCookie    string
	RefreshCookie   string
}

func (s *Service) Login(ctx context.Context, r *http.Request, email, password string) (LoginResult, error) {
	if !s.limiter.AllowLogin(r) {
		return LoginResult{}, ErrRateLimited
	}

	id, hash, role, err := s.store.FindUserByEmail(ctx, email)
	if IsNotFound(err) {
		MitigateLoginTiming(password)
		if s.logger != nil {
			s.logger.Info("ui user login failed", "reason", "invalid_credentials")
		}
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, err
	}

	ok, vErr := VerifyPassword(hash, password)
	if vErr != nil || !ok {
		if s.logger != nil {
			s.logger.Info("ui user login failed", "reason", "invalid_credentials")
		}
		return LoginResult{}, ErrInvalidCredentials
	}

	return s.issueLoginResult(ctx, id, email, role)
}

func (s *Service) SignUp(ctx context.Context, r *http.Request, email, password string) (LoginResult, error) {
	if !s.limiter.AllowSignup(r) {
		return LoginResult{}, ErrRateLimited
	}
	if err := validateSignupCredentials(email, password); err != nil {
		return LoginResult{}, err
	}

	hash, err := HashPassword(password)
	if err != nil {
		return LoginResult{}, err
	}

	user, err := s.store.CreateUser(ctx, normalizeEmail(email), hash, RoleAdmin)
	if IsUniqueViolation(err) {
		return LoginResult{}, ErrEmailRegistered
	}
	if err != nil {
		return LoginResult{}, err
	}

	result, err := s.issueLoginResult(ctx, user.ID, user.Email, user.Role)
	if err != nil {
		return LoginResult{}, err
	}
	result.User = user
	return result, nil
}

func validateSignupCredentials(email, password string) error {
	norm := normalizeEmail(email)
	if norm == "" || len(password) < 8 {
		return ErrSignupInvalid
	}
	addr, err := mail.ParseAddress(norm)
	if err != nil {
		return ErrSignupInvalid
	}
	if strings.ToLower(strings.TrimSpace(addr.Address)) != norm {
		return ErrSignupInvalid
	}
	return nil
}

func (s *Service) issueLoginResult(ctx context.Context, userID, email, role string) (LoginResult, error) {
	plain, digest, err := NewOpaqueToken()
	if err != nil {
		return LoginResult{}, err
	}

	sessionID, err := s.store.InsertSession(ctx, userID, digest, time.Now().Add(RefreshTTL))
	if err != nil {
		return LoginResult{}, err
	}

	expires := time.Now().Add(AccessTTL)
	normEmail := normalizeEmail(email)
	access, err := SignAccessJWT(s.secret, userID, normEmail, role, sessionID, expires)
	if err != nil {
		return LoginResult{}, err
	}

	user := User{ID: userID, Email: normEmail, Role: role}
	return LoginResult{
		User:          user,
		AccessCookie:  formatAccessCookie(access, s.secureCookie),
		RefreshCookie: formatRefreshCookie(plain, s.secureCookie),
	}, nil
}

type RefreshResult struct {
	AccessCookie  string
	RefreshCookie string
}

func (s *Service) Refresh(ctx context.Context, r *http.Request, refreshPlain string) (RefreshResult, error) {
	row, err := s.store.LoadSessionByRefreshHash(ctx, hashOpaque(refreshPlain))
	if IsNotFound(err) {
		return RefreshResult{}, ErrRefreshInvalid
	}
	if err != nil {
		return RefreshResult{}, err
	}

	now := time.Now()
	if row.Revoked != nil {
		_ = s.store.RevokeSessionFamilyOnReuse(ctx, row.UserID)
		if s.logger != nil {
			s.logger.Warn("refresh token reuse detected", "user_id", row.UserID)
		}
		return RefreshResult{}, ErrRefreshInvalid
	}
	if now.After(row.Expires) {
		return RefreshResult{}, ErrRefreshInvalid
	}

	newPlain, newDigest, err := NewOpaqueToken()
	if err != nil {
		return RefreshResult{}, err
	}

	newSessionID, err := s.store.InsertSession(ctx, row.UserID, newDigest, now.Add(RefreshTTL))
	if err != nil {
		return RefreshResult{}, err
	}

	replaced := newSessionID
	if err := s.store.RevokeSession(ctx, row.ID, &replaced); err != nil {
		return RefreshResult{}, err
	}

	user, err := s.store.FindUserByID(ctx, row.UserID)
	if err != nil {
		return RefreshResult{}, err
	}

	expires := now.Add(AccessTTL)
	access, err := SignAccessJWT(s.secret, user.ID, user.Email, user.Role, newSessionID, expires)
	if err != nil {
		return RefreshResult{}, err
	}

	return RefreshResult{
		AccessCookie:  formatAccessCookie(access, s.secureCookie),
		RefreshCookie: formatRefreshCookie(newPlain, s.secureCookie),
	}, nil
}

func (s *Service) Logout(ctx context.Context, claims accessClaims) error {
	return s.store.RevokeAllUserSessions(ctx, claims.Sub)
}

func (s *Service) LogoutCookies() []string {
	return []string{clearAccessCookie(s.secureCookie), clearRefreshCookie(s.secureCookie)}
}

func (s *Service) ForgotPassword(ctx context.Context, r *http.Request, email string) error {
	if !s.limiter.AllowRecovery(r) {
		return ErrRateLimited
	}

	id, _, _, err := s.store.FindUserByEmail(ctx, email)
	if IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	plain, digest, err := NewOpaqueToken()
	if err != nil {
		return err
	}
	if err := s.store.InsertResetToken(ctx, id, digest, time.Now().Add(ResetTTL)); err != nil {
		return err
	}

	if s.mailer == nil || !s.mailer.settings.Enabled() {
		if s.logger != nil {
			s.logger.Info("password reset email delivery disabled")
		}
		return nil
	}

	if err := s.mailer.SendResetLink(ctx, normalizeEmail(email), plain); err != nil {
		if s.logger != nil {
			s.logger.Warn("password reset email failed", "error", err.Error())
		}
	}
	return nil
}

func ForgotMessage() string { return forgotResponseMessage }

func (s *Service) ResetPassword(ctx context.Context, r *http.Request, tokenPlain, password string) error {
	if !s.limiter.AllowRecovery(r) {
		return ErrRateLimited
	}
	if strings.TrimSpace(password) == "" {
		return ErrResetInvalid
	}

	userID, err := s.store.ConsumeResetToken(ctx, hashOpaque(tokenPlain))
	if err != nil {
		return ErrResetInvalid
	}

	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	if err := s.store.UpdatePassword(ctx, userID, hash); err != nil {
		return err
	}
	return s.store.RevokeAllUserSessions(ctx, userID)
}

// BootstrapPrimaryAdmin creates the first admin with hub data ownership, or links an existing account.
func (s *Service) BootstrapPrimaryAdmin(ctx context.Context, email, password string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil
	}

	count, err := s.store.CountUsers(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		if err := s.store.EnsureHubOwner(ctx, email); err != nil {
			if IsNotFound(err) {
				return nil
			}
			return err
		}
		if s.logger != nil {
			s.logger.Info("primary hub owner linked", "email", email)
		}
		return nil
	}
	if password == "" {
		return nil
	}

	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	user, err := s.store.CreateUserWithHubOwnership(ctx, email, hash, RoleAdmin, true)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("primary admin bootstrap failed", "error", err.Error())
		}
		return err
	}
	if s.logger != nil {
		s.logger.Info("primary hub owner created", "user_id", user.ID)
	}
	return nil
}
