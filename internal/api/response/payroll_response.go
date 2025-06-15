package response

import (
	"payflow/internal/domain"
	"time"
)

type PayrollRunResponse struct {
	ID               uint      `json:"id"`
	BusinessID       uint      `json:"business_id"`
	Status           string    `json:"status"`
	ScheduledFor     time.Time `json:"scheduled_for"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	TotalGrossPay    int64     `json:"total_gross_pay"`
	TotalDeductions  int64     `json:"total_deductions"`
	TotalNetPay      int64     `json:"total_net_pay"`
	EmployeeCount    int       `json:"employee_count"`
	PaymentReference string    `json:"payment_reference,omitempty"`
}

func NewPayrollRunResponse(run domain.PayrollRun) PayrollRunResponse {
	return PayrollRunResponse{
		ID:               run.ID,
		BusinessID:       run.BusinessID,
		Status:           string(run.Status),
		ScheduledFor:     run.ScheduledFor,
		CreatedAt:        run.CreatedAt,
		UpdatedAt:        run.UpdatedAt,
		TotalGrossPay:    run.TotalGrossPay,
		TotalDeductions:  run.TotalDeductions,
		TotalNetPay:      run.TotalNetPay,
		EmployeeCount:    len(run.Entries),
		PaymentReference: run.PaymentReference,
	}
}
