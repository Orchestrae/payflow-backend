package tax

import (
	"testing"
)

// Helper: convert NGN to kobo
func ngn(amount int64) int64 {
	return amount * 100
}

func TestCalculate_AllDisabled(t *testing.T) {
	result := Calculate(Input{
		BasicPay:   ngn(200000),
		HousingPay: ngn(50000),
		GrossPay:   ngn(300000),
	})
	if result.PAYE != 0 || result.EmployeePension != 0 || result.NHF != 0 || result.NSITF != 0 {
		t.Errorf("all flags disabled should produce zero deductions, got PAYE=%d Pension=%d NHF=%d NSITF=%d",
			result.PAYE, result.EmployeePension, result.NHF, result.NSITF)
	}
}

func TestCalculate_MinimumWageExemption(t *testing.T) {
	// NGN 70,000/month = minimum wage, should be PAYE-exempt
	result := Calculate(Input{
		BasicPay:       ngn(50000),
		HousingPay:     ngn(10000),
		TransportPay:   ngn(10000),
		GrossPay:       ngn(70000),
		PAYEEnabled:    true,
		PensionEnabled: true,
	})
	if result.PAYE != 0 {
		t.Errorf("minimum wage should be PAYE-exempt, got %d", result.PAYE)
	}
	// Pension still applies even at minimum wage
	pensionBase := ngn(50000) + ngn(10000) + ngn(10000) // 70K
	expectedPension := pensionBase * 8 / 100
	if result.EmployeePension != expectedPension {
		t.Errorf("expected employee pension %d, got %d", expectedPension, result.EmployeePension)
	}
}

func TestCalculate_BelowMinimumWage(t *testing.T) {
	result := Calculate(Input{
		BasicPay:    ngn(40000),
		GrossPay:    ngn(50000),
		PAYEEnabled: true,
	})
	if result.PAYE != 0 {
		t.Errorf("below minimum wage should be PAYE-exempt, got %d", result.PAYE)
	}
}

func TestCalculate_Pension(t *testing.T) {
	input := Input{
		BasicPay:       ngn(200000),
		HousingPay:     ngn(50000),
		TransportPay:   ngn(30000),
		OtherPay:       ngn(20000),
		GrossPay:       ngn(300000),
		PensionEnabled: true,
	}
	result := Calculate(input)

	pensionBase := ngn(200000) + ngn(50000) + ngn(30000) // 280,000 NGN
	expectedEmployee := pensionBase * 8 / 100             // 22,400 NGN
	expectedEmployer := pensionBase * 10 / 100            // 28,000 NGN

	if result.EmployeePension != expectedEmployee {
		t.Errorf("expected employee pension %d, got %d", expectedEmployee, result.EmployeePension)
	}
	if result.EmployerPension != expectedEmployer {
		t.Errorf("expected employer pension %d, got %d", expectedEmployer, result.EmployerPension)
	}
	if result.PensionBase != pensionBase {
		t.Errorf("expected pension base %d, got %d", pensionBase, result.PensionBase)
	}
}

func TestCalculate_NHF(t *testing.T) {
	result := Calculate(Input{
		BasicPay:   ngn(200000),
		GrossPay:   ngn(300000),
		NHFEnabled: true,
	})
	expected := ngn(200000) * 25 / 1000 // 2.5% of basic = 5,000 NGN
	if result.NHF != expected {
		t.Errorf("expected NHF %d, got %d", expected, result.NHF)
	}
}

func TestCalculate_NSITF(t *testing.T) {
	result := Calculate(Input{
		GrossPay:     ngn(300000),
		NSITFEnabled: true,
	})
	expected := ngn(300000) * 1 / 100 // 1% of gross = 3,000 NGN
	if result.NSITF != expected {
		t.Errorf("expected NSITF %d, got %d", expected, result.NSITF)
	}
	// NSITF should be employer cost, not employee deduction
	if result.TotalEmployeeDeductions != 0 {
		t.Errorf("NSITF should not be in employee deductions, got %d", result.TotalEmployeeDeductions)
	}
}

func TestCalculate_PAYEBasicScenario(t *testing.T) {
	// NGN 300,000/month, no pension/NHF, no rent relief
	// Annual = 3,600,000
	// Taxable = 3,600,000
	// Bracket 1: first 800,000 at 0% = 0
	// Bracket 2: 800,001 - 3,000,000 = 2,200,000 at 15% = 330,000
	// Bracket 3: 3,000,001 - 3,600,000 = 600,000 at 18% = 108,000
	// Annual tax = 438,000
	// Monthly = 438,000 / 12 = 36,500
	result := Calculate(Input{
		BasicPay:    ngn(200000),
		OtherPay:    ngn(100000),
		GrossPay:    ngn(300000),
		PAYEEnabled: true,
	})
	expectedMonthly := ngn(36500)
	if result.PAYE != expectedMonthly {
		t.Errorf("expected monthly PAYE %d (NGN %d), got %d (NGN %d)",
			expectedMonthly, expectedMonthly/100, result.PAYE, result.PAYE/100)
	}
}

func TestCalculate_PAYEWithPensionDeduction(t *testing.T) {
	// NGN 300,000/month with pension enabled
	// Pension base = 200K + 50K + 30K = 280K
	// Employee pension = 280K * 8% = 22,400/month
	// Annual gross = 3,600,000
	// Annual pension deduction = 22,400 * 12 = 268,800
	// Taxable = 3,600,000 - 268,800 = 3,331,200
	// Bracket 1: 800,000 at 0% = 0
	// Bracket 2: 2,200,000 at 15% = 330,000
	// Bracket 3: 331,200 at 18% = 59,616
	// Annual tax = 389,616
	// Monthly = 389,616 / 12 = 32,468
	result := Calculate(Input{
		BasicPay:       ngn(200000),
		HousingPay:     ngn(50000),
		TransportPay:   ngn(30000),
		OtherPay:       ngn(20000),
		GrossPay:       ngn(300000),
		PensionEnabled: true,
		PAYEEnabled:    true,
	})
	expectedMonthly := ngn(32468)
	if result.PAYE != expectedMonthly {
		t.Errorf("expected monthly PAYE %d (NGN %d), got %d (NGN %d)",
			expectedMonthly, expectedMonthly/100, result.PAYE, result.PAYE/100)
	}
}

func TestCalculate_PAYEWithRentRelief(t *testing.T) {
	// NGN 500,000/month, rent = NGN 2,400,000/year
	// Rent relief = 20% * 2,400,000 = 480,000 (under 500K cap)
	// Annual gross = 6,000,000
	// Taxable = 6,000,000 - 480,000 = 5,520,000
	// Bracket 1: 800,000 at 0% = 0
	// Bracket 2: 2,200,000 at 15% = 330,000
	// Bracket 3: 2,520,000 at 18% = 453,600
	// Annual tax = 783,600
	// Monthly = 783,600 / 12 = 65,300
	result := Calculate(Input{
		BasicPay:       ngn(300000),
		OtherPay:       ngn(200000),
		GrossPay:       ngn(500000),
		AnnualRentPaid: ngn(2400000),
		PAYEEnabled:    true,
	})
	expectedMonthly := ngn(65300)
	if result.PAYE != expectedMonthly {
		t.Errorf("expected monthly PAYE %d (NGN %d), got %d (NGN %d)",
			expectedMonthly, expectedMonthly/100, result.PAYE, result.PAYE/100)
	}
	if result.RentRelief != ngn(480000) {
		t.Errorf("expected rent relief %d, got %d", ngn(480000), result.RentRelief)
	}
}

func TestCalculate_RentReliefCapped(t *testing.T) {
	// Rent = NGN 5,000,000/year → relief = 1,000,000 but capped at 500,000
	result := Calculate(Input{
		BasicPay:       ngn(500000),
		GrossPay:       ngn(500000),
		AnnualRentPaid: ngn(5000000),
		PAYEEnabled:    true,
	})
	if result.RentRelief != ngn(500000) {
		t.Errorf("rent relief should be capped at NGN 500,000 (%d kobo), got %d",
			ngn(500000), result.RentRelief)
	}
}

func TestCalculate_HighSalaryTopBracket(t *testing.T) {
	// NGN 5,000,000/month (60M/year) — hits the 25% bracket
	// Annual = 60,000,000
	// Bracket 1: 800,000 at 0% = 0
	// Bracket 2: 2,200,000 at 15% = 330,000
	// Bracket 3: 9,000,000 at 18% = 1,620,000
	// Bracket 4: 13,000,000 at 21% = 2,730,000
	// Bracket 5: 25,000,000 at 23% = 5,750,000
	// Bracket 6: 10,000,000 at 25% = 2,500,000
	// Annual tax = 12,930,000
	// Monthly = 12,930,000 / 12 = 1,077,500
	result := Calculate(Input{
		BasicPay:    ngn(3000000),
		OtherPay:    ngn(2000000),
		GrossPay:    ngn(5000000),
		PAYEEnabled: true,
	})
	expectedMonthly := ngn(1077500)
	if result.PAYE != expectedMonthly {
		t.Errorf("expected monthly PAYE %d (NGN %d), got %d (NGN %d)",
			expectedMonthly, expectedMonthly/100, result.PAYE, result.PAYE/100)
	}
}

func TestCalculate_FullStatutory(t *testing.T) {
	// NGN 300,000/month, all statutory enabled, rent = 1,200,000/year
	input := Input{
		BasicPay:       ngn(200000),
		HousingPay:     ngn(50000),
		TransportPay:   ngn(30000),
		OtherPay:       ngn(20000),
		GrossPay:       ngn(300000),
		AnnualRentPaid: ngn(1200000),
		PensionEnabled: true,
		NHFEnabled:     true,
		NSITFEnabled:   true,
		PAYEEnabled:    true,
	}
	result := Calculate(input)

	// Verify all components are non-zero
	if result.EmployeePension == 0 {
		t.Error("employee pension should be non-zero")
	}
	if result.EmployerPension == 0 {
		t.Error("employer pension should be non-zero")
	}
	if result.NHF == 0 {
		t.Error("NHF should be non-zero")
	}
	if result.NSITF == 0 {
		t.Error("NSITF should be non-zero")
	}
	if result.PAYE == 0 {
		t.Error("PAYE should be non-zero")
	}

	// Employee deductions = pension + NHF + PAYE (NOT NSITF)
	expectedEmployeeTotal := result.EmployeePension + result.NHF + result.PAYE
	if result.TotalEmployeeDeductions != expectedEmployeeTotal {
		t.Errorf("expected total employee deductions %d, got %d", expectedEmployeeTotal, result.TotalEmployeeDeductions)
	}

	// Employer costs = employer pension + NSITF (NOT PAYE/NHF)
	expectedEmployerTotal := result.EmployerPension + result.NSITF
	if result.TotalEmployerCosts != expectedEmployerTotal {
		t.Errorf("expected total employer costs %d, got %d", expectedEmployerTotal, result.TotalEmployerCosts)
	}
}

func TestApplyBrackets_ExactBoundaries(t *testing.T) {
	tests := []struct {
		name           string
		taxableIncome  int64
		expectedTax    int64
	}{
		{"at_zero", 0, 0},
		{"at_tax_free_ceiling", ngn(800000), 0},
		{"just_above_tax_free", ngn(800001), ngn(800001-800000) * 15 / 100},
		{"at_second_ceiling", ngn(3000000), ngn(2200000) * 15 / 100},
		{"at_third_ceiling", ngn(12000000),
			ngn(2200000)*15/100 + ngn(9000000)*18/100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyBrackets(tt.taxableIncome)
			if got != tt.expectedTax {
				t.Errorf("applyBrackets(%d) = %d, want %d", tt.taxableIncome, got, tt.expectedTax)
			}
		})
	}
}
