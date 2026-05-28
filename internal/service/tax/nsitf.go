package tax

// calculateNSITF computes the NSITF Employees' Compensation Scheme contribution.
// Rate: 1% of total gross payroll (employer-only cost, no employee deduction).
func calculateNSITF(grossPay int64) int64 {
	return grossPay * 1 / 100
}
