package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"

	"github.com/matcornic/hermes/v2"
	"github.com/rs/zerolog/log"

	"payflow/internal/config"
)

// EmailService sends beautifully designed transactional emails via SMTP.
type EmailService struct {
	smtpHost     string
	smtpPort     string
	smtpUser     string
	smtpPassword string
	from         string
	appURL       string
	Hermes       hermes.Hermes
}

// NewEmailService creates a configurable email service with Hermes templates.
func NewEmailService(cfg *config.Config) *EmailService {
	h := hermes.Hermes{
		Product: hermes.Product{
			Name:      "PayFlow",
			Link:      cfg.AppURL,
			Logo:      "", // Add logo URL when available
			Copyright: "PayFlow - Automated Payroll Platform",
		},
	}

	return &EmailService{
		smtpHost:     cfg.SMTPHost,
		smtpPort:     cfg.SMTPPort,
		smtpUser:     cfg.SMTPUser,
		smtpPassword: cfg.SMTPPassword,
		from:         cfg.SMTPFrom,
		appURL:       cfg.AppURL,
		Hermes:       h,
	}
}

// SendEmail sends a plain text email (implements NotificationService interface).
func (s *EmailService) SendEmail(ctx context.Context, to, subject, body string) error {
	// Wrap plain text in a simple Hermes email for consistent branding
	email := hermes.Email{
		Body: hermes.Body{
			FreeMarkdown: hermes.Markdown(body),
		},
	}
	return s.SendHermesEmail(ctx, to, subject, email)
}

// SendHermesEmail sends a beautifully designed HTML email using Hermes.
func (s *EmailService) SendHermesEmail(ctx context.Context, to, subject string, email hermes.Email) error {
	htmlBody, err := s.Hermes.GenerateHTML(email)
	if err != nil {
		return fmt.Errorf("failed to generate HTML email: %w", err)
	}

	textBody, err := s.Hermes.GeneratePlainText(email)
	if err != nil {
		return fmt.Errorf("failed to generate plain text email: %w", err)
	}

	return s.sendMIME(to, subject, htmlBody, textBody)
}

// sendMIME sends a multipart MIME email with both HTML and plain text.
func (s *EmailService) sendMIME(to, subject, htmlBody, textBody string) error {
	addr := s.smtpHost + ":" + s.smtpPort
	boundary := "PayFlowBoundary"

	headers := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: multipart/alternative; boundary=%s\r\n\r\n",
		s.from, to, subject, boundary)

	body := headers +
		"--" + boundary + "\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n\r\n" +
		textBody + "\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n\r\n" +
		htmlBody + "\r\n" +
		"--" + boundary + "--\r\n"

	var auth smtp.Auth
	if s.smtpUser != "" {
		auth = smtp.PlainAuth("", s.smtpUser, s.smtpPassword, s.smtpHost)
	}

	log.Info().Str("to", to).Str("subject", subject).Msg("Sending email")

	// Try with STARTTLS first (production SMTP), fall back to plain (MailHog)
	err := s.sendWithTLS(addr, auth, to, []byte(body))
	if err != nil {
		// Fallback to plain SMTP (for MailHog / local dev)
		err = smtp.SendMail(addr, auth, s.from, []string{to}, []byte(body))
	}

	if err != nil {
		log.Error().Err(err).Str("to", to).Msg("Failed to send email")
		return err
	}

	return nil
}

// sendWithTLS attempts to send email with STARTTLS.
func (s *EmailService) sendWithTLS(addr string, auth smtp.Auth, to string, msg []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()

	// Try STARTTLS
	tlsConfig := &tls.Config{ServerName: s.smtpHost}
	if err := c.StartTLS(tlsConfig); err != nil {
		return err // Caller will fallback to plain SMTP
	}

	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return err
		}
	}

	if err := c.Mail(s.from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return c.Quit()
}

// GetAppURL returns the configured app URL for building email links.
func (s *EmailService) GetAppURL() string {
	return s.appURL
}
