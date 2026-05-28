package tax

// Input contains all data needed to compute Nigerian statutory deductions for one employee, one month.
// All monetary amounts are in kobo (smallest currency unit). 100 kobo = 1 NGN.
type Input struct {
	// Monthly earning components (kobo)
	BasicPay     int64
	HousingPay   int64
	TransportPay int64
	OtherPay     int64
	GrossPay     int64 // sum of all components

	// Employee-specific
	AnnualRentPaid int64 // for rent relief calculation (kobo, annual)

	// Business statutory configuration
	PensionEnabled bool
	NHFEnabled     bool
	NSITFEnabled   bool
	PAYEEnabled    bool
}

// Result contains all computed statutory amounts for one employee, one month (kobo).
type Result struct {
	// Employee deductions (reduce net pay)
	PAYE            int64 // monthly PAYE income tax
	EmployeePension int64 // 8% of pension base
	NHF             int64 // 2.5% of basic salary

	// Employer costs (do NOT reduce employee net pay)
	EmployerPension int64 // 10% of pension base
	NSITF           int64 // 1% of gross payroll

	// Breakdown for audit trail
	PensionBase             int64 // basic + housing + transport
	AnnualGross             int64
	AnnualTaxableIncome     int64
	RentRelief              int64
	TotalEmployeeDeductions int64
	TotalEmployerCosts      int64
}
