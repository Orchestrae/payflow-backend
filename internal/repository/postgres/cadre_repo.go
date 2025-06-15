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

func (r *cadreRepository) FindByID(ctx context.Context, id uint) (*domain.Cadre, error) {
	var cadre domain.Cadre
	err := r.db.WithContext(ctx).
		Preload("EarningComponents").
		Preload("DeductionRules").
		First(&cadre, id).Error
	if err != nil {
		return nil, err
	}
	return &cadre, nil
}

func (r *cadreRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&domain.Cadre{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *cadreRepository) FindAllByBusinessID(ctx context.Context, businessID uint) ([]domain.Cadre, error) {
	var cadres []domain.Cadre
	err := r.db.WithContext(ctx).
		Preload("EarningComponents").
		Preload("DeductionRules").
		Where("business_id = ?", businessID).
		Find(&cadres).Error
	if err != nil {
		return nil, err
	}
	return cadres, nil
}

func (r *cadreRepository) WithTx(tx *gorm.DB) repository.CadreRepository {
	return &cadreRepository{db: tx}
}
