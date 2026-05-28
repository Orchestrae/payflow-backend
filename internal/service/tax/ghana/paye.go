package ghana

// Ghana 2026 PAYE brackets (GRA).
// All thresholds in pesewas (100 pesewas = 1 GHS).
// Progressive rates with 7 bands.

// Monthly brackets (annual / 12):
// Band 1: First GHS 490/month (GHS 5,880/year): 0%
// Band 2: Next GHS 110/month (GHS 1,320/year): 5%
// Band 3: Next GHS 130/month (GHS 1,560/year): 10%
// Band 4: Next GHS 3,166.67/month (GHS 38,000/year): 17.5%
// Band 5: Next GHS 16,000/month (GHS 192,000/year): 25%
// Band 6: Next GHS 30,520/month (GHS 366,240/year): 30%
// Band 7: Above: 35%

// SSNIT insurable earnings cap: GHS 69,000/month = 6,900,000 pesewas
const ssnitCap int64 = 6_900_000_00 // GHS 69,000 in pesewas

type bracket struct {
	ceiling int64 // cumulative ceiling in pesewas (0 = unlimited for top bracket)
	rate    int64 // percentage rate
}

// Annual brackets in pesewas
var brackets = []bracket{
	{ceiling: 588_000, rate: 0},       // First GHS 5,880: 0%
	{ceiling: 720_000, rate: 5},       // Next GHS 1,320: 5% (cumulative: 7,200)
	{ceiling: 876_000, rate: 10},      // Next GHS 1,560: 10% (cumulative: 8,760)
	{ceiling: 4_676_000, rate: 17},    // Next GHS 38,000: 17.5% — using 17 for integer math
	{ceiling: 23_876_000, rate: 25},   // Next GHS 192,000: 25% (cumulative: 238,760)
	{ceiling: 60_500_000, rate: 30},   // Next GHS 366,240: 30% (cumulative: 605,000)
	{ceiling: 0, rate: 35},            // Above GHS 605,000: 35%
}

// Note: Band 4 is 17.5% — we use a separate calculation for half-percent precision.

// calculateMonthlyPAYE computes Ghana monthly PAYE.
// Steps:
//  1. Deduct SSNIT (5.5%) from gross — this is tax-deductible
//  2. Annualize taxable income
//  3. Apply progressive brackets
//  4. Divide by 12
func calculateMonthlyPAYE(monthlyGross, monthlySSNIT int64) (monthlyPAYE int64, annualTaxable int64) {
	// Taxable income = gross - SSNIT employee contribution
	monthlyTaxable := monthlyGross - monthlySSNIT
	if monthlyTaxable <= 0 {
		return 0, 0
	}

	annualTaxable = monthlyTaxable * 12

	annualTax := applyBrackets(annualTaxable)
	monthlyPAYE = annualTax / 12

	return monthlyPAYE, annualTaxable
}

// applyBrackets applies Ghana's progressive tax brackets.
// All amounts in pesewas. Uses integer arithmetic.
func applyBrackets(taxableIncome int64) int64 {
	var tax int64
	var prevCeiling int64

	for _, b := range brackets {
		if taxableIncome <= prevCeiling {
			break
		}

		var taxableInBracket int64
		if b.ceiling == 0 {
			taxableInBracket = taxableIncome - prevCeiling
		} else if taxableIncome > b.ceiling {
			taxableInBracket = b.ceiling - prevCeiling
		} else {
			taxableInBracket = taxableIncome - prevCeiling
		}

		// Band 4 is actually 17.5% — handle the half percent
		if b.rate == 17 {
			tax += taxableInBracket * 175 / 1000
		} else {
			tax += taxableInBracket * b.rate / 100
		}

		prevCeiling = b.ceiling
	}

	return tax
}
