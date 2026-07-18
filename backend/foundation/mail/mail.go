// Package mail provides a minimal, best-effort SMTP email sender.
//
// A mailer is built from an explicit Config (the caller supplies it — e.g. the
// Job Queue scheduler loads it from the SMTP_Setup table). A mailer that is not
// enabled is a no-op, so unconfigured environments never attempt delivery.
// Sending is best-effort: callers should log a send error but must not let it
// fail the surrounding work.
package mail

import (
	"fmt"
	"net/smtp"
	"strings"
)

// Sender sends a plain-text email. Implemented by SMTPMailer and by test doubles.
type Sender interface {
	// Send delivers a plain-text message. Returns nil when sending is disabled.
	Send(to, subject, body string) error
	// Enabled reports whether the sender will actually attempt delivery.
	Enabled() bool
}

// Config holds SMTP settings.
type Config struct {
	Enabled  bool
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// SMTPMailer sends mail via net/smtp.
type SMTPMailer struct {
	cfg Config
}

// NewSMTPMailer builds a mailer from the given config.
func NewSMTPMailer(cfg Config) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

// Enabled reports whether delivery is configured and turned on.
func (m *SMTPMailer) Enabled() bool {
	return m.cfg.Enabled && m.cfg.Host != "" && m.cfg.From != ""
}

// Send delivers a plain-text email. When disabled it is a no-op and returns nil.
func (m *SMTPMailer) Send(to, subject, body string) error {
	if !m.Enabled() {
		return nil
	}
	if to == "" {
		return fmt.Errorf("mail: empty recipient")
	}

	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	addr := m.cfg.Host + ":" + m.cfg.Port
	msg := buildMessage(m.cfg.From, to, subject, body)
	if err := smtp.SendMail(addr, auth, m.cfg.From, []string{to}, msg); err != nil {
		return fmt.Errorf("mail: send to %s failed: %w", to, err)
	}
	return nil
}

// buildMessage assembles a minimal RFC 5322 plain-text message. Header values are
// stripped of CR/LF to prevent header injection from job-supplied subjects.
func buildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + sanitizeHeader(from) + "\r\n")
	b.WriteString("To: " + sanitizeHeader(to) + "\r\n")
	b.WriteString("Subject: " + sanitizeHeader(subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(body, "\n", "\r\n"))
	return []byte(b.String())
}

// sanitizeHeader removes CR/LF so a value cannot inject additional headers.
func sanitizeHeader(v string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(v)
}
