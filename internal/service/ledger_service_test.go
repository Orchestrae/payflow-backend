package service

import (
	"context"
	"testing"

	"payflow/internal/domain"
)

// mockLedgerRepo is a simple in-memory ledger repository for testing.
type mockLedgerRepo struct {
	entries []*domain.LedgerEntry
}

func (m *mockLedgerRepo) CreatePair(ctx context.Context, debit, credit *domain.LedgerEntry) error {
	debit.ID = uint(len(m.entries) + 1)
	credit.ID = uint(len(m.entries) + 2)
	m.entries = append(m.entries, debit, credit)
	return nil
}

func (m *mockLedgerRepo) FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.LedgerEntry, int, error) {
	var result []*domain.LedgerEntry
	for _, e := range m.entries {
		if e.BusinessID == businessID {
			result = append(result, e)
		}
	}
	return result, len(result), nil
}

func (m *mockLedgerRepo) GetBalanceByAccount(ctx context.Context, businessID uint, accountType domain.AccountType) (int64, error) {
	var credits, debits int64
	for _, e := range m.entries {
		if e.BusinessID == businessID && e.AccountType == accountType {
			if e.EntryType == domain.EntryCredit {
				credits += e.Amount
			} else {
				debits += e.Amount
			}
		}
	}
	if accountType == domain.AccountWallet {
		return credits - debits, nil
	}
	return debits - credits, nil
}

func (m *mockLedgerRepo) Reconcile(ctx context.Context, businessID uint) (int64, int64, int64, error) {
	var credits, debits int64
	for _, e := range m.entries {
		if e.BusinessID == businessID && e.AccountType == domain.AccountWallet {
			if e.EntryType == domain.EntryCredit {
				credits += e.Amount
			} else {
				debits += e.Amount
			}
		}
	}
	return credits, debits, credits - debits, nil
}

func TestLedgerDeposit_CreatesDebitAndCredit(t *testing.T) {
	repo := &mockLedgerRepo{}
	svc := NewLedgerService(repo)

	err := svc.RecordDeposit(context.Background(), 1, 500000, "DEP-001", "Test deposit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(repo.entries))
	}

	// Same transaction ID
	if repo.entries[0].TransactionID != repo.entries[1].TransactionID {
		t.Error("debit and credit should have same transaction ID")
	}

	// One debit (external), one credit (wallet)
	debit := repo.entries[0]
	credit := repo.entries[1]

	if debit.EntryType != domain.EntryDebit || debit.AccountType != domain.AccountExternal {
		t.Errorf("expected debit on external, got %s on %s", debit.EntryType, debit.AccountType)
	}
	if credit.EntryType != domain.EntryCredit || credit.AccountType != domain.AccountWallet {
		t.Errorf("expected credit on wallet, got %s on %s", credit.EntryType, credit.AccountType)
	}

	// Same amount
	if debit.Amount != 500000 || credit.Amount != 500000 {
		t.Errorf("amounts should match: debit=%d credit=%d", debit.Amount, credit.Amount)
	}
}

func TestLedgerWithdrawal_CreatesDebitAndCredit(t *testing.T) {
	repo := &mockLedgerRepo{}
	svc := NewLedgerService(repo)

	err := svc.RecordWithdrawal(context.Background(), 1, 300000, "WDR-001", "Salary payment")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(repo.entries))
	}

	debit := repo.entries[0]
	credit := repo.entries[1]

	// Withdrawal: debit wallet (money out), credit external
	if debit.AccountType != domain.AccountWallet || debit.EntryType != domain.EntryDebit {
		t.Errorf("expected debit on wallet, got %s on %s", debit.EntryType, debit.AccountType)
	}
	if credit.AccountType != domain.AccountExternal || credit.EntryType != domain.EntryCredit {
		t.Errorf("expected credit on external, got %s on %s", credit.EntryType, credit.AccountType)
	}
}

func TestLedgerReconcile_BalancesMatch(t *testing.T) {
	repo := &mockLedgerRepo{}
	svc := NewLedgerService(repo)

	// Deposit 500K
	svc.RecordDeposit(context.Background(), 1, 500000, "DEP-001", "Deposit")
	// Withdraw 200K
	svc.RecordWithdrawal(context.Background(), 1, 200000, "WDR-001", "Withdrawal")

	// Expected wallet balance: 500K - 200K = 300K
	result, err := svc.Reconcile(context.Background(), 1, 300000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsReconciled {
		t.Errorf("should be reconciled, discrepancy: %d", result.Discrepancy)
	}
	if result.LedgerBalance != 300000 {
		t.Errorf("expected ledger balance 300000, got %d", result.LedgerBalance)
	}
}

func TestLedgerReconcile_DetectsDiscrepancy(t *testing.T) {
	repo := &mockLedgerRepo{}
	svc := NewLedgerService(repo)

	// Deposit 500K in ledger
	svc.RecordDeposit(context.Background(), 1, 500000, "DEP-001", "Deposit")

	// But wallet says 600K (inflated!)
	result, err := svc.Reconcile(context.Background(), 1, 600000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsReconciled {
		t.Error("should NOT be reconciled — wallet is inflated")
	}
	if result.Discrepancy != 100000 {
		t.Errorf("expected discrepancy 100000, got %d", result.Discrepancy)
	}
}

func TestLedgerFee_CreditsRevenue(t *testing.T) {
	repo := &mockLedgerRepo{}
	svc := NewLedgerService(repo)

	err := svc.RecordFee(context.Background(), 1, 5000, "FEE-001", "Transfer fee")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	credit := repo.entries[1]
	if credit.AccountType != domain.AccountRevenue {
		t.Errorf("fee credit should go to revenue account, got %s", credit.AccountType)
	}
}

func TestLedgerGetEntries(t *testing.T) {
	repo := &mockLedgerRepo{}
	svc := NewLedgerService(repo)

	svc.RecordDeposit(context.Background(), 1, 100000, "D1", "Dep 1")
	svc.RecordDeposit(context.Background(), 1, 200000, "D2", "Dep 2")
	svc.RecordDeposit(context.Background(), 2, 300000, "D3", "Dep 3 (different biz)")

	entries, total, err := svc.GetEntries(context.Background(), 1, 1, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Business 1 has 2 deposits × 2 entries = 4
	if total != 4 {
		t.Errorf("expected 4 entries for business 1, got %d", total)
	}
	if len(entries) != 4 {
		t.Errorf("expected 4 entries returned, got %d", len(entries))
	}
}
