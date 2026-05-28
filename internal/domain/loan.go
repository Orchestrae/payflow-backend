package domain

import "time"

// EmployeeLoan tracks a loan given to an employee with monthly repayment deductions.
type EmployeeLoan struct {
	Model
	BusinessID       uint      `gorm:"index" json:"business_id"`
	EmployeeID       uint      `gorm:"index" json:"employee_id"`
	LoanAmount       int64     `json:"loan_amount"`        // total loan in kobo
	MonthlyDeduction int64     `json:"monthly_deduction"`  // fixed monthly repayment in kobo
	TotalRepaid      int64     `gorm:"default:0" json:"total_repaid"`
	RemainingBalance int64     `json:"remaining_balance"`
	Status           string    `gorm:"size:20;default:'active'" json:"status"` // active, completed, cancelled
	StartDate        time.Time `json:"start_date"`
	Description      string    `gorm:"size:500" json:"description"`
	Employee         *Employee `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
}
