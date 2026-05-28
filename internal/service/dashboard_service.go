package service

import (
	"context"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

// DashboardSummary contains key metrics for the business dashboard.
type DashboardSummary struct {
	TotalEmployees     int    `json:"total_employees"`
	ActiveEmployees    int    `json:"active_employees"`
	TotalPayrollRuns   int    `json:"total_payroll_runs"`
	PendingApprovals   int    `json:"pending_approvals"`
	LastPayrollPeriod  string `json:"last_payroll_period,omitempty"`
	LastPayrollAmount  int64  `json:"last_payroll_amount"`
	WalletBalance      int64  `json:"wallet_balance"`
	MonthlyPayrollCost int64  `json:"monthly_payroll_cost"`
}

// DashboardService provides dashboard summary data.
type DashboardService interface {
	GetSummary(ctx context.Context, businessID uint) (*DashboardSummary, error)
}

type dashboardService struct {
	employeeRepo repository.EmployeeRepository
	payrollRepo  repository.PayrollRepository
	walletRepo   repository.WalletRepository
}

// NewDashboardService creates a new dashboard service.
func NewDashboardService(
	employeeRepo repository.EmployeeRepository,
	payrollRepo repository.PayrollRepository,
	walletRepo repository.WalletRepository,
) DashboardService {
	return &dashboardService{
		employeeRepo: employeeRepo,
		payrollRepo:  payrollRepo,
		walletRepo:   walletRepo,
	}
}

func (s *dashboardService) GetSummary(ctx context.Context, businessID uint) (*DashboardSummary, error) {
	summary := &DashboardSummary{}

	// Employee counts
	employees, err := s.employeeRepo.FindByBusinessID(ctx, businessID)
	if err == nil {
		summary.TotalEmployees = len(employees)
		for _, emp := range employees {
			if emp.IsActive {
				summary.ActiveEmployees++
			}
		}
	}

	// Payroll stats
	runs, err := s.payrollRepo.FindByBusinessID(ctx, businessID)
	if err == nil {
		summary.TotalPayrollRuns = len(runs)
		for _, run := range runs {
			if run.Status == domain.StatusPendingApproval {
				summary.PendingApprovals++
			}
		}
		// Last completed/processing payroll
		for _, run := range runs {
			if run.Status == domain.StatusCompleted || run.Status == domain.StatusProcessing {
				summary.LastPayrollPeriod = run.Period.Format("2006-01")
				summary.LastPayrollAmount = run.TotalNetPay
				summary.MonthlyPayrollCost = run.TotalCostToCompany
				break // runs are ordered by most recent
			}
		}
	}

	// Wallet balance
	wallet, err := s.walletRepo.FindByBusinessID(ctx, businessID)
	if err == nil && wallet != nil {
		summary.WalletBalance = wallet.Balance
	}

	return summary, nil
}
