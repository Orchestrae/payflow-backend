package postgres

import (
	"context"
	"log/slog"
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
		if err == gorm.ErrRecordNotFound {
			slog.Error("Cadre not found", "id", id, "businessID", businessID)
			return nil, domain.ErrNotFound
		}
		slog.Error("Error retrieving cadre by ID", "error", err, "id", id, "businessID", businessID)
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

// FindCadreByBusinessID finds the first cadre by business ID.
// Returns gorm.ErrRecordNotFound if no cadre exists for the business.
func (r *cadreRepository) FindCadreByBusinessID(ctx context.Context, businessID uint) (*domain.Cadre, error) {
	var cadre domain.Cadre
	err := r.db.WithContext(ctx).
		Where("business_id = ?", businessID).
		Preload("EarningComponents").
		Preload("DeductionRules").
		First(&cadre).Error
	if err != nil {
		return nil, err
	}
	return &cadre, nil
}

// verify that cadre name is unique (cadre.Name) per business
// verify that cadre.EarningComponents[].Name is unique per cadre within the business
func (r *cadreRepository) IsCadreNameUnique(
	ctx context.Context,
	cadre domain.Cadre,
) (bool, error) {

	var count int64

	q := r.db.WithContext(ctx).
		Model(&domain.Cadre{}).
		Where("business_id = ? AND name = ?", cadre.BusinessID, cadre.Name)

	if cadre.ID != 0 {
		q = q.Where("id != ?", cadre.ID)
	}

	if err := q.Count(&count).Error; err != nil {
		return false, err
	}

	return count == 0, nil
}

func (r *cadreRepository) WithTx(tx repository.Transactioner) repository.CadreRepository {
	if txr, ok := tx.(*transactioner); ok {
		return &cadreRepository{db: txr.db}
	}
	return r
}
