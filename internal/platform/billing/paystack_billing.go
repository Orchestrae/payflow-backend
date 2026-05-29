package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PaystackBillingClient handles Paystack subscription/plan API calls.
type PaystackBillingClient struct {
	httpClient *http.Client
	baseURL    string
	secretKey  string
}

// NewPaystackBillingClient creates a new Paystack billing client.
func NewPaystackBillingClient(secretKey, baseURL string) *PaystackBillingClient {
	return &PaystackBillingClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		secretKey:  secretKey,
	}
}

func (c *PaystackBillingClient) makeRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.secretKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("paystack billing request failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

// InitializeTransaction starts a payment for a subscription.
// Returns the authorization URL to redirect the customer to.
func (c *PaystackBillingClient) InitializeTransaction(ctx context.Context, email string, amount int64, reference, planCode, callbackURL string) (string, error) {
	reqBody := map[string]interface{}{
		"email":        email,
		"amount":       amount,
		"reference":    reference,
		"plan":         planCode,
		"callback_url": callbackURL,
	}

	respBytes, err := c.makeRequest(ctx, "POST", "/transaction/initialize", reqBody)
	if err != nil {
		return "", err
	}

	var resp struct {
		Status bool `json:"status"`
		Data   struct {
			AuthorizationURL string `json:"authorization_url"`
			Reference        string `json:"reference"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return "", err
	}
	if !resp.Status {
		return "", fmt.Errorf("failed to initialize transaction")
	}
	return resp.Data.AuthorizationURL, nil
}

// CreatePlan creates a subscription plan on Paystack.
func (c *PaystackBillingClient) CreatePlan(ctx context.Context, name string, amount int64, interval string) (string, error) {
	reqBody := map[string]interface{}{
		"name":     name,
		"amount":   amount,
		"interval": interval,
	}

	respBytes, err := c.makeRequest(ctx, "POST", "/plan", reqBody)
	if err != nil {
		return "", err
	}

	var resp struct {
		Status bool `json:"status"`
		Data   struct {
			PlanCode string `json:"plan_code"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return "", err
	}
	return resp.Data.PlanCode, nil
}
