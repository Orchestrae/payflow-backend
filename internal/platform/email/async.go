package email

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

const TypeEmailSend = "email:send"

// emailPayload represents an email task in the Asynq queue.
type emailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// AsyncEmailService wraps EmailService with Asynq-backed queueing.
// If Redis/Asynq is available, emails go through the queue with retry.
// If not, falls back to direct sending via goroutine.
type AsyncEmailService struct {
	directSvc *EmailService
	client    *asynq.Client
}

// NewAsyncEmailService creates an async email service.
// Pass nil for client to fall back to direct sending.
func NewAsyncEmailService(directSvc *EmailService, client *asynq.Client) *AsyncEmailService {
	return &AsyncEmailService{
		directSvc: directSvc,
		client:    client,
	}
}

// SendEmail queues an email for delivery via Asynq (or sends directly if no queue).
func (s *AsyncEmailService) SendEmail(ctx context.Context, to, subject, body string) error {
	if s.client == nil {
		// No queue available — send directly (same as before)
		return s.directSvc.SendEmail(ctx, to, subject, body)
	}

	payload, err := json.Marshal(emailPayload{To: to, Subject: subject, Body: body})
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	task := asynq.NewTask(TypeEmailSend, payload,
		asynq.MaxRetry(3),
		asynq.Queue("email"),
	)

	if _, err := s.client.Enqueue(task); err != nil {
		// Queue failed — send directly as fallback
		log.Warn().Err(err).Str("to", to).Msg("Failed to queue email, sending directly")
		return s.directSvc.SendEmail(ctx, to, subject, body)
	}

	log.Debug().Str("to", to).Str("subject", subject).Msg("Email queued via Asynq")
	return nil
}

// HandleEmailTask processes an email task from the Asynq queue.
// Register this as a handler in the Asynq server mux.
func (s *AsyncEmailService) HandleEmailTask(ctx context.Context, t *asynq.Task) error {
	var p emailPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal email payload: %w", err)
	}

	log.Info().Str("to", p.To).Str("subject", p.Subject).Msg("Processing queued email")
	return s.directSvc.SendEmail(ctx, p.To, p.Subject, p.Body)
}
