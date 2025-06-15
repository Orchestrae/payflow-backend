// internal/platform/sendgrid/email.go
package sendgrid

import (
	"context"
	"fmt"
	"net/smtp"
	"payflow/internal/service"

	"github.com/rs/zerolog/log"
)

type mailhogService struct {
	smtpHost string
	smtpPort string
	from     string
}

// NewMailhogService creates a notification service that sends email via MailHog.
func NewMailhogService() service.NotificationService {
	// These values would come from config in a real app
	return &mailhogService{
		smtpHost: "localhost",
		smtpPort: "1025",
		from:     "no-reply@payflow.com",
	}
}

func (s *mailhogService) SendEmail(ctx context.Context, to, subject, body string) error {
	addr := s.smtpHost + ":" + s.smtpPort
	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", to, s.from, subject, body))

	// For MailHog, we use PlainAuth with empty values.
	auth := smtp.PlainAuth("", s.from, "", s.smtpHost)

	log.Info().Str("to", to).Str("subject", subject).Msg("Sending email via MailHog")

	err := smtp.SendMail(addr, auth, s.from, []string{to}, msg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send email")
		return err
	}
	return nil
}
