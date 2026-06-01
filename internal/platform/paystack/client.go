package paystack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles communication with the Paystack API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	secretKey  string
}

// NewClient creates a new Paystack client.
func NewClient(secretKey, baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		secretKey:  secretKey,
	}
}

// makeRequest is a helper to make HTTP requests with Bearer token authentication.
func (c *Client) makeRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.secretKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("paystack request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("paystack request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return bodyBytes, nil
}

// ResolveBVN verifies a BVN via Paystack.
// Endpoint: GET /bank/resolve_bvn/{bvn} (costs NGN 10 per call)
func (c *Client) ResolveBVN(ctx context.Context, bvn string) (*ResolveBVNResponse, error) {
	endpoint := fmt.Sprintf("/bank/resolve_bvn/%s", bvn)
	bodyBytes, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("paystack BVN resolve failed: %w", err)
	}

	var response ResolveBVNResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode paystack BVN response: %w", err)
	}

	return &response, nil
}

// CreateTransferRecipient creates a transfer recipient.
// Endpoint: POST /transferrecipient
func (c *Client) CreateTransferRecipient(ctx context.Context, req CreateRecipientRequest) (*CreateRecipientResponse, error) {
	bodyBytes, err := c.makeRequest(ctx, "POST", "/transferrecipient", req)
	if err != nil {
		return nil, fmt.Errorf("paystack create recipient failed: %w", err)
	}

	var response CreateRecipientResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode paystack recipient response: %w", err)
	}

	return &response, nil
}

// InitiateTransfer initiates a single transfer.
// Endpoint: POST /transfer
func (c *Client) InitiateTransfer(ctx context.Context, req TransferRequest) (*TransferResponse, error) {
	bodyBytes, err := c.makeRequest(ctx, "POST", "/transfer", req)
	if err != nil {
		return nil, fmt.Errorf("paystack transfer failed: %w", err)
	}

	var response TransferResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode paystack transfer response: %w", err)
	}

	return &response, nil
}

// InitiateBulkTransfer initiates a bulk transfer (max 100 per batch).
// Endpoint: POST /transfer/bulk
func (c *Client) InitiateBulkTransfer(ctx context.Context, req BulkTransferRequest) (*BulkTransferResponse, error) {
	bodyBytes, err := c.makeRequest(ctx, "POST", "/transfer/bulk", req)
	if err != nil {
		return nil, fmt.Errorf("paystack bulk transfer failed: %w", err)
	}

	var response BulkTransferResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode paystack bulk transfer response: %w", err)
	}

	return &response, nil
}

// GetBalance fetches the Paystack account balance.
// Endpoint: GET /balance
func (c *Client) GetBalance(ctx context.Context) (*BalanceResponse, error) {
	bodyBytes, err := c.makeRequest(ctx, "GET", "/balance", nil)
	if err != nil {
		return nil, fmt.Errorf("paystack get balance failed: %w", err)
	}

	var response BalanceResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode paystack balance response: %w", err)
	}

	return &response, nil
}

// ResolveAccountNumber resolves a bank account number to get the account name.
// Endpoint: GET /bank/resolve?account_number=...&bank_code=...
func (c *Client) ResolveAccountNumber(ctx context.Context, accountNumber, bankCode string) (*ResolveAccountResponse, error) {
	endpoint := fmt.Sprintf("/bank/resolve?account_number=%s&bank_code=%s", accountNumber, bankCode)
	bodyBytes, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("paystack resolve account failed: %w", err)
	}

	var response ResolveAccountResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode paystack resolve response: %w", err)
	}

	return &response, nil
}
