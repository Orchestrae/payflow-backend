// tests/integration/payflow_test.go
package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"

	"payflow/internal/domain"
	"payflow/internal/platform/database"
	postgresRepo "payflow/internal/repository/postgres"
	"payflow/internal/service"
	"payflow/pkg/utils"
)

// testDB holds the shared database connection for integration tests.
var testDB *gorm.DB

// setupPostgresContainer starts a real Postgres container and returns a GORM DB connected to it.
func setupPostgresContainer(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("payflow_test"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	db, err := database.NewPostgresDB(connStr)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// Run SQL migration files in order
	migrationsDir := findMigrationsDir()
	if err := runMigrations(db, migrationsDir); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		_ = pgContainer.Terminate(ctx)
	}

	return db, cleanup
}

// findMigrationsDir locates the migrations directory relative to the test file.
func findMigrationsDir() string {
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	return filepath.Join(projectRoot, "migrations")
}

// runMigrations executes all *.up.sql files in order.
func runMigrations(db *gorm.DB, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read migrations dir: %w", err)
	}

	var upFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".up.sql") {
			upFiles = append(upFiles, e.Name())
		}
	}
	sort.Strings(upFiles) // Ensures numeric order: 000001, 000002, ...

	for _, f := range upFiles {
		content, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", f, err)
		}
		if err := db.Exec(string(content)).Error; err != nil {
			// Skip non-critical errors (type already exists, etc.)
			errStr := err.Error()
			if strings.Contains(errStr, "already exists") || strings.Contains(errStr, "duplicate") {
				continue
			}
			return fmt.Errorf("failed to execute migration %s: %w", f, err)
		}
	}
	return nil
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestRegistrationFlow(t *testing.T) {
	db, cleanup := setupPostgresContainer(t)
	defer cleanup()

	ctx := context.Background()

	userRepo := postgresRepo.NewUserRepository(db)
	businessRepo := postgresRepo.NewBusinessRepository(db)

	// Step 1: Create a business directly via repository
	business := &domain.Business{
		Name:     "Test Corp",
		Currency: "NGN",
	}
	if err := businessRepo.Create(ctx, business); err != nil {
		t.Fatalf("failed to create business: %v", err)
	}
	if business.ID == 0 {
		t.Fatal("expected business to have an ID after creation")
	}

	// Step 2: Create an admin user linked to the business
	hashedPassword, err := utils.HashPassword("SecureP@ss123")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := &domain.User{
		BusinessID:   business.ID,
		Email:        "admin@testcorp.com",
		PasswordHash: hashedPassword,
		Role:         domain.RoleAdmin,
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected user to have an ID after creation")
	}

	// Step 3: Verify user can be found
	foundUser, err := userRepo.FindByEmail(ctx, "admin@testcorp.com")
	if err != nil {
		t.Fatalf("failed to find user by email: %v", err)
	}
	if foundUser.BusinessID != business.ID {
		t.Errorf("expected business_id %d, got %d", business.ID, foundUser.BusinessID)
	}
	if foundUser.Role != domain.RoleAdmin {
		t.Errorf("expected role admin, got %s", foundUser.Role)
	}

	// Step 4: Verify business can be found
	foundBusiness, err := businessRepo.FindByID(ctx, business.ID)
	if err != nil {
		t.Fatalf("failed to find business by ID: %v", err)
	}
	if foundBusiness.Name != "Test Corp" {
		t.Errorf("expected business name 'Test Corp', got %q", foundBusiness.Name)
	}
}

func TestEmployeeLifecycle(t *testing.T) {
	db, cleanup := setupPostgresContainer(t)
	defer cleanup()

	ctx := context.Background()

	businessRepo := postgresRepo.NewBusinessRepository(db)
	cadreRepo := postgresRepo.NewCadreRepository(db)
	employeeRepo := postgresRepo.NewEmployeeRepository(db)

	// Setup: create business
	business := &domain.Business{Name: "Lifecycle Corp", Currency: "NGN"}
	if err := businessRepo.Create(ctx, business); err != nil {
		t.Fatalf("failed to create business: %v", err)
	}

	// Setup: create cadre with earning components
	cadre := &domain.Cadre{
		BusinessID: business.ID,
		Name:       "Senior Engineer",
		EarningComponents: []domain.EarningComponent{
			{Name: "Basic Salary", Amount: 50000000, ComponentType: domain.ComponentBasic},
			{Name: "Housing", Amount: 20000000, ComponentType: domain.ComponentHousing},
			{Name: "Transport", Amount: 10000000, ComponentType: domain.ComponentTransport},
		},
	}
	if err := cadreRepo.Create(ctx, cadre); err != nil {
		t.Fatalf("failed to create cadre: %v", err)
	}

	// Step 1: Create employee using service
	empSvc := service.NewEmployeeService(employeeRepo, cadreRepo)
	emp := &domain.Employee{
		BusinessID:        business.ID,
		CadreID:           cadre.ID,
		FullName:          "John Doe",
		Email:             "john@lifecyclecorp.com",
		BankName:          "Test Bank",
		BankCode:          "058",
		BankAccountNumber: "1234567890",
		IsActive:          true,
	}
	createdEmp, err := empSvc.CreateEmployee(ctx, emp)
	if err != nil {
		t.Fatalf("failed to create employee: %v", err)
	}
	if createdEmp.ID == 0 {
		t.Fatal("expected employee to have an ID")
	}

	// Step 2: Update employee
	createdEmp.FullName = "John Updated Doe"
	updatedEmp, err := empSvc.UpdateEmployee(ctx, createdEmp)
	if err != nil {
		t.Fatalf("failed to update employee: %v", err)
	}
	if updatedEmp.FullName != "John Updated Doe" {
		t.Errorf("expected updated name, got %q", updatedEmp.FullName)
	}

	// Step 3: Deactivate employee
	if err := empSvc.DeactivateEmployee(ctx, createdEmp.ID, business.ID); err != nil {
		t.Fatalf("failed to deactivate employee: %v", err)
	}

	// Step 4: Verify employee is deactivated
	deactivated, err := empSvc.GetByID(ctx, createdEmp.ID, business.ID)
	if err != nil {
		t.Fatalf("failed to get deactivated employee: %v", err)
	}
	if deactivated.IsActive {
		t.Error("expected employee to be deactivated")
	}
}

func TestPayrollCreation(t *testing.T) {
	db, cleanup := setupPostgresContainer(t)
	defer cleanup()

	ctx := context.Background()

	businessRepo := postgresRepo.NewBusinessRepository(db)
	cadreRepo := postgresRepo.NewCadreRepository(db)
	employeeRepo := postgresRepo.NewEmployeeRepository(db)
	payrollRepo := postgresRepo.NewPayrollRepository(db)

	// Setup: create business with PAYE enabled
	business := &domain.Business{
		Name:        "Payroll Corp",
		Currency:    "NGN",
		PAYEEnabled: true,
	}
	if err := businessRepo.Create(ctx, business); err != nil {
		t.Fatalf("failed to create business: %v", err)
	}

	// Setup: create cadre
	cadre := &domain.Cadre{
		BusinessID: business.ID,
		Name:       "Staff",
		EarningComponents: []domain.EarningComponent{
			{Name: "Basic Salary", Amount: 30000000, ComponentType: domain.ComponentBasic},
			{Name: "Housing", Amount: 12000000, ComponentType: domain.ComponentHousing},
			{Name: "Transport", Amount: 8000000, ComponentType: domain.ComponentTransport},
		},
	}
	if err := cadreRepo.Create(ctx, cadre); err != nil {
		t.Fatalf("failed to create cadre: %v", err)
	}

	// Setup: create employees
	for i := 0; i < 3; i++ {
		emp := &domain.Employee{
			BusinessID:        business.ID,
			CadreID:           cadre.ID,
			FullName:          fmt.Sprintf("Employee %d", i+1),
			Email:             fmt.Sprintf("emp%d@payrollcorp.com", i+1),
			BankName:          "Test Bank",
			BankCode:          "058",
			BankAccountNumber: fmt.Sprintf("123456789%d", i),
			IsActive:          true,
		}
		if err := employeeRepo.Create(ctx, emp); err != nil {
			t.Fatalf("failed to create employee %d: %v", i+1, err)
		}
	}

	// Step 1: Create a payroll run directly
	period := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	payrollRun := &domain.PayrollRun{
		BusinessID:    business.ID,
		Period:        period,
		Status:        domain.StatusDraft,
		TotalGrossPay: 150000000, // 3 employees * 50M gross each
		TotalNetPay:   120000000,
		TotalDeductions: 30000000,
		Entries: []domain.PayrollRunEntry{
			{
				EmployeeID:      1,
				GrossPay:        50000000,
				TotalDeductions: 10000000,
				NetPay:          40000000,
			},
		},
	}
	if err := payrollRepo.Create(ctx, payrollRun); err != nil {
		t.Fatalf("failed to create payroll run: %v", err)
	}
	if payrollRun.ID == 0 {
		t.Fatal("expected payroll run to have an ID")
	}
	if payrollRun.Status != domain.StatusDraft {
		t.Errorf("expected status draft, got %s", payrollRun.Status)
	}
	if payrollRun.TotalGrossPay != 150000000 {
		t.Errorf("expected total gross pay 150000000, got %d", payrollRun.TotalGrossPay)
	}

	// Step 2: Verify payroll can be retrieved
	runs, err := payrollRepo.FindByBusinessID(ctx, business.ID)
	if err != nil {
		t.Fatalf("failed to list payroll runs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 payroll run, got %d", len(runs))
	}
}

func TestDepositAndBalance(t *testing.T) {
	db, cleanup := setupPostgresContainer(t)
	defer cleanup()

	ctx := context.Background()

	businessRepo := postgresRepo.NewBusinessRepository(db)
	walletRepo := postgresRepo.NewWalletRepository(db)
	walletTxRepo := postgresRepo.NewWalletTransactionRepository(db)

	// Setup: create business
	business := &domain.Business{Name: "Wallet Corp", Currency: "NGN"}
	if err := businessRepo.Create(ctx, business); err != nil {
		t.Fatalf("failed to create business: %v", err)
	}

	// Setup: create wallet
	wallet := &domain.BusinessWallet{
		BusinessID:              business.ID,
		Balance:                 0,
		Currency:                "NGN",
		VirtualAccountNumber:    "1234567890",
		VirtualAccountBankCode:  "035",
		VirtualAccountBankName:  "Wema Bank",
		VirtualAccountReference: fmt.Sprintf("ref-%d", business.ID),
		VirtualAccountStatus:    "active",
		Provider:                "korapay",
	}
	if err := walletRepo.Create(ctx, wallet); err != nil {
		t.Fatalf("failed to create wallet: %v", err)
	}

	// Step 1: Record a deposit (increment balance)
	depositAmount := int64(5000000) // 50,000 NGN in kobo
	updatedWallet, err := walletRepo.IncrementBalance(ctx, business.ID, depositAmount)
	if err != nil {
		t.Fatalf("failed to increment balance: %v", err)
	}
	if updatedWallet.Balance != depositAmount {
		t.Errorf("expected balance %d, got %d", depositAmount, updatedWallet.Balance)
	}

	// Step 2: Record a wallet transaction
	tx := &domain.WalletTransaction{
		BusinessID:      business.ID,
		TransactionType: domain.WalletTransactionDeposit,
		Amount:          depositAmount,
		BalanceBefore:   0,
		BalanceAfter:    depositAmount,
		Currency:        "NGN",
		Reference:       "DEP-001",
		Description:     "Test deposit",
		Status:          "completed",
	}
	if err := walletTxRepo.Create(ctx, tx); err != nil {
		t.Fatalf("failed to create wallet transaction: %v", err)
	}

	// Step 3: Verify transaction history
	transactions, total, err := walletTxRepo.FindByBusinessID(ctx, business.ID, 1, 10)
	if err != nil {
		t.Fatalf("failed to list transactions: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 transaction, got %d", total)
	}
	if len(transactions) != 1 {
		t.Fatalf("expected 1 transaction in slice, got %d", len(transactions))
	}
	if transactions[0].Amount != depositAmount {
		t.Errorf("expected amount %d, got %d", depositAmount, transactions[0].Amount)
	}
}

func TestLedgerConsistency(t *testing.T) {
	db, cleanup := setupPostgresContainer(t)
	defer cleanup()

	ctx := context.Background()

	businessRepo := postgresRepo.NewBusinessRepository(db)
	ledgerRepo := postgresRepo.NewLedgerRepository(db)

	// Setup: create business
	business := &domain.Business{Name: "Ledger Corp", Currency: "NGN"}
	if err := businessRepo.Create(ctx, business); err != nil {
		t.Fatalf("failed to create business: %v", err)
	}

	// Use ledger service for double-entry operations
	ledgerSvc := service.NewLedgerService(ledgerRepo)

	// Step 1: Record a deposit (credit wallet, debit external)
	depositAmount := int64(10000000) // 100,000 NGN
	if err := ledgerSvc.RecordDeposit(ctx, business.ID, depositAmount, "DEP-001", "Client deposit"); err != nil {
		t.Fatalf("failed to record deposit: %v", err)
	}

	// Step 2: Record a withdrawal (debit wallet, credit external)
	withdrawalAmount := int64(3000000) // 30,000 NGN
	if err := ledgerSvc.RecordWithdrawal(ctx, business.ID, withdrawalAmount, "WD-001", "Salary payment"); err != nil {
		t.Fatalf("failed to record withdrawal: %v", err)
	}

	// Step 3: Check ledger entries
	entries, total, err := ledgerSvc.GetEntries(ctx, business.ID, 1, 100)
	if err != nil {
		t.Fatalf("failed to get ledger entries: %v", err)
	}
	// 2 entries per operation (debit + credit) * 2 operations = 4 entries
	if total != 4 {
		t.Errorf("expected 4 ledger entries, got %d", total)
	}
	if len(entries) != 4 {
		t.Errorf("expected 4 entries in slice, got %d", len(entries))
	}

	// Step 4: Reconcile -- sum of credits must equal sum of debits
	var totalCredits, totalDebits int64
	for _, entry := range entries {
		if entry.EntryType == domain.EntryCredit {
			totalCredits += entry.Amount
		} else if entry.EntryType == domain.EntryDebit {
			totalDebits += entry.Amount
		}
	}
	if totalCredits != totalDebits {
		t.Errorf("ledger not balanced: credits=%d debits=%d", totalCredits, totalDebits)
	}

	// Step 5: Credits and debits should each total deposit+withdrawal amounts
	expectedTotal := depositAmount + withdrawalAmount
	if totalCredits != expectedTotal {
		t.Errorf("expected total credits %d, got %d", expectedTotal, totalCredits)
	}
	if totalDebits != expectedTotal {
		t.Errorf("expected total debits %d, got %d", expectedTotal, totalDebits)
	}

	// Step 6: Check wallet account balance via repo
	walletBalance, err := ledgerRepo.GetBalanceByAccount(ctx, business.ID, domain.AccountWallet)
	if err != nil {
		t.Fatalf("failed to get wallet balance from ledger: %v", err)
	}
	// Wallet balance = credits to wallet - debits from wallet = deposit - withdrawal
	expectedWalletBalance := depositAmount - withdrawalAmount
	if walletBalance != expectedWalletBalance {
		t.Errorf("expected wallet ledger balance %d, got %d", expectedWalletBalance, walletBalance)
	}
}

