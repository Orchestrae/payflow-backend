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
	return t.db.WithContext(ctx).Begin()
}

func (t *transactioner) Commit(tx interface{}) error {
	if gormTx, ok := tx.(*gorm.DB); ok {
		return gormTx.Commit().Error
	}
	return nil
}

func (t *transactioner) Rollback(tx interface{}) error {
	if gormTx, ok := tx.(*gorm.DB); ok {
		return gormTx.Rollback().Error
	}
	return nil
}
