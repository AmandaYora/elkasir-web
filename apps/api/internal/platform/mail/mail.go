// Package mail sends plain-SMTP notification emails (net/smtp — no third-party email API SDK,
// consistent with this project's self-hosted, single-container philosophy). Used today only by
// `withdrawal`'s best-effort superadmin notification (PLAN.md §2.10, Phase B4).
package mail

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// Config mirrors config.Config.SMTP — kept as its own type so this package doesn't import
// internal/platform/config (composition root converts one to the other).
type Config struct {
	Host      string
	Port      string
	Username  string
	Password  string
	FromEmail string
	FromName  string
}

// Enabled reports whether SMTP is configured. If not, Sender.Send no-ops (returns nil) — same
// pattern already used for optional object storage.
func (c Config) Enabled() bool {
	return strings.TrimSpace(c.Host) != "" && strings.TrimSpace(c.FromEmail) != ""
}

// Sender sends plain-text notification emails over SMTP (STARTTLS via net/smtp.SendMail, which
// upgrades automatically when the server advertises the STARTTLS extension).
type Sender struct{ cfg Config }

// New builds a Sender. Safe to construct even when cfg.Enabled() is false — Send simply no-ops.
func New(cfg Config) *Sender { return &Sender{cfg: cfg} }

// Send delivers one message to every address in `to` in a single SMTP transaction. No-ops
// (returns nil) if SMTP isn't configured — callers (e.g. withdrawal's best-effort notification)
// must never let a failed/misconfigured send affect their own response.
func (s *Sender) Send(ctx context.Context, to []string, subject, body string) error {
	if !s.cfg.Enabled() || len(to) == 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%s", s.cfg.Host, s.cfg.Port)
	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	msg := fmt.Sprintf(
		"From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		s.cfg.FromName, s.cfg.FromEmail, strings.Join(to, ", "), subject, body,
	)

	return smtp.SendMail(addr, auth, s.cfg.FromEmail, to, []byte(msg))
}
