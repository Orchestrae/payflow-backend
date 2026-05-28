package tax

// calculateEmployeePension computes the employee's 8% RSA contribution.
// Base = Basic + Housing + Transport (monthly, kobo).
func calculateEmployeePension(pensionBase int64) int64 {
	return pensionBase * 8 / 100
}

// calculateEmployerPension computes the employer's 10% RSA contribution.
// Base = Basic + Housing + Transport (monthly, kobo).
func calculateEmployerPension(pensionBase int64) int64 {
	return pensionBase * 10 / 100
}
