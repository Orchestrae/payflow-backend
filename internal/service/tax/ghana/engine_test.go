package ghana

import (
	"testing"
)

// Helper: convert GHS to pesewas
func ghs(amount int64) int64 {
	return amount * 100
}

func TestCalculate_AllDisabled(t *testing.T) {
	result := Calculate(Input{
		BasicPay: ghs(5000),
		GrossPay: ghs(7000),
	})
	if result.PAYE != 0 || result.EmployeePension != 0 || result.EmployerPension != 0 {
		t.Errorf("all flags disabled should produce zero, got PAYE=%d Pension=%d", result.PAYE, result.EmployeePension)
	}
}

func TestCalculate_SSNITContributions(t *testing.T) {
	// Basic: GHS 5,000/month
	result := Calculate(Input{
		BasicPay:       ghs(5000),
		GrossPay:       ghs(7000),
		PensionEnabled: true,
	})

	// Employee: 5.5% of 5,000 = 275
	expectedEmployee := ghs(5000) * 55 / 1000
	if result.EmployeePension != expectedEmployee {
		t.Errorf("expected employee SSNIT %d, got %d", expectedEmployee, result.EmployeePension)
	}

	// Employer: 13% of 5,000 = 650
	expectedEmployer := ghs(5000) * 13 / 100
	if result.EmployerPension != expectedEmployer {
		t.Errorf("expected employer SSNIT %d, got %d", expectedEmployer, result.EmployerPension)
	}

	// Tier 2: 5% of 5,000 = 250
	expectedTier2 := ghs(5000) * 5 / 100
	if result.Tier2Pension != expectedTier2 {
		t.Errorf("expected Tier 2 %d, got %d", expectedTier2, result.Tier2Pension)
	}
}

func TestCalculate_SSNITCap(t *testing.T) {
	// Basic: GHS 80,000/month (above GHS 69,000 cap)
	result := Calculate(Input{
		BasicPay:       ghs(80000),
		GrossPay:       ghs(80000),
		PensionEnabled: true,
	})

	// Should be capped at GHS 69,000
	cappedBase := ghs(69000)
	expectedEmployee := cappedBase * 55 / 1000
	if result.EmployeePension != expectedEmployee {
		t.Errorf("SSNIT should be capped: expected %d, got %d", expectedEmployee, result.EmployeePension)
	}
}

func TestCalculate_PAYELowIncome(t *testing.T) {
	// GHS 490/month gross = within tax-free band (GHS 5,880/year)
	result := Calculate(Input{
		BasicPay:    ghs(490),
		GrossPay:    ghs(490),
		PAYEEnabled: true,
	})
	if result.PAYE != 0 {
		t.Errorf("income within tax-free band should have PAYE=0, got %d", result.PAYE)
	}
}

func TestCalculate_PAYEBasicScenario(t *testing.T) {
	// GHS 3,000/month, no pension deduction
	// Annual = 36,000
	// Band 1: first 5,880 at 0% = 0
	// Band 2: next 1,320 at 5% = 66
	// Band 3: next 1,560 at 10% = 156
	// Band 4: next 27,240 (36,000 - 8,760) at 17.5% = 4,767
	// Annual tax = 4,989
	// Monthly = 4,989 / 12 = 415.75 → 415 (integer division)
	result := Calculate(Input{
		BasicPay:    ghs(2000),
		OtherPay:    ghs(1000),
		GrossPay:    ghs(3000),
		PAYEEnabled: true,
	})

	// Manual calculation: let me verify with pesewas
	// Annual gross = 3000 * 100 * 12 = 3,600,000 pesewas = GHS 36,000
	// Band 1: 588,000 at 0% = 0
	// Band 2: (720,000 - 588,000) = 132,000 at 5% = 6,600
	// Band 3: (876,000 - 720,000) = 156,000 at 10% = 15,600
	// Band 4: (3,600,000 - 876,000) = 2,724,000 at 17.5% = 476,700
	// Annual tax = 498,900 pesewas
	// Monthly = 498,900 / 12 = 41,575 pesewas = GHS 415.75

	expectedMonthly := int64(41575)
	if result.PAYE != expectedMonthly {
		t.Errorf("expected monthly PAYE %d (GHS %.2f), got %d (GHS %.2f)",
			expectedMonthly, float64(expectedMonthly)/100, result.PAYE, float64(result.PAYE)/100)
	}
}

func TestCalculate_PAYEWithSSNIT(t *testing.T) {
	// GHS 5,000/month gross, SSNIT enabled
	// SSNIT employee = 5.5% of 5,000 = 275
	// Taxable = 5,000 - 275 = 4,725/month
	// Annual taxable = 56,700
	// Band 1: 5,880 at 0% = 0
	// Band 2: 1,320 at 5% = 66
	// Band 3: 1,560 at 10% = 156
	// Band 4: 38,000 at 17.5% = 6,650
	// Band 5: (56,700 - 46,760) = 9,940 at 25% = 2,485
	// Annual tax = 9,357 GHS
	// In pesewas: 935,700
	// Monthly: 935,700/12 = 77,975

	result := Calculate(Input{
		BasicPay:       ghs(5000),
		GrossPay:       ghs(5000),
		PensionEnabled: true,
		PAYEEnabled:    true,
	})

	// Let me recalculate in pesewas precisely:
	// SSNIT = 500000 * 55 / 1000 = 27500 pesewas/month
	// Monthly taxable = 500000 - 27500 = 472500
	// Annual taxable = 472500 * 12 = 5,670,000
	// Band 1: 588,000 at 0% = 0
	// Band 2: 132,000 at 5% = 6,600
	// Band 3: 156,000 at 10% = 15,600
	// Band 4: 3,800,000 at 17.5% = 665,000
	// Band 5: (5,670,000 - 4,676,000) = 994,000 at 25% = 248,500
	// Total annual = 935,700
	// Monthly = 935,700 / 12 = 77,975

	expectedMonthly := int64(77975)
	if result.PAYE != expectedMonthly {
		t.Errorf("expected monthly PAYE %d (GHS %.2f), got %d (GHS %.2f)",
			expectedMonthly, float64(expectedMonthly)/100, result.PAYE, float64(result.PAYE)/100)
	}
}

func TestCalculate_PAYEHighIncome(t *testing.T) {
	// GHS 60,000/month gross, no pension (to simplify)
	// Annual = 720,000
	// Band 1: 5,880 at 0% = 0
	// Band 2: 1,320 at 5% = 66
	// Band 3: 1,560 at 10% = 156
	// Band 4: 38,000 at 17.5% = 6,650
	// Band 5: 192,000 at 25% = 48,000
	// Band 6: 366,240 at 30% = 109,872
	// Band 7: (720,000 - 605,000) = 115,000 at 35% = 40,250
	// Annual tax = 204,994 GHS
	// Pesewas: 20,499,400
	// Monthly: 20,499,400/12 = 1,708,283

	result := Calculate(Input{
		BasicPay:    ghs(40000),
		OtherPay:    ghs(20000),
		GrossPay:    ghs(60000),
		PAYEEnabled: true,
	})

	// Pesewas calc:
	// Band 1: 588,000 at 0% = 0
	// Band 2: 132,000 at 5% = 6,600
	// Band 3: 156,000 at 10% = 15,600
	// Band 4: 3,800,000 at 17.5% = 665,000
	// Band 5: 19,200,000 at 25% = 4,800,000
	// Band 6: 36,624,000 at 30% = 10,987,200
	// Band 7: (72,000,000 - 60,500,000) = 11,500,000 at 35% = 4,025,000
	// Total = 20,499,400
	// Monthly = 20,499,400 / 12 = 1,708,283

	expectedMonthly := int64(1708283)
	if result.PAYE != expectedMonthly {
		t.Errorf("expected monthly PAYE %d (GHS %.2f), got %d (GHS %.2f)",
			expectedMonthly, float64(expectedMonthly)/100, result.PAYE, float64(result.PAYE)/100)
	}
}

func TestCalculate_FullStatutory(t *testing.T) {
	// GHS 5,000/month, all enabled
	result := Calculate(Input{
		BasicPay:       ghs(5000),
		GrossPay:       ghs(5000),
		PensionEnabled: true,
		PAYEEnabled:    true,
	})

	// Verify all components non-zero
	if result.EmployeePension == 0 {
		t.Error("employee SSNIT should be non-zero")
	}
	if result.EmployerPension == 0 {
		t.Error("employer SSNIT should be non-zero")
	}
	if result.Tier2Pension == 0 {
		t.Error("Tier 2 pension should be non-zero")
	}
	if result.PAYE == 0 {
		t.Error("PAYE should be non-zero")
	}

	// Employee deductions = SSNIT 5.5% + Tier2 5% + PAYE
	expectedTotal := result.EmployeePension + result.Tier2Pension + result.PAYE
	if result.TotalEmployeeDeductions != expectedTotal {
		t.Errorf("expected total employee deductions %d, got %d", expectedTotal, result.TotalEmployeeDeductions)
	}

	// Employer costs = SSNIT 13% only
	if result.TotalEmployerCosts != result.EmployerPension {
		t.Errorf("expected total employer costs %d, got %d", result.EmployerPension, result.TotalEmployerCosts)
	}
}

func TestApplyBrackets_TaxFreeOnly(t *testing.T) {
	// GHS 5,880 annual = exactly tax-free
	tax := applyBrackets(588_000)
	if tax != 0 {
		t.Errorf("expected 0 tax for tax-free band, got %d", tax)
	}
}

func TestApplyBrackets_JustAboveTaxFree(t *testing.T) {
	// GHS 5,881 annual = 1 GHS in 5% band
	tax := applyBrackets(588_100)
	expected := int64(100 * 5 / 100) // 100 pesewas * 5% = 5 pesewas
	if tax != expected {
		t.Errorf("expected %d, got %d", expected, tax)
	}
}
