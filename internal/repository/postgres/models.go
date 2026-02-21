// internal/repository/postgres/models.go
package postgres

import (
	"payflow/internal/domain"
	"time"

	"gorm.io/gorm"
)

// This file contains GORM-specific models.
// They are used to interact with the database and can contain DB tags.

type Business struct {
	gorm.Model
	AdminID                 uint
	Name                    string
	RCNumber                *string
	IncorporationDate       *time.Time
	DirectorBVN             *string
	VFDAccountNumber        *string
	VFDAccountName          *string
	PayrollRequiresApproval bool   `gorm:"default:true"`
	PayrollAutoProcess      bool   `gorm:"default:false"`
	Users                   []User `gorm:"foreignKey:BusinessID"`
}

type User struct {
	gorm.Model
	BusinessID   uint
	Email        string `gorm:"uniqueIndex"`
	PasswordHash string
	Role         domain.UserRole
	IsVerified   bool
}

// ToDomain converts a postgres.User model to a domain.User model.
func (u *User) ToDomain() *domain.User {
	return &domain.User{
		Model: domain.Model{
			ID:        u.Model.ID,
			CreatedAt: u.Model.CreatedAt,
			UpdatedAt: u.Model.UpdatedAt,
			DeletedAt: u.Model.DeletedAt,
		},
		BusinessID:   u.BusinessID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
		IsVerified:   u.IsVerified,
	}
}

func UserFromDomain(u *domain.User) *User {
	return &User{
		Model: gorm.Model{
			ID:        u.Model.ID,
			CreatedAt: u.Model.CreatedAt,
			UpdatedAt: u.Model.UpdatedAt,
			DeletedAt: u.Model.DeletedAt,
		},
		BusinessID:   u.BusinessID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
		IsVerified:   u.IsVerified,
	}
}

func (b *Business) ToDomain() *domain.Business {
	return &domain.Business{
		Model: domain.Model{
			ID:        b.Model.ID,
			CreatedAt: b.Model.CreatedAt,
			UpdatedAt: b.Model.UpdatedAt,
			DeletedAt: b.Model.DeletedAt,
		},
		AdminID:                 b.AdminID,
		Name:                    b.Name,
		RCNumber:                b.RCNumber,
		IncorporationDate:       b.IncorporationDate,
		DirectorBVN:             b.DirectorBVN,
		VFDAccountNumber:        b.VFDAccountNumber,
		VFDAccountName:          b.VFDAccountName,
		PayrollRequiresApproval: b.PayrollRequiresApproval,
		PayrollAutoProcess:      b.PayrollAutoProcess,
	}
}

func BusinessFromDomain(b *domain.Business) *Business {
	return &Business{
		Model: gorm.Model{
			ID:        b.Model.ID,
			CreatedAt: b.Model.CreatedAt,
			UpdatedAt: b.Model.UpdatedAt,
			DeletedAt: b.Model.DeletedAt,
		},
		AdminID:                 b.AdminID,
		Name:                    b.Name,
		RCNumber:                b.RCNumber,
		IncorporationDate:       b.IncorporationDate,
		DirectorBVN:             b.DirectorBVN,
		VFDAccountNumber:        b.VFDAccountNumber,
		VFDAccountName:          b.VFDAccountName,
		PayrollRequiresApproval: b.PayrollRequiresApproval,
		PayrollAutoProcess:      b.PayrollAutoProcess,
	}
}

// --- DeductionRule ---
type DeductionRule struct {
	gorm.Model
	BusinessID       uint   `gorm:"uniqueIndex:idx_business_deduction_name;not null"`
	Name             string `gorm:"uniqueIndex:idx_business_deduction_name;not null"`
	Type             domain.DeductionRuleType
	Value            float64
	CalculationBasis domain.CalculationBasis
}

func (dr *DeductionRule) ToDomain() *domain.DeductionRule {
	return &domain.DeductionRule{
		Model: domain.Model{
			ID:        dr.Model.ID,
			CreatedAt: dr.Model.CreatedAt,
			UpdatedAt: dr.Model.UpdatedAt,
			DeletedAt: dr.Model.DeletedAt,
		},
		BusinessID:       dr.BusinessID,
		Name:             dr.Name,
		Type:             dr.Type,
		Value:            dr.Value,
		CalculationBasis: dr.CalculationBasis,
	}
}

func DeductionRuleFromDomain(dr *domain.DeductionRule) *DeductionRule {
	return &DeductionRule{
		Model: gorm.Model{
			ID:        dr.Model.ID,
			CreatedAt: dr.Model.CreatedAt,
			UpdatedAt: dr.Model.UpdatedAt,
			DeletedAt: dr.Model.DeletedAt,
		},
		BusinessID:       dr.BusinessID,
		Name:             dr.Name,
		Type:             dr.Type,
		Value:            dr.Value,
		CalculationBasis: dr.CalculationBasis,
	}
}

// --- Cadre ---
type Cadre struct {
	gorm.Model
	BusinessID        uint   `gorm:"uniqueIndex:idx_business_cadre_name;not null"`
	Name              string `gorm:"uniqueIndex:idx_business_cadre_name;not null"`
	EarningComponents []EarningComponent
	DeductionRules    []DeductionRule `gorm:"many2many:cadre_deduction_rules;"`
}

func (c *Cadre) ToDomain() *domain.Cadre {
	earningComponents := make([]domain.EarningComponent, len(c.EarningComponents))
	for i, ec := range c.EarningComponents {
		earningComponents[i] = *ec.ToDomain()
	}

	deductionRules := make([]domain.DeductionRule, len(c.DeductionRules))
	for i, dr := range c.DeductionRules {
		deductionRules[i] = *dr.ToDomain()
	}

	return &domain.Cadre{
		Model: domain.Model{
			ID:        c.Model.ID,
			CreatedAt: c.Model.CreatedAt,
			UpdatedAt: c.Model.UpdatedAt,
			DeletedAt: c.Model.DeletedAt,
		},
		BusinessID:        c.BusinessID,
		Name:              c.Name,
		EarningComponents: earningComponents,
		DeductionRules:    deductionRules,
	}
}

func CadreFromDomain(c *domain.Cadre) *Cadre {
	earningComponents := make([]EarningComponent, len(c.EarningComponents))
	for i, ec := range c.EarningComponents {
		earningComponents[i] = *EarningComponentFromDomain(&ec)
	}

	deductionRules := make([]DeductionRule, len(c.DeductionRules))
	for i, dr := range c.DeductionRules {
		deductionRules[i] = *DeductionRuleFromDomain(&dr)
	}

	return &Cadre{
		Model: gorm.Model{
			ID:        c.Model.ID,
			CreatedAt: c.Model.CreatedAt,
			UpdatedAt: c.Model.UpdatedAt,
			DeletedAt: c.Model.DeletedAt,
		},
		BusinessID:        c.BusinessID,
		Name:              c.Name,
		EarningComponents: earningComponents,
		DeductionRules:    deductionRules,
	}
}

// --- EarningComponent ---
type EarningComponent struct {
	gorm.Model
	CadreID uint
	Name    string
	Amount  int64
}

func (ec *EarningComponent) ToDomain() *domain.EarningComponent {
	return &domain.EarningComponent{
		Model: domain.Model{
			ID:        ec.Model.ID,
			CreatedAt: ec.Model.CreatedAt,
			UpdatedAt: ec.Model.UpdatedAt,
			DeletedAt: ec.Model.DeletedAt,
		},
		CadreID: ec.CadreID,
		Name:    ec.Name,
		Amount:  ec.Amount,
	}
}

func EarningComponentFromDomain(ec *domain.EarningComponent) *EarningComponent {
	return &EarningComponent{
		Model: gorm.Model{
			ID:        ec.Model.ID,
			CreatedAt: ec.Model.CreatedAt,
			UpdatedAt: ec.Model.UpdatedAt,
			DeletedAt: ec.Model.DeletedAt,
		},
		CadreID: ec.CadreID,
		Name:    ec.Name,
		Amount:  ec.Amount,
	}
}

// --- Employee ---
type Employee struct {
	gorm.Model
	BusinessID        uint   `gorm:"uniqueIndex:idx_business_employee_email;not null"`
	CadreID           uint   `gorm:"not null"`
	FullName          string `gorm:"not null"`
	Email             string `gorm:"uniqueIndex:idx_business_employee_email;not null"`
	BankName          string
	BankCode          string `gorm:"size:10"`
	BankAccountNumber string
	IsActive          bool  `gorm:"default:true"`
	Cadre             Cadre `gorm:"foreignKey:CadreID"`
}

func (e *Employee) ToDomain() *domain.Employee {
	domCadre := e.Cadre.ToDomain()
	return &domain.Employee{
		Model: domain.Model{
			ID:        e.Model.ID,
			CreatedAt: e.Model.CreatedAt,
			UpdatedAt: e.Model.UpdatedAt,
			DeletedAt: e.Model.DeletedAt,
		},
		BusinessID:        e.BusinessID,
		CadreID:           e.CadreID,
		FullName:          e.FullName,
		Email:             e.Email,
		BankName:          e.BankName,
		BankCode:          e.BankCode,
		BankAccountNumber: e.BankAccountNumber,
		IsActive:          e.IsActive,
		Cadre:             domCadre,
	}
}

func EmployeeFromDomain(e *domain.Employee) *Employee {
	return &Employee{
		Model: gorm.Model{
			ID:        e.Model.ID,
			CreatedAt: e.Model.CreatedAt,
			UpdatedAt: e.Model.UpdatedAt,
			DeletedAt: e.Model.DeletedAt,
		},
		BusinessID:        e.BusinessID,
		CadreID:           e.CadreID,
		FullName:          e.FullName,
		Email:             e.Email,
		BankName:          e.BankName,
		BankCode:          e.BankCode,
		BankAccountNumber: e.BankAccountNumber,
		IsActive:          e.IsActive,
	}
}

// --- PayrollRun ---
type PayrollRun struct {
	gorm.Model
	BusinessID       uint
	Period           time.Time
	Status           domain.PayrollStatus `gorm:"type:payroll_status"`
	TotalGrossPay    int64
	TotalDeductions  int64
	TotalNetPay      int64
	ScheduledFor     time.Time
	ProcessedAt      *time.Time
	PaymentReference string
	RejectionReason  string
	Entries          []PayrollRunEntry `gorm:"foreignKey:PayrollRunID"`
}

func (pr *PayrollRun) ToDomain() *domain.PayrollRun {
	entries := make([]domain.PayrollRunEntry, len(pr.Entries))
	for i, e := range pr.Entries {
		entries[i] = *e.ToDomain()
	}

	return &domain.PayrollRun{
		Model: domain.Model{
			ID:        pr.Model.ID,
			CreatedAt: pr.Model.CreatedAt,
			UpdatedAt: pr.Model.UpdatedAt,
			DeletedAt: pr.Model.DeletedAt,
		},
		BusinessID:       pr.BusinessID,
		Period:           pr.Period,
		Status:           pr.Status,
		TotalGrossPay:    pr.TotalGrossPay,
		TotalDeductions:  pr.TotalDeductions,
		TotalNetPay:      pr.TotalNetPay,
		ScheduledFor:     pr.ScheduledFor,
		ProcessedAt:      pr.ProcessedAt,
		PaymentReference: pr.PaymentReference,
		RejectionReason:  pr.RejectionReason,
		Entries:          entries,
	}
}

func PayrollRunFromDomain(pr *domain.PayrollRun) *PayrollRun {
	entries := make([]PayrollRunEntry, len(pr.Entries))
	for i, e := range pr.Entries {
		entries[i] = *PayrollRunEntryFromDomain(&e)
	}

	return &PayrollRun{
		Model: gorm.Model{
			ID:        pr.Model.ID,
			CreatedAt: pr.Model.CreatedAt,
			UpdatedAt: pr.Model.UpdatedAt,
			DeletedAt: pr.Model.DeletedAt,
		},
		BusinessID:       pr.BusinessID,
		Period:           pr.Period,
		Status:           pr.Status,
		TotalGrossPay:    pr.TotalGrossPay,
		TotalDeductions:  pr.TotalDeductions,
		TotalNetPay:      pr.TotalNetPay,
		ScheduledFor:     pr.ScheduledFor,
		ProcessedAt:      pr.ProcessedAt,
		PaymentReference: pr.PaymentReference,
		RejectionReason:  pr.RejectionReason,
		Entries:          entries,
	}
}

// --- PayrollRunEntry ---
type PayrollRunEntry struct {
	gorm.Model
	PayrollRunID    uint
	EmployeeID      uint
	GrossPay        int64
	TotalDeductions int64
	Bonuses         int64
	NetPay          int64
	Details         []PayrollRunEntryDetail `gorm:"foreignKey:PayrollRunEntryID"`
}

func (e *PayrollRunEntry) ToDomain() *domain.PayrollRunEntry {
	details := make([]domain.PayrollRunEntryDetail, len(e.Details))
	for i, d := range e.Details {
		details[i] = *d.ToDomain()
	}

	return &domain.PayrollRunEntry{
		Model: domain.Model{
			ID:        e.Model.ID,
			CreatedAt: e.Model.CreatedAt,
			UpdatedAt: e.Model.UpdatedAt,
			DeletedAt: e.Model.DeletedAt,
		},
		PayrollRunID:    e.PayrollRunID,
		EmployeeID:      e.EmployeeID,
		GrossPay:        e.GrossPay,
		TotalDeductions: e.TotalDeductions,
		Bonuses:         e.Bonuses,
		NetPay:          e.NetPay,
		Details:         details,
	}
}

func PayrollRunEntryFromDomain(e *domain.PayrollRunEntry) *PayrollRunEntry {
	details := make([]PayrollRunEntryDetail, len(e.Details))
	for i, d := range e.Details {
		details[i] = *PayrollRunEntryDetailFromDomain(&d)
	}

	return &PayrollRunEntry{
		Model: gorm.Model{
			ID:        e.Model.ID,
			CreatedAt: e.Model.CreatedAt,
			UpdatedAt: e.Model.UpdatedAt,
			DeletedAt: e.Model.DeletedAt,
		},
		PayrollRunID:    e.PayrollRunID,
		EmployeeID:      e.EmployeeID,
		GrossPay:        e.GrossPay,
		TotalDeductions: e.TotalDeductions,
		Bonuses:         e.Bonuses,
		NetPay:          e.NetPay,
		Details:         details,
	}
}

// --- PayrollRunEntryDetail ---
type PayrollRunEntryDetail struct {
	gorm.Model
	PayrollRunEntryID uint
	Type              domain.PayrollEntryDetailType `gorm:"type:payroll_entry_detail_type"`
	Name              string
	Amount            int64
	Description       string
}

func (d *PayrollRunEntryDetail) ToDomain() *domain.PayrollRunEntryDetail {
	return &domain.PayrollRunEntryDetail{
		Model: domain.Model{
			ID:        d.Model.ID,
			CreatedAt: d.Model.CreatedAt,
			UpdatedAt: d.Model.UpdatedAt,
			DeletedAt: d.Model.DeletedAt,
		},
		PayrollRunEntryID: d.PayrollRunEntryID,
		Type:              d.Type,
		Name:              d.Name,
		Amount:            d.Amount,
		Description:       d.Description,
	}
}

func PayrollRunEntryDetailFromDomain(d *domain.PayrollRunEntryDetail) *PayrollRunEntryDetail {
	return &PayrollRunEntryDetail{
		Model: gorm.Model{
			ID:        d.Model.ID,
			CreatedAt: d.Model.CreatedAt,
			UpdatedAt: d.Model.UpdatedAt,
			DeletedAt: d.Model.DeletedAt,
		},
		PayrollRunEntryID: d.PayrollRunEntryID,
		Type:              d.Type,
		Name:              d.Name,
		Amount:            d.Amount,
		Description:       d.Description,
	}
}
