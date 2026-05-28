package ghana

// Calculate computes all Ghanaian statutory deductions for one employee for one month.
// Pure function — no side effects, no database access, fully testable.
// All amounts in pesewas (100 pesewas = 1 GHS).
func Calculate(input Input) Result {
	var r Result

	// Pension base is basic salary in Ghana (SSNIT is on basic only)
	r.PensionBase = input.BasicPay
	r.AnnualGross = input.GrossPay * 12

	// 1. Pension (SSNIT) — must be computed before PAYE since it's tax-deductible
	if input.PensionEnabled && input.BasicPay > 0 {
		r.EmployeePension = calculateEmployeePension(input.BasicPay)
		r.EmployerPension = calculateEmployerPension(input.BasicPay)
		r.Tier2Pension = calculateTier2Pension(input.BasicPay)
	}

	// 2. PAYE — computed after pension since SSNIT employee contribution is tax-deductible
	if input.PAYEEnabled && input.GrossPay > 0 {
		paye, taxable := calculateMonthlyPAYE(input.GrossPay, r.EmployeePension)
		r.PAYE = paye
		r.AnnualTaxableIncome = taxable
	}

	// Employee deductions: SSNIT 5.5% + Tier 2 5% + PAYE
	r.TotalEmployeeDeductions = r.EmployeePension + r.Tier2Pension + r.PAYE

	// Employer costs: SSNIT 13% (includes 2.5% NHIA redirect)
	r.TotalEmployerCosts = r.EmployerPension

	return r
}
