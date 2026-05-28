package tax

import (
	"payflow/internal/service/tax/ghana"
)

// CalculateForCountry routes tax computation to the correct country engine.
// currency: "NGN" for Nigeria, "GHS" for Ghana.
// Falls back to Nigeria engine for unknown currencies.
func CalculateForCountry(currency string, input Input) Result {
	switch currency {
	case "GHS":
		return calculateGhana(input)
	default:
		return Calculate(input) // Nigeria (default)
	}
}

// calculateGhana maps the generic Input to Ghana-specific input and back.
func calculateGhana(input Input) Result {
	ghResult := ghana.Calculate(ghana.Input{
		BasicPay:       input.BasicPay,
		HousingPay:     input.HousingPay,
		TransportPay:   input.TransportPay,
		OtherPay:       input.OtherPay,
		GrossPay:       input.GrossPay,
		PensionEnabled: input.PensionEnabled,
		PAYEEnabled:    input.PAYEEnabled,
	})

	return Result{
		PAYE:            ghResult.PAYE,
		EmployeePension: ghResult.EmployeePension + ghResult.Tier2Pension, // Combined employee pension
		NHF:             0, // Ghana has no NHF equivalent
		EmployerPension: ghResult.EmployerPension,
		NSITF:           0, // Ghana has no NSITF equivalent
		PensionBase:     ghResult.PensionBase,
		AnnualGross:     ghResult.AnnualGross,
		AnnualTaxableIncome:     ghResult.AnnualTaxableIncome,
		TotalEmployeeDeductions: ghResult.TotalEmployeeDeductions,
		TotalEmployerCosts:      ghResult.TotalEmployerCosts,
	}
}
