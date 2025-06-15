// internal/api/request/payroll_request.go
package request

// Adjustment represents a one-time bonus or deduction for an employee in a payroll run.
type Adjustment struct {
	EmployeeID uint  `json:"employee_id" validate:"required"`
	Amount     int64 `json:"amount"` // Positive for bonus, negative for deduction
}

// CreatePayrollRunRequest is the body for initiating a new payroll run.
type CreatePayrollRunRequest struct {
	// A map where the key is the string representation of employee ID
	Adjustments map[string]int64 `json:"adjustments"`
}

type RejectPayrollRequest struct {
	Reason string `json:"reason" validate:"required"`
}
