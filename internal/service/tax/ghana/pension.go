package ghana

// Ghana Three-Tier Pension System (National Pensions Act 766):
// - Employee: 5.5% of basic salary (to SSNIT Tier 1)
// - Employer: 13% of basic salary, split:
//     - 11% to SSNIT Tier 1
//     - 2.5% to NHIA (health insurance, deducted from employer's 13%)
// - Tier 2: 5% of basic salary (from the 18.5% total, managed by private trustees)
//
// Total: Employee 5.5% + Employer 13% = 18.5%
//   Tier 1 (SSNIT): 13.5% (11% pension + 2.5% NHIA)
//   Tier 2: 5%
//
// SSNIT insurable earnings cap: GHS 69,000/month (2026)

const ssnitCapMonthly int64 = 6_900_000 // GHS 69,000 in pesewas

// calculateEmployeePension computes employee's 5.5% SSNIT contribution.
func calculateEmployeePension(basicPay int64) int64 {
	base := basicPay
	if base > ssnitCapMonthly {
		base = ssnitCapMonthly
	}
	return base * 55 / 1000 // 5.5%
}

// calculateEmployerPension computes employer's 13% SSNIT contribution.
func calculateEmployerPension(basicPay int64) int64 {
	base := basicPay
	if base > ssnitCapMonthly {
		base = ssnitCapMonthly
	}
	return base * 13 / 100 // 13%
}

// calculateTier2Pension computes the mandatory Tier 2 (5%) occupational pension.
// This is part of the 18.5% total, managed by private trustees.
func calculateTier2Pension(basicPay int64) int64 {
	base := basicPay
	if base > ssnitCapMonthly {
		base = ssnitCapMonthly
	}
	return base * 5 / 100 // 5%
}
