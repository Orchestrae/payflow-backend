// internal/service/payroll_service.go
package service

import (
	"context"
	"fmt"
	"payflow/internal/domain"
	"payflow/internal/repository"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type payrollService struct {
	payrollRepo     repository.PayrollRepository
	employeeRepo    repository.EmployeeRepository
	cadreRepo       repository.CadreRepository
	businessRepo    repository.BusinessRepository
	transferRepo    repository.TransferRepository
	loanRepo        repository.LoanRepository
	txer            repository.Transactioner
	payoutSvc       PayoutService
	notificationSvc NotificationService
	scheduler       domain.Scheduler
	userRepo        repository.UserRepository
	transferSvc     TransferService
}

// NewPayrollService creates a new payroll service instance.
func NewPayrollService(
	payrollRepo repository.PayrollRepository,
	employeeRepo repository.EmployeeRepository,
	cadreRepo repository.CadreRepository,
	businessRepo repository.BusinessRepository,
	transferRepo repository.TransferRepository,
	txer repository.Transactioner,
	payoutSvc PayoutService,
	notificationSvc NotificationService,
	userRepo repository.UserRepository,
	scheduler domain.Scheduler,
	transferSvc TransferService,
	loanRepo repository.LoanRepository,
) PayrollService {
	return &payrollService{
		payrollRepo:     payrollRepo,
		employeeRepo:    employeeRepo,
		cadreRepo:       cadreRepo,
		businessRepo:    businessRepo,
		transferRepo:    transferRepo,
		loanRepo:        loanRepo,
		txer:            txer,
		payoutSvc:       payoutSvc,
		notificationSvc: notificationSvc,
		userRepo:        userRepo,
		scheduler:       scheduler,
		transferSvc:     transferSvc,
	}
}

// ListByBusinessID retrieves all payroll runs for a business.
func (s *payrollService) ListByBusinessID(ctx context.Context, businessID uint) ([]*domain.PayrollRun, error) {
	return s.payrollRepo.FindByBusinessID(ctx, businessID)
}

// GetByID retrieves a specific payroll run by ID, ensuring it belongs to the specified business.
func (s *payrollService) GetByID(ctx context.Context, runID, businessID uint) (*domain.PayrollRun, error) {
	return s.payrollRepo.FindByID(ctx, runID, businessID)
}

// SubmitForApproval submits a draft payroll run for approval.
func (s *payrollService) SubmitForApproval(ctx context.Context, runID, userID uint) (*domain.PayrollRun, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	run, err := s.payrollRepo.FindByID(ctx, runID, user.BusinessID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payroll run: %w", err)
	}

	if run.Status != domain.StatusDraft {
		return nil, fmt.Errorf("%w: cannot submit non-draft run", domain.ErrValidationFailed)
	}

	business, err := s.businessRepo.FindByID(ctx, user.BusinessID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch business: %w", err)
	}

	// Verification gate: warn about unverified bank accounts (block on paid plans)
	if run.Entries != nil && len(run.Entries) > 0 {
		// Batch-load employees if not already loaded
		var empIDs []uint
		for i := range run.Entries {
			if run.Entries[i].Employee == nil {
				empIDs = append(empIDs, run.Entries[i].EmployeeID)
			}
		}
		if len(empIDs) > 0 {
			emps, _ := s.employeeRepo.FindByIDs(ctx, empIDs, user.BusinessID)
			empMap := make(map[uint]*domain.Employee, len(emps))
			for _, e := range emps {
				empMap[e.ID] = e
			}
			for i := range run.Entries {
				if run.Entries[i].Employee == nil {
					run.Entries[i].Employee = empMap[run.Entries[i].EmployeeID]
				}
			}
		}

		var unverifiedNames []string
		for _, entry := range run.Entries {
			if entry.Employee != nil && !entry.Employee.BankAccountVerified {
				unverifiedNames = append(unverifiedNames, entry.Employee.FullName)
			}
		}
		if len(unverifiedNames) > 0 {
			if domain.PlanTier(business.SubscriptionTier) != domain.PlanFree {
				return nil, fmt.Errorf("%w: %d employees have unverified bank accounts: %s",
					domain.ErrValidationFailed, len(unverifiedNames), strings.Join(unverifiedNames, ", "))
			}
			log.Warn().Int("count", len(unverifiedNames)).Msg("Unverified bank accounts in payroll — allowed on Free plan")
		}
	}

	// If business doesn't require approval, auto-approve
	if !business.PayrollRequiresApproval {
		run.Status = domain.StatusApproved
		log.Info().Uint("run_id", runID).Uint("business_id", user.BusinessID).Msg("Auto-approving payroll run")

		if business.PayrollAutoProcess {
			return s.processPayrollRunInstantly(ctx, run, user.BusinessID)
		}

		if err := s.scheduler.SchedulePayout(*run); err != nil {
			return nil, fmt.Errorf("failed to schedule payout: %w", err)
		}
	} else {
		run.Status = domain.StatusPendingApproval
	}

	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to update payroll run status: %w", err)
	}

	return run, nil
}

// ApprovePayrollRun approves a pending payroll run.
func (s *payrollService) ApprovePayrollRun(ctx context.Context, runID, approverID uint) (*domain.PayrollRun, error) {
	approver, err := s.userRepo.FindByID(ctx, approverID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch approver: %w", err)
	}

	run, err := s.payrollRepo.FindByID(ctx, runID, approver.BusinessID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payroll run: %w", err)
	}

	if run.Status != domain.StatusPendingApproval {
		return nil, domain.ErrValidationFailed
	}

	business, err := s.businessRepo.FindByID(ctx, approver.BusinessID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch business: %w", err)
	}

	run.Status = domain.StatusApproved
	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return nil, err
	}

	// Send approval confirmation to operators
	if s.notificationSvc != nil {
		go func() {
			operators, _ := s.userRepo.FindByBusinessID(context.Background(), approver.BusinessID)
			for _, op := range operators {
				if op.Role == domain.RoleOperator {
					s.notificationSvc.SendEmail(context.Background(), op.Email,
						"Payroll Approved: "+run.Period.Format("January 2006"),
						fmt.Sprintf("The payroll run for %s has been approved and is scheduled for processing.", run.Period.Format("January 2006")))
				}
			}
		}()
	}

	if business.PayrollAutoProcess {
		log.Info().Uint("run_id", runID).Msg("Auto-processing payroll run")
		return s.processPayrollRunInstantly(ctx, run, approver.BusinessID)
	}

	if err := s.scheduler.SchedulePayout(*run); err != nil {
		log.Error().Err(err).Uint("runID", runID).Msg("CRITICAL: Failed to schedule payout for approved run")
		run.Status = domain.StatusPendingApproval
		_ = s.payrollRepo.Update(ctx, run)
		return nil, fmt.Errorf("could not schedule payout job: %w", err)
	}

	return run, nil
}

// RejectPayrollRun rejects a pending payroll run with a reason.
func (s *payrollService) RejectPayrollRun(ctx context.Context, runID, rejecterID uint, reason string) (*domain.PayrollRun, error) {
	log.Ctx(ctx).Info().Uint("run_id", runID).Uint("user_id", rejecterID).Msg("Attempting to reject payroll run")

	if reason == "" {
		return nil, fmt.Errorf("%w: rejection requires a reason", domain.ErrValidationFailed)
	}

	rejecter, err := s.userRepo.FindByID(ctx, rejecterID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not find rejecting user")
		return nil, domain.ErrInternalServer
	}

	run, err := s.payrollRepo.FindByID(ctx, runID, rejecter.BusinessID)
	if err != nil {
		return nil, err
	}

	if run.Status != domain.StatusPendingApproval {
		return nil, fmt.Errorf("%w: cannot reject a payroll run with status '%s'", domain.ErrValidationFailed, run.Status)
	}

	if rejecter.BusinessID != run.BusinessID {
		return nil, domain.ErrForbidden
	}

	run.Status = domain.StatusRejected
	run.RejectionReason = reason
	run.UpdatedAt = time.Now()

	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return nil, err
	}

	// Notify operators
	users, err := s.userRepo.FindByBusinessID(ctx, run.BusinessID)
	if err == nil {
		for _, user := range users {
			if user.Role == domain.RoleOperator {
				go s.notificationSvc.SendEmail(context.Background(), user.Email,
					fmt.Sprintf("Action Required: Payroll for %s was Rejected", run.Period.Format("January 2006")),
					fmt.Sprintf("Hello %s,\n\nThe payroll run for %s has been rejected.\n\nReason: %s\n\nPlease review and resubmit.",
						user.Email, run.Period.Format("January 2006"), reason))
			}
		}
	}

	return run, nil
}

// AmendPayrollRun recalculates a draft payroll run with updated adjustments.
func (s *payrollService) AmendPayrollRun(ctx context.Context, runID, businessID uint, adjustments map[uint][]EmployeeAdjustment) (*domain.PayrollRun, error) {
	run, err := s.payrollRepo.FindByID(ctx, runID, businessID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payroll run: %w", err)
	}

	if run.Status != domain.StatusDraft {
		return nil, fmt.Errorf("%w: can only amend draft payroll runs (current status: %s)", domain.ErrValidationFailed, run.Status)
	}

	newRun, err := s.CalculatePayrollRun(ctx, businessID, run.Period, adjustments)
	if err != nil {
		return nil, fmt.Errorf("failed to recalculate payroll: %w", err)
	}

	run.TotalGrossPay = newRun.TotalGrossPay
	run.TotalDeductions = newRun.TotalDeductions
	run.TotalNetPay = newRun.TotalNetPay
	run.TotalEmployerCosts = newRun.TotalEmployerCosts
	run.TotalCostToCompany = newRun.TotalCostToCompany
	run.Entries = newRun.Entries
	run.UpdatedAt = time.Now()

	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to save amended payroll: %w", err)
	}

	log.Info().Uint("run_id", runID).Msg("Payroll run amended successfully")
	return run, nil
}

// ReversePayrollRun marks a completed payroll run as reversed.
func (s *payrollService) ReversePayrollRun(ctx context.Context, runID, userID uint, reason string) (*domain.PayrollRun, error) {
	if reason == "" {
		return nil, fmt.Errorf("%w: reversal requires a reason", domain.ErrValidationFailed)
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	run, err := s.payrollRepo.FindByID(ctx, runID, user.BusinessID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payroll run: %w", err)
	}

	if run.Status != domain.StatusCompleted {
		return nil, fmt.Errorf("%w: can only reverse completed payroll runs", domain.ErrValidationFailed)
	}

	run.Status = domain.StatusRejected
	run.RejectionReason = fmt.Sprintf("REVERSED: %s", reason)
	run.UpdatedAt = time.Now()

	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to save reversed payroll: %w", err)
	}

	if s.notificationSvc != nil {
		go s.notificationSvc.SendEmail(context.Background(), user.Email,
			fmt.Sprintf("Payroll Reversed: %s", run.Period.Format("January 2006")),
			fmt.Sprintf("Payroll run for %s has been reversed.\n\nReason: %s\n\nNote: Bank transfers already processed cannot be undone.",
				run.Period.Format("January 2006"), reason))
	}

	log.Warn().Uint("run_id", runID).Uint("user_id", userID).Str("reason", reason).Msg("Payroll run reversed")
	return run, nil
}
