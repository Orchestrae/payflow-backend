package tax

// Nigeria Tax Act 2025 — PAYE brackets effective January 1, 2026.
// All thresholds in kobo (100 kobo = 1 NGN).

// minimumWageMonthly is NGN 70,000 in kobo. Employees at or below this are fully tax-exempt.
const minimumWageMonthly int64 = 7_000_000

// maxRentRelief is the cap on rent relief: 20% of annual rent, max NGN 500,000 (kobo).
const maxRentRelief int64 = 50_000_000

type bracket struct {
	ceiling int64 // cumulative ceiling in kobo (0 = unlimited for top bracket)
	rate    int64 // percentage rate (integer, e.g. 15 = 15%)
}

// 2026 PAYE brackets (Nigeria Tax Act 2025)
var brackets = []bracket{
	{ceiling: 80_000_000, rate: 0},        // First NGN 800,000: 0%
	{ceiling: 300_000_000, rate: 15},      // NGN 800,001 - 3,000,000: 15%
	{ceiling: 1_200_000_000, rate: 18},    // NGN 3,000,001 - 12,000,000: 18%
	{ceiling: 2_500_000_000, rate: 21},    // NGN 12,000,001 - 25,000,000: 21%
	{ceiling: 5_000_000_000, rate: 23},    // NGN 25,000,001 - 50,000,000: 23%
	{ceiling: 0, rate: 25},                // Above NGN 50,000,000: 25%
}

// calculateMonthlyPAYE computes monthly PAYE income tax.
// Steps:
//  1. Annualize gross (x12)
//  2. Check minimum wage exemption
//  3. Deduct pension + NHF (tax-deductible items)
//  4. Apply rent relief
//  5. Apply progressive brackets to taxable income
//  6. Divide annual tax by 12 for monthly amount
func calculateMonthlyPAYE(monthlyGross, monthlyPension, monthlyNHF, annualRentPaid int64) (monthlyPAYE int64, annualTaxable int64, rentRelief int64) {
	// Minimum wage exemption
	if monthlyGross <= minimumWageMonthly {
		return 0, 0, 0
	}

	annualGross := monthlyGross * 12

	// Tax-deductible items (annualized)
	annualPension := monthlyPension * 12
	annualNHF := monthlyNHF * 12

	// Rent relief: 20% of annual rent, capped at NGN 500,000
	rentRelief = annualRentPaid * 20 / 100
	if rentRelief > maxRentRelief {
		rentRelief = maxRentRelief
	}

	// Taxable income
	annualTaxable = annualGross - annualPension - annualNHF - rentRelief
	if annualTaxable <= 0 {
		return 0, 0, rentRelief
	}

	// Apply progressive brackets
	annualTax := applyBrackets(annualTaxable)

	// Monthly PAYE
	monthlyPAYE = annualTax / 12
	return monthlyPAYE, annualTaxable, rentRelief
}

// applyBrackets applies progressive tax brackets to the annual taxable income.
// All amounts in kobo. Uses integer arithmetic only.
func applyBrackets(taxableIncome int64) int64 {
	var tax int64
	var prevCeiling int64

	for _, b := range brackets {
		if taxableIncome <= prevCeiling {
			break
		}

		var taxableInBracket int64
		if b.ceiling == 0 {
			// Top bracket (unlimited)
			taxableInBracket = taxableIncome - prevCeiling
		} else if taxableIncome > b.ceiling {
			taxableInBracket = b.ceiling - prevCeiling
		} else {
			taxableInBracket = taxableIncome - prevCeiling
		}

		tax += taxableInBracket * b.rate / 100
		prevCeiling = b.ceiling
	}

	return tax
}
