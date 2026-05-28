package report

import (
	"fmt"
	"time"

	"payflow/internal/domain"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// GeneratePayslip generates a PDF payslip for a single employee entry.
func GeneratePayslip(entry *domain.PayrollRunEntry, business *domain.Business, period time.Time) ([]byte, error) {
	m := maroto.New()

	// Company header
	m.AddRows(
		row.New(12).Add(
			col.New(8).Add(
				text.New(business.Name, props.Text{Size: 14, Style: fontstyle.Bold}),
			),
			col.New(4).Add(
				text.New("PAYSLIP", props.Text{Size: 14, Style: fontstyle.Bold, Align: align.Right}),
			),
		),
		row.New(8).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("Period: %s", period.Format("January 2006")), props.Text{Size: 10}),
			),
		),
	)

	addSeparator(m)

	// Employee info
	emp := entry.Employee
	if emp != nil {
		m.AddRows(
			row.New(7).Add(
				col.New(6).Add(text.New(fmt.Sprintf("Employee: %s", emp.FullName), props.Text{Size: 10, Style: fontstyle.Bold})),
				col.New(6).Add(text.New(fmt.Sprintf("TIN: %s", safeString(emp.TIN)), props.Text{Size: 9, Align: align.Right})),
			),
			row.New(7).Add(
				col.New(6).Add(text.New(fmt.Sprintf("Bank: %s - %s", emp.BankName, emp.BankAccountNumber), props.Text{Size: 9})),
				col.New(6).Add(text.New(fmt.Sprintf("RSA PIN: %s", safeString(emp.PensionRSAPIN)), props.Text{Size: 9, Align: align.Right})),
			),
		)
	}

	addSeparator(m)

	// Earnings section
	addSectionHeader(m, "EARNINGS")
	earnings := findAllDetails(entry.Details, domain.DetailTypeEarning)
	for _, e := range earnings {
		addLineItem(m, e.Name, e.Amount)
	}
	addTotalLine(m, "GROSS PAY", entry.GrossPay)

	addSeparator(m)

	// Deductions section (custom + statutory)
	addSectionHeader(m, "DEDUCTIONS")
	customDeductions := findAllDetails(entry.Details, domain.DetailTypeDeduction)
	for _, d := range customDeductions {
		addLineItem(m, d.Name, d.Amount)
	}
	statutoryDeductions := findAllDetails(entry.Details, domain.DetailTypeStatutoryDeduction)
	for _, d := range statutoryDeductions {
		addLineItem(m, d.Name, d.Amount)
	}
	addTotalLine(m, "TOTAL DEDUCTIONS", entry.TotalDeductions)

	addSeparator(m)

	// Net pay
	m.AddRows(
		row.New(10).Add(
			col.New(8).Add(text.New("NET PAY", props.Text{Size: 12, Style: fontstyle.Bold})),
			col.New(4).Add(text.New(formatNGN(entry.NetPay), props.Text{Size: 12, Style: fontstyle.Bold, Align: align.Right})),
		),
	)

	addSeparator(m)

	// Employer costs section
	employerCosts := findAllDetails(entry.Details, domain.DetailTypeEmployerCost)
	if len(employerCosts) > 0 {
		addSectionHeader(m, "EMPLOYER COSTS")
		for _, c := range employerCosts {
			addLineItem(m, c.Name, c.Amount)
		}
		addTotalLine(m, "TOTAL COST TO COMPANY", entry.TotalCostToCompany)
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate payslip PDF: %w", err)
	}

	return doc.GetBytes(), nil
}

func addSectionHeader(m core.Maroto, title string) {
	m.AddRows(
		row.New(8).Add(
			col.New(12).Add(text.New(title, props.Text{Size: 10, Style: fontstyle.Bold, Top: 2})),
		),
	)
}

func addLineItem(m core.Maroto, name string, amountKobo int64) {
	m.AddRows(
		row.New(6).Add(
			col.New(8).Add(text.New("  "+name, props.Text{Size: 9})),
			col.New(4).Add(text.New(formatNGN(amountKobo), props.Text{Size: 9, Align: align.Right})),
		),
	)
}

func addTotalLine(m core.Maroto, label string, amountKobo int64) {
	m.AddRows(
		row.New(8).Add(
			col.New(8).Add(text.New(label, props.Text{Size: 10, Style: fontstyle.Bold, Top: 2})),
			col.New(4).Add(text.New(formatNGN(amountKobo), props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Right, Top: 2})),
		),
	)
}

func addSeparator(m core.Maroto) {
	m.AddRows(row.New(3))
}
