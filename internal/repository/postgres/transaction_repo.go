// internal/repository/postgres/transaction_repo.go
package postgres

import (
	"context"
	"payflow/internal/repository"

	"gorm.io/gorm"
)

type transactioner struct {
	db *gorm.DB
}

func NewTransactioner(db *gorm.DB) repository.Transactioner {
	return &transactioner{db: db}
}

func (t *transactioner) Begin(ctx context.Context) interface{} {
	tx := t.db.WithContext(ctx).Begin()
	// Return a new transactioner wrapping the transaction DB session.
	// This satisfies repository.Transactioner interface.
	return &transactioner{db: tx}
}

func (t *transactioner) Commit(tx interface{}) error {
	// Check if tx is *transactioner
	if txr, ok := tx.(*transactioner); ok {
		return txr.db.Commit().Error
	}
	// Fallback for raw *gorm.DB if ever used (legacy)
	if gormTx, ok := tx.(*gorm.DB); ok {
		return gormTx.Commit().Error
	}
	return nil
}

func (t *transactioner) Rollback(tx interface{}) error {
	if txr, ok := tx.(*transactioner); ok {
		return txr.db.Rollback().Error
	}
	if gormTx, ok := tx.(*gorm.DB); ok {
		return gormTx.Rollback().Error
	}
	return nil
}
