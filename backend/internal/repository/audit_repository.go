package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"sterile-packaging-release-control/backend/internal/dto"
	"sterile-packaging-release-control/backend/internal/model"
)

type AuditFilter struct {
	dto.PageQuery
	EntityType string
	ActorID    uint
}

type AuditRepository interface {
	Create(context.Context, *model.AuditLog) error
	List(context.Context, AuditFilter) ([]model.AuditLog, int64, error)
	FindByRequestID(context.Context, string) ([]model.AuditLog, error)
}

type auditRepository struct{ db *gorm.DB }

func NewAuditRepository(db *gorm.DB) AuditRepository { return &auditRepository{db: db} }

func (r *auditRepository) Create(ctx context.Context, log *model.AuditLog) error {
	return dbForContext(ctx, r.db).Create(log).Error
}

func (r *auditRepository) List(ctx context.Context, filter AuditFilter) ([]model.AuditLog, int64, error) {
	query := filter.PageQuery.Normalize()
	db := dbForContext(ctx, r.db).Model(&model.AuditLog{})
	if filter.EntityType != "" {
		db = db.Where("entity_type = ?", filter.EntityType)
	}
	if filter.ActorID > 0 {
		db = db.Where("actor_id = ?", filter.ActorID)
	}
	if search := strings.TrimSpace(query.Search); search != "" {
		like := "%" + search + "%"
		db = db.Where("action ILIKE ? OR actor_name ILIKE ? OR request_id ILIKE ?", like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var logs []model.AuditLog
	err := db.Order("created_at DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&logs).Error
	return logs, total, err
}

func (r *auditRepository) FindByRequestID(ctx context.Context, requestID string) ([]model.AuditLog, error) {
	var logs []model.AuditLog
	err := dbForContext(ctx, r.db).Where("request_id = ?", requestID).Order("created_at ASC").Find(&logs).Error
	return logs, err
}
