package userauth

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"net/url"
	"strings"
)

// SMTPSettings configures optional password-reset email delivery.
type SMTPSettings struct {
	Host string
	Port int
	User string
	Password string
	From string
	PublicAppURL string
}

// Enabled reports whether outbound mail can be attempted.
func (s SMTPSettings) Enabled() bool {
	return s.Host != "" && s.Port > 0 && s.From != "" && s.PublicAppURL != ""
}

type Mailer struct {
	settings SMTPSettings
	logger   *slog.Logger
}

func NewMailer(settings SMTPSettings, logger *slog.Logger) *Mailer {
	return &Mailer{settings: settings, logger: logger}
}

func (m *Mailer) SendResetLink(ctx context.Context, to, token string) error {
	if !m.settings.Enabled() {
		return errMailDisabled
	}

	base, err := url.Parse(m.settings.PublicAppURL)
	if err != nil {
		return err
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/reset-password"
	q := base.Query()
	q.Set("token", token)
	base.RawQuery = q.Encode()
	link := base.String()

	subject := "AI Hub password reset"
	body := fmt.Sprintf("Reset your password using this link (valid for 30 minutes):\r\n\r\n%s\r\n", link)
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, body))

	addr := net.JoinHostPort(m.settings.Host, fmt.Sprintf("%d", m.settings.Port))
	auth := smtp.PlainAuth("", m.settings.User, m.settings.Password, m.settings.Host)
	return smtp.SendMail(addr, auth, m.settings.From, []string{to}, msg)
}

var errMailDisabled = fmt.Errorf("smtp delivery disabled")
