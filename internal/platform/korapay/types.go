// internal/platform/korapay/types.go
package korapay

// This file contains structs that map directly to the KoraPay API.
// Based on: https://docs.korapay.com/reference/post_payout-bulk

type AuthRequest struct {
	SecretKey string `json:"secret_key"`
}

type AuthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		SecretKey string `json:"secret_key"` // They call it secret_key, but it's the token
	} `json:"data"`
}

type BulkPayoutDestination struct {
	BankAccount string  `json:"bank_account"`
	BankCode    string  `json:"bank_code"` // We'll need a way to map bank names to codes
	Amount      float64 `json:"amount"`    // Kora uses float for amount, we must convert from int64
	Currency    string  `json:"currency"`  // e.g., "NGN"
}

type BulkPayoutRequest struct {
	Reference    string                  `json:"reference"`
	Destinations []BulkPayoutDestination `json:"destinations"`
}

type BulkPayoutResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Reference string `json:"reference"`
	} `json:"data"`
}
