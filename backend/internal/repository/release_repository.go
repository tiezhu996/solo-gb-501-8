package repository

import (
	"context"

	"gorm.io/gorm"

	"sterile-packaging-release-control/backend/internal/dto"
	"sterile-packaging-release-control/backend/internal/model"
)

type ReleaseRepository interface {
	List(context.Context, dto.PageQuery, string) ([]model.ReleaseDecision, int64, error)
	Find(context.Context, uint) (*model.ReleaseDecision, error)
	LatestForBatch(context.Context, uint) (*model.ReleaseDecision, error)
	CreateWithBatch(context.Context, *model.ReleaseDecision, *model.ProductionBatch) error
}

type releaseRepository struct{ db *gorm.DB }

func NewReleaseRepository(db *gorm.DB) ReleaseRepository { return &releaseRepository{db: db} }

func (r *releaseRepository) List(ctx context.Context, query dto.PageQuery, decision string) ([]model.ReleaseDecision, int64, error) {
	query = query.Normalize()
	db := dbForContext(ctx, r.db).Model(&model.ReleaseDecision{})
	if decision != "" {
		db = db.Where("decision = ?", decision)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var decisions []model.ReleaseDecision
	err := db.Preload("ProductionBatch").Order("created_at DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&decisions).Error
	return decisions, total, err
}

func (r *releaseRepository) Find(ctx context.Context, id uint) (*model.ReleaseDecision, error) {
	var decision model.ReleaseDecision
	err := dbForContext(ctx, r.db).Preload("ProductionBatch").First(&decision, id).Error
	return &decision, err
}

func (r *releaseRepository) LatestForBatch(ctx context.Context, batchID uint) (*model.ReleaseDecision, error) {
	var decision model.ReleaseDecision
	err := dbForContext(ctx, r.db).Where("production_batch_id = ?", batchID).Order("created_at DESC").First(&decision).Error
	return &decision, err
}

func (r *releaseRepository) CreateWithBatch(ctx context.Context, decision *model.ReleaseDecision, batch *model.ProductionBatch) error {
	db := dbForContext(ctx, r.db)
	if err := db.Create(decision).Error; err != nil {
		return err
	}
	return db.Save(batch).Error
}
