// internal/api/request/payroll_request.go
package request

// AdjustmentItem represents a single adjustment item for an employee.
// This structure allows for detailed historical tracking of adjustments.
type AdjustmentItem struct {
	ItemName      string `json:"item_name" validate:"required"`      // Name of the adjustment item (e.g., "bonus", "penalty")
	Amount        int64  `json:"amount" validate:"required"`         // Positive for earnings, negative for deductions
	Description   string `json:"description,omitempty"`              // Optional description for historical tracking
	ComponentType string `json:"component_type,omitempty"`           // Optional: "earnings" or "deduction" (auto-inferred from amount if not provided)
}

// CreatePayrollRunRequest is the body for initiating a new payroll run.
type CreatePayrollRunRequest struct {
	// Period is the payroll period (year and month). Format: "YYYY-MM" or "YYYY-MM-DD"
	// If not provided, defaults to current month
	Period string `json:"period,omitempty"` // e.g., "2026-01" for January 2026
	
	// Adjustments is a map where:
	// - Key: Employee identifier (can be employee ID as string or email)
	// - Value: Array of adjustment items for that employee
	// This allows multiple adjustments per employee with detailed tracking
	Adjustments map[string][]AdjustmentItem `json:"adjustments,omitempty"`
}

type RejectPayrollRequest struct {
	Reason string `json:"reason" validate:"required"`
}
