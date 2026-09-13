package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"sterile-packaging-release-control/backend/internal/dto"
	"sterile-packaging-release-control/backend/internal/model"
)

type LineRepository interface {
	List(context.Context, dto.PageQuery) ([]model.PackagingLine, int64, error)
	Find(context.Context, uint) (*model.PackagingLine, error)
	FindForUpdate(context.Context, uint) (*model.PackagingLine, error)
	FindByCode(context.Context, string) (*model.PackagingLine, error)
	Create(context.Context, *model.PackagingLine) error
	Save(context.Context, *model.PackagingLine) error
	CountActiveBatches(context.Context, uint) (int64, error)
}

type lineRepository struct{ db *gorm.DB }

func NewLineRepository(db *gorm.DB) LineRepository { return &lineRepository{db: db} }

func (r *lineRepository) List(ctx context.Context, query dto.PageQuery) ([]model.PackagingLine, int64, error) {
	query = query.Normalize()
	db := dbForContext(ctx, r.db).Model(&model.PackagingLine{})
	if search := strings.TrimSpace(query.Search); search != "" {
		like := "%" + search + "%"
		db = db.Where("code ILIKE ? OR name ILIKE ? OR team ILIKE ?", like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var lines []model.PackagingLine
	err := db.Preload("Batches", func(tx *gorm.DB) *gorm.DB {
		return tx.Order("created_at DESC").Limit(5)
	}).Order("active DESC, code ASC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&lines).Error
	return lines, total, err
}

func (r *lineRepository) Find(ctx context.Context, id uint) (*model.PackagingLine, error) {
	var line model.PackagingLine
	err := dbForContext(ctx, r.db).Preload("Batches").First(&line, id).Error
	return &line, err
}

func (r *lineRepository) FindForUpdate(ctx context.Context, id uint) (*model.PackagingLine, error) {
	var line model.PackagingLine
	err := dbForContext(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Batches").First(&line, id).Error
	return &line, err
}

func (r *lineRepository) FindByCode(ctx context.Context, code string) (*model.PackagingLine, error) {
	var line model.PackagingLine
	err := dbForContext(ctx, r.db).Where("code = ?", code).First(&line).Error
	return &line, err
}

func (r *lineRepository) Create(ctx context.Context, line *model.PackagingLine) error {
	return dbForContext(ctx, r.db).Create(line).Error
}

func (r *lineRepository) Save(ctx context.Context, line *model.PackagingLine) error {
	return dbForContext(ctx, r.db).Save(line).Error
}

func (r *lineRepository) CountActiveBatches(ctx context.Context, lineID uint) (int64, error) {
	var count int64
	err := dbForContext(ctx, r.db).Model(&model.ProductionBatch{}).
		Where("packaging_line_id = ? AND status IN ?", lineID, []string{"draft", "running", "hold", "rework"}).Count(&count).Error
	return count, err
}
