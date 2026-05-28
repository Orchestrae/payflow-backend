package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// TermiiService sends SMS via Termii API.
type TermiiService struct {
	apiKey   string
	senderID string
	client   *http.Client
}

// NewTermiiService creates a new Termii SMS service.
func NewTermiiService(apiKey, senderID string) *TermiiService {
	return &TermiiService{
		apiKey:   apiKey,
		senderID: senderID,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

type termiiRequest struct {
	To       string `json:"to"`
	From     string `json:"from"`
	SMS      string `json:"sms"`
	Type     string `json:"type"`
	Channel  string `json:"channel"`
	APIKey   string `json:"api_key"`
}

type termiiResponse struct {
	MessageID string `json:"message_id"`
	Message   string `json:"message"`
	Balance   float64 `json:"balance"`
}

// SendSMS sends an SMS via Termii.
func (s *TermiiService) SendSMS(ctx context.Context, phone, message string) error {
	if s.apiKey == "" {
		log.Warn().Msg("SMS not sent: Termii API key not configured")
		return nil
	}

	reqBody := termiiRequest{
		To:      phone,
		From:    s.senderID,
		SMS:     message,
		Type:    "plain",
		Channel: "generic",
		APIKey:  s.apiKey,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal SMS request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.ng.termii.com/api/sms/send", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create SMS request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("SMS request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("termii SMS failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.Info().Str("phone", phone).Msg("SMS sent via Termii")
	return nil
}
