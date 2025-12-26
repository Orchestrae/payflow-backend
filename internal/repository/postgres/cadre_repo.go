package postgres

import (
	"context"
	"payflow/internal/domain"
	"payflow/internal/repository"

	"gorm.io/gorm"
)

type cadreRepository struct {
	db *gorm.DB
}

func NewCadreRepository(db *gorm.DB) repository.CadreRepository {
	return &cadreRepository{db: db}
}

func (r *cadreRepository) Create(ctx context.Context, cadre *domain.Cadre) error {
	return r.db.WithContext(ctx).Create(cadre).Error
}

func (r *cadreRepository) Update(ctx context.Context, cadre *domain.Cadre) error {
	return r.db.WithContext(ctx).Save(cadre).Error
}

func (r *cadreRepository) FindByID(ctx context.Context, id uint, businessID uint) (*domain.Cadre, error) {
	var cadre domain.Cadre
	err := r.db.WithContext(ctx).
		Preload("EarningComponents").
		Preload("DeductionRules").
		Where("id = ? AND business_id = ?", id, businessID).
		First(&cadre).Error
	if err != nil {
		return nil, err
	}
	return &cadre, nil
}

func (r *cadreRepository) Delete(ctx context.Context, id uint, businessID uint) error {
	result := r.db.WithContext(ctx).Where("business_id = ?", businessID).Delete(&domain.Cadre{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *cadreRepository) FindByBusinessID(ctx context.Context, businessID uint) ([]*domain.Cadre, error) {
	var cadres []domain.Cadre
	err := r.db.WithContext(ctx).
		Preload("EarningComponents").
		Preload("DeductionRules").
		Where("business_id = ?", businessID).
		Find(&cadres).Error
	if err != nil {
		return nil, err
	}
	domainCadres := make([]*domain.Cadre, len(cadres))
	for i, c := range cadres {
		domainCadres[i] = &c
	}
	return domainCadres, nil
}

func (r *cadreRepository) WithTx(tx repository.Transactioner) repository.CadreRepository {
	if txr, ok := tx.(*transactioner); ok {
		return &cadreRepository{db: txr.db}
	}
	return r
}
