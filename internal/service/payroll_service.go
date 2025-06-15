// internal/service/payroll_service.go
package service

import (
	"context"
	"fmt"
	"payflow/internal/domain"
	"payflow/internal/repository"
	"time"

	"github.com/rs/zerolog/log"
)

type payrollService struct {
	payrollRepo     repository.PayrollRepository
	employeeRepo    repository.EmployeeRepository // Assumed to be created
	cadreRepo       repository.CadreRepository    // Assumed to be created
	txer            repository.Transactioner
	payoutSvc       PayoutService
	notificationSvc NotificationService
	scheduler       domain.Scheduler
	userRepo        repository.UserRepository
}

// ... (inside the payrollService struct and NewPayrollService function)

// RejectPayrollRun handles the rejection of a payroll run.
// It can only be called on a run that is 'pending_approval'.
func (s *payrollService) RejectPayrollRun(ctx context.Context, runID, rejecterID uint, reason string) (*domain.PayrollRun, error) {
	log.Ctx(ctx).Info().Uint("run_id", runID).Uint("user_id", rejecterID).Msg("Attempting to reject payroll run")

	// 1. Basic Validation: Ensure a reason is provided.
	if reason == "" {
		return nil, fmt.Errorf("%w: rejection requires a reason", domain.ErrValidationFailed)
	}

	// 2. Fetch the payroll run from the database.
	run, err := s.payrollRepo.FindByID(ctx, runID)
	if err != nil {
		// This will correctly return domain.ErrNotFound if the ID is invalid.
		return nil, err
	}

	// 3. Business Rule: Check for correct state.
	// A payroll can only be rejected if it's waiting for approval.
	if run.Status != domain.StatusPendingApproval {
		log.Ctx(ctx).Warn().
			Str("current_status", string(run.Status)).
			Msg("Attempted to reject a payroll run not in 'pending_approval' state")
		return nil, fmt.Errorf("%w: cannot reject a payroll run with status '%s'", domain.ErrValidationFailed, run.Status)
	}

	// 4. Permission Check: Ensure the user belongs to the same business.
	// A more advanced check could verify if the rejecterID has the 'Approver' role.
	// For now, we assume role checks are handled by middleware at the API layer.
	rejecter, err := s.userRepo.FindByID(ctx, rejecterID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not find rejecting user")
		return nil, domain.ErrInternalServer // The user from the token should always exist.
	}
	if rejecter.BusinessID != run.BusinessID {
		log.Ctx(ctx).Error().
			Uint("user_business_id", rejecter.BusinessID).
			Uint("run_business_id", run.BusinessID).
			Msg("Permission denied: user tried to reject a run for another business")
		return nil, domain.ErrForbidden
	}

	// 5. Update the run's status and record the reason.
	// The run is sent back to 'draft' so the Operator can correct it.
	run.Status = domain.StatusRejected
	run.RejectionReason = reason
	run.UpdatedAt = time.Now() // Explicitly set update time for clarity

	if err := s.payrollRepo.Update(ctx, run); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to update payroll run status to rejected")
		return nil, err
	}

	log.Ctx(ctx).Info().Msg("Payroll run successfully rejected")

	// 6. Side Effect: Notify the operator(s) that the run was rejected.
	// This is a crucial feedback loop for the user.
	operators, err := s.userRepo.FindOperatorsByBusinessID(ctx, run.BusinessID) // Assumes this repo method exists
	if err != nil {
		// Non-fatal error. The core action succeeded, but we should log the notification failure.
		log.Ctx(ctx).Error().Err(err).Msg("Failed to fetch operators for rejection notification")
	} else {
		for _, operator := range operators {
			emailSubject := fmt.Sprintf("Action Required: Payroll for %s was Rejected", run.Period.Format("January 2006"))
			emailBody := fmt.Sprintf(
				"Hello %s,\n\nThe payroll run for the period %s has been rejected by an approver.\n\nReason: %s\n\nPlease log in to PayFlow to review and correct the details before resubmitting.\n\nThank you,\nThe PayFlow Team",
				operator.Email, // Ideally, we'd have the operator's name
				run.Period.Format("January 2006"),
				reason,
			)
			// Send the email. We log errors but don't fail the entire operation if an email fails.
			go s.notificationSvc.SendEmail(context.Background(), operator.Email, emailSubject, emailBody)
		}
	}

	return run, nil
}

// NewPayrollService creates a new payroll service instance.
func NewPayrollService(
	payrollRepo repository.PayrollRepository,
	employeeRepo repository.EmployeeRepository,
	cadreRepo repository.CadreRepository,
	txer repository.Transactioner,
	payoutSvc PayoutService,
	notificationSvc NotificationService,
	userRepo repository.UserRepository,
	scheduler domain.Scheduler,
) PayrollService {
	return &payrollService{
		payrollRepo:     payrollRepo,
		employeeRepo:    employeeRepo,
		cadreRepo:       cadreRepo,
		txer:            txer,
		payoutSvc:       payoutSvc,
		notificationSvc: notificationSvc,
		userRepo:        userRepo,
		scheduler:       scheduler,
	}
}

// CalculatePayrollRun performs an in-memory calculation of the payroll.
func (s *payrollService) CalculatePayrollRun(ctx context.Context, businessID uint, adjustments map[uint]int64) (*domain.PayrollRun, error) {
	// 1. Fetch all active employees for the business with their cadres preloaded.
	employees, err := s.employeeRepo.FindActiveByBusinessID(ctx, businessID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch active employees: %w", err)
	}

	if len(employees) == 0 {
		return nil, fmt.Errorf("no active employees found for business to run payroll")
	}

	// 2. For each employee, calculate their pay.
	var totalGross, totalDeductions, totalNet int64
	runEntries := make([]domain.PayrollRunEntry, 0, len(employees))

	for _, emp := range employees {
		if emp.Cadre == nil {
			// Skip employees without a valid cadre, or return an error.
			// Logging this is a good practice.
			continue
		}

		var entryGross, entryDeductions, entryBonus int64
		details := make([]domain.PayrollRunEntryDetail, 0)

		// Calculate Gross Pay from earning components
		for _, ec := range emp.Cadre.EarningComponents {
			entryGross += ec.Amount
			details = append(details, domain.PayrollRunEntryDetail{
				Type:   domain.DetailTypeEarning,
				Name:   ec.Name,
				Amount: ec.Amount,
			})
		}

		// Calculate Deductions from cadre rules
		for _, dr := range emp.Cadre.DeductionRules {
			var deductionAmount int64
			if dr.Type == domain.DeductionTypePercentage {
				// Note: Using float64 for money is risky. This is simplified.
				// Production systems should use decimal types or careful int64 math.
				deductionAmount = int64(float64(entryGross) * (dr.Value / 100.0))
			} else {
				deductionAmount = int64(dr.Value)
			}
			entryDeductions += deductionAmount
			details = append(details, domain.PayrollRunEntryDetail{
				Type:   domain.DetailTypeDeduction,
				Name:   dr.Name,
				Amount: deductionAmount,
			})
		}

		// Apply any one-time adjustments (bonuses/deductions)
		if adj, ok := adjustments[emp.ID]; ok {
			entryBonus = adj
			details = append(details, domain.PayrollRunEntryDetail{
				Type:   domain.DetailTypeBonus,
				Name:   "One-time Adjustment",
				Amount: adj, // Can be negative
			})
		}

		entryNet := entryGross + entryBonus - entryDeductions

		runEntries = append(runEntries, domain.PayrollRunEntry{
			EmployeeID:      emp.ID,
			Employee:        &emp,
			GrossPay:        entryGross,
			TotalDeductions: entryDeductions,
			Bonuses:         entryBonus,
			NetPay:          entryNet,
			Details:         details,
		})

		totalGross += entryGross
		totalDeductions += entryDeductions
		totalNet += entryNet
	}

	// 3. Assemble the final payroll run object.
	now := time.Now()
	payrollRun := &domain.PayrollRun{
		BusinessID:      businessID,
		Period:          time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
		Status:          domain.StatusDraft,
		TotalGrossPay:   totalGross,
		TotalDeductions: totalDeductions,
		TotalNetPay:     totalNet,
		ScheduledFor:    now.AddDate(0, 0, 5), // Default schedule: 5 days from now
		Entries:         runEntries,
	}

	return payrollRun, nil
}

func (s *payrollService) CreateAndStorePayrollRun(ctx context.Context, businessID uint, adjustments map[uint]int64) (*domain.PayrollRun, error) {
	payrollRun, err := s.CalculatePayrollRun(ctx, businessID, adjustments)
	if err != nil {
		return nil, err
	}
	if err := s.payrollRepo.Create(ctx, payrollRun); err != nil {
		return nil, fmt.Errorf("failed to save payroll run: %w", err)
	}
	return payrollRun, nil
}

// Implementation for Submit, Approve, Reject would follow a similar pattern:
// 1. Fetch the run.
// 2. Check current status and user permissions.
// 3. Update status.
// 4. Save changes.
// 5. Trigger notifications or other side effects.
func (s *payrollService) ApprovePayrollRun(ctx context.Context, runID, approverID uint) (*domain.PayrollRun, error) {
	run, err := s.payrollRepo.FindByID(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payroll run: %w", err)
	}

	if run.Status != domain.StatusPendingApproval {
		return nil, domain.ErrValidationFailed // Or a more specific error
	}

	run.Status = domain.StatusApproved
	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return nil, err
	}

	// --- THIS IS THE KEY SIDE EFFECT ---
	if err := s.scheduler.SchedulePayout(*run); err != nil {
		// If scheduling fails, we should ideally roll back the approval.
		// This highlights the need for robust transaction management across services.
		log.Error().Err(err).Uint("runID", runID).Msg("CRITICAL: Failed to schedule payout for approved run")
		// Revert status back to pending approval
		run.Status = domain.StatusPendingApproval
		_ = s.payrollRepo.Update(ctx, run)
		return nil, fmt.Errorf("could not schedule payout job: %w", err)
	}

	return run, nil
}

// ... update NewPayrollService to accept NotificationService

func (s *payrollService) SubmitForApproval(ctx context.Context, runID, userID uint) (*domain.PayrollRun, error) {
	// 1. Fetch the payroll run
	run, err := s.payrollRepo.FindByID(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payroll run: %w", err)
	}

	// 2. Business Rule: Check status
	if run.Status != domain.StatusDraft {
		return nil, fmt.Errorf("%w: cannot submit non-draft run", domain.ErrValidationFailed)
	}

	// 3. Business Rule: Check permissions (optional, can also be at API layer)
	// Here we might check if userID belongs to run.BusinessID

	// 4. Update status
	run.Status = domain.StatusPendingApproval
	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to update payroll run status: %w", err)
	}

	// 5. Side Effect: Notify approvers
	// TODO: Implement approver notification
	// approvers, err := s.userRepo.FindApproversByBusinessID(ctx, run.BusinessID)
	// if err != nil {
	//     log.Error().Err(err).Msg("Failed to fetch approvers for notification")
	// } else {
	//     for _, approver := range approvers {
	//         s.notificationSvc.SendEmail(approver.Email, "Payroll Pending Approval", "...")
	//     }
	// }

	return run, nil
}

// ProcessApprovedPayroll implements the method required by the scheduler's interface.
func (s *payrollService) ProcessApprovedPayroll(ctx context.Context, runID uint) error {
	log.Ctx(ctx).Info().Uint("run_id", runID).Msg("Processing approved payroll")

	// 1. Fetch run, ensure it's in 'Approved' status.
	run, err := s.payrollRepo.FindByID(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to fetch payroll run: %w", err)
	}
	if run.Status != domain.StatusApproved {
		log.Ctx(ctx).Warn().Str("status", string(run.Status)).Msg("Skipping payroll processing, not in approved state")
		return nil // Not a fatal error, just skip.
	}

	// 2. Update status to 'Processing'
	run.Status = domain.StatusProcessing
	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return fmt.Errorf("failed to update payroll run status to processing: %w", err)
	}

	// 3. Call the external payment gateway service.
	ref, err := s.payoutSvc.DisburseBulkPayment(ctx, *run)
	if err != nil {
		// Payment failed, update status to 'Failed'
		run.Status = domain.StatusFailed
		if updateErr := s.payrollRepo.Update(ctx, run); updateErr != nil {
			log.Ctx(ctx).Error().Err(updateErr).Msg("Failed to update payroll run status to failed")
		}
		log.Ctx(ctx).Error().Err(err).Msg("Disbursement failed")
		// TODO: Notify admin of failure
		return fmt.Errorf("disbursement failed: %w", err)
	}

	// 4. Payment succeeded, update status to 'Completed'.
	run.Status = domain.StatusCompleted
	run.PaymentReference = ref
	now := time.Now()
	run.ProcessedAt = &now
	if err := s.payrollRepo.Update(ctx, run); err != nil {
		// Log this critically. The payment went through but we failed to record it.
		log.Ctx(ctx).Error().Err(err).Msg("CRITICAL: Disbursement succeeded but failed to update run status")
		return fmt.Errorf("failed to update payroll run status to completed: %w", err)
	}

	log.Ctx(ctx).Info().Msg("Payroll processing completed successfully")
	// TODO: Notify admin of success
	return nil
}

// ListByBusinessID retrieves all payroll runs for a business
func (s *payrollService) ListByBusinessID(ctx context.Context, businessID uint) ([]domain.PayrollRun, error) {
	return s.payrollRepo.FindAllByBusinessID(ctx, businessID)
}

// GetByID retrieves a specific payroll run by ID, ensuring it belongs to the specified business
func (s *payrollService) GetByID(ctx context.Context, runID, businessID uint) (*domain.PayrollRun, error) {
	run, err := s.payrollRepo.FindByID(ctx, runID)
	if err != nil {
		return nil, err
	}

	// Ensure the run belongs to the specified business
	if run.BusinessID != businessID {
		return nil, domain.ErrForbidden
	}

	return run, nil
}
