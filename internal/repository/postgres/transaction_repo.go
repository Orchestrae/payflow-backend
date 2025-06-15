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

func (t *transactioner) Begin(ctx context.Context) *gorm.DB {
	return t.db.WithContext(ctx).Begin()
}

func (t *transactioner) Commit(tx *gorm.DB) error {
	return tx.Commit().Error
}

func (t *transactioner) Rollback(tx *gorm.DB) {
	tx.Rollback()
}
