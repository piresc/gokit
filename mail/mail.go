// Package mail provides email sending with SMTP, log, and in-memory implementations.
package mail

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"

	"github.com/piresc/gokit/config"
)

// Sender sends an email.
type Sender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// NewSender returns an SMTPSender when cfg.Provider == "smtp" and cfg.SMTPHost is set,
// otherwise a LogSender.
func NewSender(cfg config.MailConfig) Sender {
	if cfg.Provider == "smtp" && cfg.SMTPHost != "" {
		return &SMTPSender{cfg: cfg}
	}
	return &LogSender{}
}

// SMTPSender sends email via SMTP.
type SMTPSender struct {
	cfg config.MailConfig
}

// Send delivers the message via SMTP.
func (s *SMTPSender) Send(_ context.Context, to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.SMTPHost)
	}
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		s.cfg.From, to, subject, body))
	return smtp.SendMail(addr, auth, s.cfg.From, []string{to}, msg)
}

// LogSender prints emails to stdout (or an slog.Logger if set).
type LogSender struct {
	Log *slog.Logger
}

// Send logs the email instead of delivering it.
func (s *LogSender) Send(_ context.Context, to, subject, body string) error {
	if s.Log != nil {
		s.Log.Info("mail (log sender)",
			slog.String("to", to),
			slog.String("subject", subject),
			slog.String("body", body),
		)
	} else {
		fmt.Printf("[mail] to=%s subject=%s body=%s\n", to, subject, body)
	}
	return nil
}

// Message holds a captured email for testing.
type Message struct {
	To      string
	Subject string
	Body    string
}

// InMemorySender captures emails in memory for testing.
type InMemorySender struct {
	Messages []Message
}

// Send appends the message to the internal slice.
func (s *InMemorySender) Send(_ context.Context, to, subject, body string) error {
	s.Messages = append(s.Messages, Message{To: to, Subject: subject, Body: body})
	return nil
}
