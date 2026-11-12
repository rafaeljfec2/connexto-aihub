package userauth

import "time"

const (
	AccessCookieName  = "aihub_ui_access"
	RefreshCookieName = "aihub_ui_refresh"

	AccessTTL   = 15 * time.Minute
	RefreshTTL  = 7 * 24 * time.Hour
	ResetTTL    = 30 * time.Minute
	RoleAdmin   = "admin"
	ModeUsers   = "users"
	ModeNone    = "none"
)

const forgotResponseMessage = "If an account exists for that email, password reset instructions were sent."
