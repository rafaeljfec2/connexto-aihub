package main

import (
	"log/slog"

	"aihub/internal/access"
	"aihub/internal/config"
	"aihub/internal/ui"
	"aihub/internal/userauth"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UIAuthBundle wires operator or users UI authentication.
type UIAuthBundle struct {
	OperatorGate access.OperatorGate
	Operator     ui.Gate
	Users        *userauth.Service
	UserGate     userauth.Gate
	UsersMode    bool
}

func buildUIAuthBundle(cfg config.Config, pool *pgxpool.Pool, logger *slog.Logger) (UIAuthBundle, error) {
	if cfg.UIUsersAuthEnabled() {
		if pool == nil {
			return UIAuthBundle{}, configMissingDatabaseForUsers()
		}
		store := userauth.NewStore(pool)
		mailer := userauth.NewMailer(userauth.SMTPSettings{
			Host: cfg.Auth.SMTPHost, Port: cfg.Auth.SMTPPort,
			User: cfg.Auth.SMTPUser, Password: cfg.Auth.SMTPPassword,
			From: cfg.Auth.SMTPFrom, PublicAppURL: cfg.Auth.PublicAppURL,
		}, logger)
		svc := userauth.NewService(store, []byte(cfg.UI.SessionSecret), cfg.UI.SecureCookies, mailer, logger)
		gate := userauth.NewGate(store, []byte(cfg.UI.SessionSecret), cfg.UI.SecureCookies)
		return UIAuthBundle{OperatorGate: gate, Users: svc, UserGate: gate, UsersMode: true}, nil
	}

	operatorGate, err := cfg.BuildOperatorGate()
	if err != nil {
		return UIAuthBundle{}, err
	}
	return UIAuthBundle{OperatorGate: operatorGate, Operator: operatorGate}, nil
}

func configMissingDatabaseForUsers() error {
	return errUsersModeNeedsDatabase
}

var errUsersModeNeedsDatabase = uiUsersDatabaseError{}

type uiUsersDatabaseError struct{}

func (uiUsersDatabaseError) Error() string {
	return "users auth mode requires a database connection"
}
