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
	destinations := make([]BulkPayoutDestination, len(run.Entries))
	for i, entry := range run.Entries {
		destinations[i] = BulkPayoutDestination{
			BankAccount: entry.Employee.BankAccountNumber,
			BankCode:    "058",                         // HARDCODED for MVP. Production needs a Bank Name -> Code mapping!
			Amount:      float64(entry.NetPay) / 100.0, // Convert from cents to main currency unit
			Currency:    "NGN",                         // HARDCODED for MVP.
		}
	}

	koraRequest := BulkPayoutRequest{
		Reference:    fmt.Sprintf("PAYFLOW-RUN-%d-%d", run.ID, time.Now().Unix()),
		Destinations: destinations,
	}

	resp, err := s.client.SendBulkPayout(koraRequest)
	if err != nil {
		return "", fmt.Errorf("%w: %v", domain.ErrPaymentGatewayFailed, err)
	}

	if resp.Status != "success" {
		return "", fmt.Errorf("%w: %s", domain.ErrPaymentGatewayFailed, resp.Message)
	}

	return resp.Data.Reference, nil
}
