package domain

// Double-entry accounting ledger.
// Every financial operation creates 2 entries: one debit and one credit.
// SUM(debits) == SUM(credits) must always hold.

// AccountType represents the type of ledger account.
type AccountType string

const (
	AccountWallet   AccountType = "wallet"    // Business wallet (asset)
	AccountExternal AccountType = "external"  // External bank/provider (liability)
	AccountRevenue  AccountType = "revenue"   // PayFlow fee revenue
	AccountPayable  AccountType = "payable"   // Amount owed to employee
)

// EntryType represents debit or credit.
type EntryType string

const (
	EntryDebit  EntryType = "debit"
	EntryCredit EntryType = "credit"
)

// LedgerEntry represents a single entry in the double-entry ledger.
type LedgerEntry struct {
	Model
	BusinessID    uint        `gorm:"index" json:"business_id"`
	TransactionID string      `gorm:"size:100;index" json:"transaction_id"` // Groups debit+credit pair
	AccountType   AccountType `gorm:"type:varchar(20)" json:"account_type"`
	EntryType     EntryType   `gorm:"type:varchar(10)" json:"entry_type"`
	Amount        int64       `json:"amount"`           // Always positive (direction from EntryType)
	Currency      string      `gorm:"size:10;default:'NGN'" json:"currency"`
	Description   string      `gorm:"size:500" json:"description"`
	Reference     string      `gorm:"size:100;index" json:"reference"` // Links to transfer/deposit ref
	BalanceAfter  int64       `json:"balance_after"`    // Running balance after this entry
}
