package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"sterile-packaging-release-control/backend/internal/dto"
	"sterile-packaging-release-control/backend/internal/model"
)

type InspectionFilter struct {
	dto.PageQuery
	Result  string
	BatchID uint
}

type InspectionRepository interface {
	List(context.Context, InspectionFilter) ([]model.InspectionSample, int64, error)
	Find(context.Context, uint) (*model.InspectionSample, error)
	FindForUpdate(context.Context, uint) (*model.InspectionSample, error)
	FindByCode(context.Context, string) (*model.InspectionSample, error)
	Create(context.Context, *model.InspectionSample) error
	Save(context.Context, *model.InspectionSample) error
	CountByResult(context.Context, uint, string) (int64, error)
	CountIncomplete(context.Context, uint) (int64, error)
}

type inspectionRepository struct{ db *gorm.DB }

func NewInspectionRepository(db *gorm.DB) InspectionRepository {
	return &inspectionRepository{db: db}
}

func (r *inspectionRepository) List(ctx context.Context, filter InspectionFilter) ([]model.InspectionSample, int64, error) {
	query := filter.PageQuery.Normalize()
	db := dbForContext(ctx, r.db).Model(&model.InspectionSample{})
	if filter.Result != "" {
		db = db.Where("result = ?", filter.Result)
	}
	if filter.BatchID > 0 {
		db = db.Where("production_batch_id = ?", filter.BatchID)
	}
	if search := strings.TrimSpace(query.Search); search != "" {
		like := "%" + search + "%"
		db = db.Where("sample_code ILIKE ? OR inspection_item ILIKE ? OR sampling_position ILIKE ?", like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var samples []model.InspectionSample
	err := db.Preload("ProductionBatch").Order("created_at DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&samples).Error
	return samples, total, err
}

func (r *inspectionRepository) Find(ctx context.Context, id uint) (*model.InspectionSample, error) {
	var sample model.InspectionSample
	err := dbForContext(ctx, r.db).Preload("ProductionBatch").First(&sample, id).Error
	return &sample, err
}

func (r *inspectionRepository) FindForUpdate(ctx context.Context, id uint) (*model.InspectionSample, error) {
	var sample model.InspectionSample
	err := dbForContext(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("ProductionBatch").First(&sample, id).Error
	return &sample, err
}

func (r *inspectionRepository) FindByCode(ctx context.Context, code string) (*model.InspectionSample, error) {
	var sample model.InspectionSample
	err := dbForContext(ctx, r.db).Where("sample_code = ?", code).First(&sample).Error
	return &sample, err
}

func (r *inspectionRepository) Create(ctx context.Context, sample *model.InspectionSample) error {
	return dbForContext(ctx, r.db).Create(sample).Error
}

func (r *inspectionRepository) Save(ctx context.Context, sample *model.InspectionSample) error {
	return dbForContext(ctx, r.db).Save(sample).Error
}

func (r *inspectionRepository) CountByResult(ctx context.Context, batchID uint, result string) (int64, error) {
	var count int64
	err := dbForContext(ctx, r.db).Model(&model.InspectionSample{}).
		Where("production_batch_id = ? AND result = ?", batchID, result).Count(&count).Error
	return count, err
}

func (r *inspectionRepository) CountIncomplete(ctx context.Context, batchID uint) (int64, error) {
	var count int64
	err := dbForContext(ctx, r.db).Model(&model.InspectionSample{}).
		Where("production_batch_id = ? AND (result = ? OR retest_status = ?)", batchID, "pending", "requested").Count(&count).Error
	return count, err
}
