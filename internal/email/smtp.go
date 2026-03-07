// Package email provides SMTP-based email sending for Rampart.
package email

import (
	"fmt"
	"net/smtp"
	"strings"
)

// Config holds SMTP connection settings.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Sender sends emails via SMTP.
type Sender struct {
	cfg Config
}

// NewSender creates a new SMTP email sender.
func NewSender(cfg Config) *Sender {
	return &Sender{cfg: cfg}
}

// Send sends a plain-text email.
func (s *Sender) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	msg := strings.Join([]string{
		"From: " + s.cfg.From,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	return smtp.SendMail(addr, auth, s.cfg.From, []string{to}, []byte(msg))
}

// Enabled reports whether SMTP is configured.
func (s *Sender) Enabled() bool {
	return s.cfg.Host != "" && s.cfg.From != ""
}

// NoOpSender is a sender that logs instead of sending (for development).
type NoOpSender struct{}

// Send does nothing and returns nil.
func (n *NoOpSender) Send(_, _, _ string) error { return nil }

// Enabled returns false.
func (n *NoOpSender) Enabled() bool { return false }
