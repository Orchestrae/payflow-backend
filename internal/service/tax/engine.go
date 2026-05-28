package tax

// Calculate computes all Nigerian statutory deductions for one employee for one month.
// Pure function — no side effects, no database access, fully testable.
// All amounts in kobo (100 kobo = 1 NGN).
func Calculate(input Input) Result {
	var r Result

	pensionBase := input.BasicPay + input.HousingPay + input.TransportPay
	r.PensionBase = pensionBase
	r.AnnualGross = input.GrossPay * 12

	// 1. Pension (RSA) — must be computed before PAYE since it's tax-deductible
	if input.PensionEnabled && pensionBase > 0 {
		r.EmployeePension = calculateEmployeePension(pensionBase)
		r.EmployerPension = calculateEmployerPension(pensionBase)
	}

	// 2. NHF — must be computed before PAYE since it's tax-deductible
	if input.NHFEnabled && input.BasicPay > 0 {
		r.NHF = calculateNHF(input.BasicPay)
	}

	// 3. NSITF (employer-only cost)
	if input.NSITFEnabled && input.GrossPay > 0 {
		r.NSITF = calculateNSITF(input.GrossPay)
	}

	// 4. PAYE — computed last because pension and NHF are tax-deductible
	if input.PAYEEnabled && input.GrossPay > 0 {
		paye, taxable, relief := calculateMonthlyPAYE(
			input.GrossPay,
			r.EmployeePension,
			r.NHF,
			input.AnnualRentPaid,
		)
		r.PAYE = paye
		r.AnnualTaxableIncome = taxable
		r.RentRelief = relief
	}

	r.TotalEmployeeDeductions = r.EmployeePension + r.NHF + r.PAYE
	r.TotalEmployerCosts = r.EmployerPension + r.NSITF

	return r
}
