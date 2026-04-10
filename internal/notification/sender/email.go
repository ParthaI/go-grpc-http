package sender

import (
	"fmt"
	"log/slog"
	"net/smtp"
	"time"
)

// EmailSender sends real emails via SMTP (Mailtrap, Gmail, or any SMTP provider).
type EmailSender struct {
	host     string // SMTP host (e.g., "sandbox.smtp.mailtrap.io")
	port     string // SMTP port (e.g., "587")
	username string // SMTP username (e.g., Mailtrap username)
	password string // SMTP password
	from     string // Sender email (e.g., "notifications@go-grpc-http.dev")
	logger   *slog.Logger
}

func NewEmailSender(host, port, username, password, from string, logger *slog.Logger) *EmailSender {
	return &EmailSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		logger:   logger,
	}
}

func (s *EmailSender) Send(to, subject, body string) {
	// If SMTP is not configured, fall back to mock logging
	if s.host == "" || s.username == "" || s.password == "" {
		s.logger.Info("EMAIL (mock - SMTP not configured)",
			slog.String("to", to),
			slog.String("subject", subject),
			slog.String("body", body),
		)
		return
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"utf-8\"\r\n\r\n%s",
		s.from, to, subject, body)

	// Rate limit: Mailtrap free tier allows ~1 email/sec
	time.Sleep(1500 * time.Millisecond)

	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	err := smtp.SendMail(
		s.host+":"+s.port,
		auth,
		s.from,
		[]string{to},
		[]byte(msg),
	)

	if err != nil {
		s.logger.Error("EMAIL SEND FAILED",
			slog.String("to", to),
			slog.String("subject", subject),
			slog.String("error", err.Error()),
		)
		return
	}

	s.logger.Info("EMAIL SENT",
		slog.String("to", to),
		slog.String("subject", subject),
	)
}
