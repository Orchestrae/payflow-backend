package tax

// calculateNHF computes the National Housing Fund contribution.
// Rate: 2.5% of basic salary (employee deduction).
func calculateNHF(basicPay int64) int64 {
	return basicPay * 25 / 1000
}
