package notification

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

type PasswordResetSender interface {
	SendPasswordReset(ctx context.Context, contact, resetURL string, expiresAt time.Time) error
}

type SMTPPasswordResetSender struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPPasswordResetSender(host, port, username, password, from string) *SMTPPasswordResetSender {
	return &SMTPPasswordResetSender{host: host, port: port, username: username, password: password, from: from}
}

func (s *SMTPPasswordResetSender) SendPasswordReset(_ context.Context, contact, resetURL string, expiresAt time.Time) error {
	recipient, err := mail.ParseAddress(strings.TrimSpace(contact))
	if err != nil || !strings.EqualFold(recipient.Address, strings.TrimSpace(contact)) {
		return fmt.Errorf("password reset contact is not an email address")
	}
	from, err := mail.ParseAddress(s.from)
	if err != nil {
		return fmt.Errorf("parse password reset sender: %w", err)
	}
	message := strings.Join([]string{
		"From: " + from.String(),
		"To: " + recipient.String(),
		"Subject: Soulmark password reset",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		"Use this link to reset your password:",
		resetURL,
		"",
		"This link expires at " + expiresAt.UTC().Format(time.RFC3339) + ".",
	}, "\r\n")
	auth := smtp.PlainAuth("", s.username, s.password, s.host)
	if err := smtp.SendMail(net.JoinHostPort(s.host, s.port), auth, from.Address, []string{recipient.Address}, []byte(message)); err != nil {
		return fmt.Errorf("send password reset email: %w", err)
	}
	return nil
}

type LogPasswordResetSender struct{}

func (LogPasswordResetSender) SendPasswordReset(_ context.Context, contact, resetURL string, expiresAt time.Time) error {
	slog.Warn("development password reset link", "contact", contact, "reset_url", resetURL, "expires_at", expiresAt)
	return nil
}

type DiscardPasswordResetSender struct{}

func (DiscardPasswordResetSender) SendPasswordReset(context.Context, string, string, time.Time) error {
	return nil
}
