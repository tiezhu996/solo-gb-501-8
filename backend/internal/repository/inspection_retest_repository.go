package repository

import (
	"context"

	"gorm.io/gorm"

	"sterile-packaging-release-control/backend/internal/model"
)

type RetestRepository interface {
	Create(context.Context, *model.InspectionRetest) error
	ListBySample(context.Context, uint) ([]model.InspectionRetest, error)
	CountBySample(context.Context, uint) (int64, error)
}

type retestRepository struct{ db *gorm.DB }

func NewRetestRepository(db *gorm.DB) RetestRepository {
	return &retestRepository{db: db}
}

func (r *retestRepository) Create(ctx context.Context, record *model.InspectionRetest) error {
	return dbForContext(ctx, r.db).Create(record).Error
}

func (r *retestRepository) ListBySample(ctx context.Context, sampleID uint) ([]model.InspectionRetest, error) {
	var records []model.InspectionRetest
	err := dbForContext(ctx, r.db).
		Where("inspection_sample_id = ?", sampleID).
		Order("sequence ASC").
		Find(&records).Error
	return records, err
}

func (r *retestRepository) CountBySample(ctx context.Context, sampleID uint) (int64, error) {
	var count int64
	err := dbForContext(ctx, r.db).Model(&model.InspectionRetest{}).
		Where("inspection_sample_id = ?", sampleID).Count(&count).Error
	return count, err
}
