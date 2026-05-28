package ghana

// Input contains all data needed to compute Ghanaian statutory deductions for one employee, one month.
// All monetary amounts are in pesewas (smallest currency unit). 100 pesewas = 1 GHS.
type Input struct {
	// Monthly earning components (pesewas)
	BasicPay     int64
	HousingPay   int64
	TransportPay int64
	OtherPay     int64
	GrossPay     int64 // sum of all components

	// Business statutory configuration
	PensionEnabled bool
	PAYEEnabled    bool
}

// Result contains all computed Ghanaian statutory amounts for one employee, one month (pesewas).
type Result struct {
	// Employee deductions (reduce net pay)
	PAYE             int64 // monthly PAYE income tax
	EmployeePension  int64 // 5.5% of basic (SSNIT employee)
	Tier2Pension     int64 // 5% of basic (occupational pension)

	// Employer costs (do NOT reduce employee net pay)
	EmployerPension  int64 // 13% of basic (SSNIT employer)

	// Breakdown
	PensionBase             int64 // basic salary (SSNIT base)
	AnnualGross             int64
	AnnualTaxableIncome     int64
	TotalEmployeeDeductions int64
	TotalEmployerCosts      int64
}
