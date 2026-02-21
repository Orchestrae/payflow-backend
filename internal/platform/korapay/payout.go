// internal/platform/korapay/payout.go
package korapay

import (
	"context"
	"fmt"
	"payflow/internal/domain"
	"payflow/internal/service"
	"time"
)

type payoutService struct {
	client *Client
}

// NewPayoutService creates a new KoraPay payout service.
func NewPayoutService(client *Client) service.PayoutService {
	return &payoutService{client: client}
}

func (s *payoutService) DisburseBulkPayment(ctx context.Context, run domain.PayrollRun) (string, error) {
	// Convert payroll run entries to Korapay bulk payout items
	payouts := make([]BulkPayoutItem, len(run.Entries))
	for i, entry := range run.Entries {
		if entry.Employee == nil {
			return "", fmt.Errorf("%w: payroll entry %d has no employee data", domain.ErrPaymentGatewayFailed, i)
		}
		payouts[i] = BulkPayoutItem{
			Reference: fmt.Sprintf("PAYFLOW-RUN-%d-ENTRY-%d", run.ID, entry.Employee.ID),
			Amount:    float64(entry.NetPay) / 100.0, // Convert from cents to main currency unit
			Type:      "bank_account",
			Narration: fmt.Sprintf("Payroll payment for %s", entry.Employee.FullName),
			BankAccount: &BulkBankAccountDestination{
				BankCode:      entry.Employee.BankCode,
				AccountNumber: entry.Employee.BankAccountNumber,
			},
			Customer: Customer{
				Name:  entry.Employee.FullName,
				Email: entry.Employee.Email,
			},
		}
	}

	// Create bulk payout request matching actual Korapay API structure
	koraRequest := BulkPayoutRequest{
		BatchReference:    fmt.Sprintf("PAYFLOW-RUN-%d-%d", run.ID, time.Now().Unix()),
		Description:       fmt.Sprintf("Payroll run %d", run.ID),
		MerchantBearsCost: true,
		Currency:          "NGN",
		Payouts:           payouts,
	}

	resp, err := s.client.SendBulkPayout(ctx, koraRequest)
	if err != nil {
		return "", fmt.Errorf("%w: %v", domain.ErrPaymentGatewayFailed, err)
	}

	if !resp.Status {
		return "", fmt.Errorf("%w: %s", domain.ErrPaymentGatewayFailed, resp.Message)
	}

	// Return batch reference from response, or fallback to our batch reference
	if resp.Data != nil && resp.Data.Reference != "" {
		return resp.Data.Reference, nil
	}
	return koraRequest.BatchReference, nil
}
