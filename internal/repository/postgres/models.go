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
	AdminID uint
	Name    string
	Users   []User `gorm:"foreignKey:BusinessID"`
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
		ID:           u.ID,
		BusinessID:   u.BusinessID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
		IsVerified:   u.IsVerified,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

// fromDomain converts a domain.User model to a postgres.User model.
func UserFromDomain(u *domain.User) *User {
	return &User{
		Model:        gorm.Model{ID: u.ID, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt},
		BusinessID:   u.BusinessID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
		IsVerified:   u.IsVerified,
	}
}

func (b *Business) ToDomain() *domain.Business {
	return &domain.Business{
		ID:        b.ID,
		AdminID:   b.AdminID,
		Name:      b.Name,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
}
func DeductionRuleFromDomain(dr *domain.DeductionRule) *DeductionRule {
	/* ... */
	return &DeductionRule{}
}

func BusinessFromDomain(b *domain.Business) *Business {
	return &Business{
		Model:   gorm.Model{ID: b.ID, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt},
		AdminID: b.AdminID,
		Name:    b.Name,
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

// --- EarningComponent ---
type EarningComponent struct {
	gorm.Model
	CadreID uint
	Name    string
	Amount  int64
}

// --- Employee ---
type Employee struct {
	gorm.Model
	BusinessID        uint   `gorm:"uniqueIndex:idx_business_employee_email;not null"`
	CadreID           uint   `gorm:"not null"`
	FullName          string `gorm:"not null"`
	Email             string `gorm:"uniqueIndex:idx_business_employee_email;not null"`
	BankName          string
	BankAccountNumber string
	IsActive          bool  `gorm:"default:true"`
	Cadre             Cadre `gorm:"foreignKey:CadreID"`
}

func (e *Employee) ToDomain() *domain.Employee {
	domCadre := e.Cadre.ToDomain()
	return &domain.Employee{
		ID:                e.ID,
		BusinessID:        e.BusinessID,
		CadreID:           e.CadreID,
		FullName:          e.FullName,
		Email:             e.Email,
		BankName:          e.BankName,
		BankAccountNumber: e.BankAccountNumber,
		IsActive:          e.IsActive,
		CreatedAt:         e.CreatedAt,
		UpdatedAt:         e.UpdatedAt,
		Cadre:             domCadre,
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
	return &domain.DeductionRule{}
}

func (c *Cadre) ToDomain() *domain.Cadre {
	return &domain.Cadre{}
}

func CadreFromDomain(c *domain.Cadre) *Cadre {
	return &Cadre{}
}

func (ec *EarningComponent) ToDomain() *domain.EarningComponent {
	return &domain.EarningComponent{}
}

func EarningComponentFromDomain(ec *domain.EarningComponent) *EarningComponent {
	return &EarningComponent{}
}

func EmployeeFromDomain(e *domain.Employee) *Employee {
	return &Employee{}
}

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
	return &domain.PayrollRun{}
}

func PayrollRunFromDomain(pr *domain.PayrollRun) *PayrollRun {
	return &PayrollRun{}
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

// --- PayrollRunEntryDetail ---
type PayrollRunEntryDetail struct {
	gorm.Model
	PayrollRunEntryID uint
	Type              domain.PayrollEntryDetailType `gorm:"type:payroll_entry_detail_type"`
	Name              string
	Amount            int64
}

// ... Mappers ...
