package userauth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User is a persisted UI account.
type User struct {
	ID    string
	Email string
	Role  string
}

// Store performs parameterized auth queries against PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore attaches a store to a connection pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT count(*)::int FROM users`).Scan(&count)
	return count, err
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash, role string) (User, error) {
	return s.CreateUserWithHubOwnership(ctx, email, passwordHash, role, false)
}

func (s *Store) CreateUserWithHubOwnership(ctx context.Context, email, passwordHash, role string, ownsHubData bool) (User, error) {
	id := uuid.NewString()
	email = normalizeEmail(email)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, role, status, owns_hub_data)
		VALUES ($1, $2, $3, $4, 'active', $5)`,
		id, email, passwordHash, role, ownsHubData,
	)
	if err != nil {
		return User{}, err
	}
	return User{ID: id, Email: email, Role: role}, nil
}

func (s *Store) EnsureHubOwner(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	tag, err := s.pool.Exec(ctx, `
		UPDATE users SET owns_hub_data = true, updated_at = now()
		WHERE email = $1 AND status = 'active'`,
		email,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) FindUserByEmail(ctx context.Context, email string) (id, passwordHash, role string, err error) {
	email = normalizeEmail(email)
	err = s.pool.QueryRow(ctx, `
		SELECT id, password_hash, role FROM users
		WHERE email = $1 AND status = 'active'`,
		email,
	).Scan(&id, &passwordHash, &role)
	return id, passwordHash, role, err
}

func (s *Store) FindUserByID(ctx context.Context, id string) (User, error) {
	var user User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, role FROM users WHERE id = $1 AND status = 'active'`,
		id,
	).Scan(&user.ID, &user.Email, &user.Role)
	return user, err
}

type sessionRow struct {
	ID       string
	UserID   string
	Expires  time.Time
	Revoked  *time.Time
}

func (s *Store) InsertSession(ctx context.Context, userID string, refreshHash []byte, expires time.Time) (sessionID string, err error) {
	sessionID = uuid.NewString()
	_, err = s.pool.Exec(ctx, `
		INSERT INTO user_sessions (id, user_id, refresh_token_hash, expires_at)
		VALUES ($1, $2, $3, $4)`,
		sessionID, userID, refreshHash, expires,
	)
	return sessionID, err
}

func (s *Store) LoadSessionByRefreshHash(ctx context.Context, refreshHash []byte) (sessionRow, error) {
	var row sessionRow
	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, expires_at, revoked_at
		FROM user_sessions WHERE refresh_token_hash = $1`,
		refreshHash,
	).Scan(&row.ID, &row.UserID, &row.Expires, &row.Revoked)
	return row, err
}

func (s *Store) RevokeSession(ctx context.Context, sessionID string, replacedBy *string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE user_sessions SET revoked_at = now(), replaced_by = $2
		WHERE id = $1 AND revoked_at IS NULL`,
		sessionID, replacedBy,
	)
	return err
}

func (s *Store) RevokeAllUserSessions(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE user_sessions SET revoked_at = now()
		WHERE user_id = $1 AND revoked_at IS NULL`,
		userID,
	)
	return err
}

func (s *Store) RevokeSessionFamilyOnReuse(ctx context.Context, userID string) error {
	return s.RevokeAllUserSessions(ctx, userID)
}

func (s *Store) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`,
		userID, passwordHash,
	)
	return err
}

func (s *Store) InsertResetToken(ctx context.Context, userID string, tokenHash []byte, expires time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		userID, tokenHash, expires,
	)
	return err
}

func (s *Store) ConsumeResetToken(ctx context.Context, tokenHash []byte) (userID string, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var expires time.Time
	var used *time.Time
	err = tx.QueryRow(ctx, `
		SELECT user_id, expires_at, used_at FROM password_reset_tokens
		WHERE token_hash = $1 FOR UPDATE`,
		tokenHash,
	).Scan(&userID, &expires, &used)
	if err != nil {
		return "", err
	}
	if used != nil || time.Now().After(expires) {
		return "", errors.New("reset token invalid")
	}

	_, err = tx.Exec(ctx, `UPDATE password_reset_tokens SET used_at = now() WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return userID, nil
}

// IsNotFound reports pgx.ErrNoRows.
func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
