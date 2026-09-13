package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"sterile-packaging-release-control/backend/internal/dto"
	"sterile-packaging-release-control/backend/internal/model"
)

type BatchFilter struct {
	dto.PageQuery
	Status string
	LineID uint
}

type BatchRepository interface {
	List(context.Context, BatchFilter) ([]model.ProductionBatch, int64, error)
	Find(context.Context, uint) (*model.ProductionBatch, error)
	FindForUpdate(context.Context, uint) (*model.ProductionBatch, error)
	FindByNumber(context.Context, string) (*model.ProductionBatch, error)
	Create(context.Context, *model.ProductionBatch) error
	Save(context.Context, *model.ProductionBatch) error
	Overview(context.Context) (*dto.QualityOverview, error)
}

type batchRepository struct{ db *gorm.DB }

func NewBatchRepository(db *gorm.DB) BatchRepository { return &batchRepository{db: db} }

func (r *batchRepository) List(ctx context.Context, filter BatchFilter) ([]model.ProductionBatch, int64, error) {
	query := filter.PageQuery.Normalize()
	db := dbForContext(ctx, r.db).Model(&model.ProductionBatch{})
	if filter.Status != "" {
		db = db.Where("status = ?", filter.Status)
	}
	if filter.LineID > 0 {
		db = db.Where("packaging_line_id = ?", filter.LineID)
	}
	if search := strings.TrimSpace(query.Search); search != "" {
		like := "%" + search + "%"
		db = db.Where("batch_no ILIKE ? OR specification ILIKE ? OR responsible_team ILIKE ?", like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var batches []model.ProductionBatch
	err := db.Preload("PackagingLine").Preload("Inspections").Preload("Decisions").
		Order("created_at DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&batches).Error
	return batches, total, err
}

func (r *batchRepository) Find(ctx context.Context, id uint) (*model.ProductionBatch, error) {
	var batch model.ProductionBatch
	err := dbForContext(ctx, r.db).Preload("PackagingLine").Preload("Inspections").Preload("Decisions", func(tx *gorm.DB) *gorm.DB {
		return tx.Order("created_at DESC")
	}).First(&batch, id).Error
	return &batch, err
}

func (r *batchRepository) FindForUpdate(ctx context.Context, id uint) (*model.ProductionBatch, error) {
	var batch model.ProductionBatch
	err := dbForContext(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("PackagingLine").Preload("Inspections").Preload("Decisions").First(&batch, id).Error
	return &batch, err
}

func (r *batchRepository) FindByNumber(ctx context.Context, number string) (*model.ProductionBatch, error) {
	var batch model.ProductionBatch
	err := dbForContext(ctx, r.db).Where("batch_no = ?", number).First(&batch).Error
	return &batch, err
}

func (r *batchRepository) Create(ctx context.Context, batch *model.ProductionBatch) error {
	return dbForContext(ctx, r.db).Create(batch).Error
}

func (r *batchRepository) Save(ctx context.Context, batch *model.ProductionBatch) error {
	return dbForContext(ctx, r.db).Save(batch).Error
}

func (r *batchRepository) Overview(ctx context.Context) (*dto.QualityOverview, error) {
	db := dbForContext(ctx, r.db)
	result := &dto.QualityOverview{GeneratedAt: time.Now(), BatchStatuses: make([]dto.StatusCount, 0), Risks: make([]dto.QualityRisk, 0)}
	if err := db.Model(&model.PackagingLine{}).Count(&result.TotalLines).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&model.PackagingLine{}).Where("active = ? AND equipment_status IN ?", true, []string{"ready", "running"}).Count(&result.AvailableLines).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&model.PackagingLine{}).Where("equipment_status IN ?", []string{"maintenance", "fault"}).Count(&result.AttentionLines).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&model.ProductionBatch{}).Count(&result.TotalBatches).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&model.ProductionBatch{}).Select("status, count(*) AS count").Group("status").Order("status").Scan(&result.BatchStatuses).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&model.ProductionBatch{}).Where("status IN ?", []string{"running", "hold", "rework"}).Count(&result.AwaitingApproval).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&model.InspectionSample{}).Count(&result.TotalInspections).Error; err != nil {
		return nil, err
	}
	db.Model(&model.InspectionSample{}).Where("result = ?", "pass").Count(&result.PassedInspections)
	db.Model(&model.InspectionSample{}).Where("result = ?", "fail").Count(&result.FailedInspections)
	db.Model(&model.InspectionSample{}).Where("result = ?", "pending").Count(&result.PendingInspections)
	db.Model(&model.InspectionSample{}).Where("retest_status = ?", "requested").Count(&result.RequestedRetests)
	today := time.Now().Truncate(24 * time.Hour)
	db.Model(&model.ReleaseDecision{}).Where("created_at >= ?", today).Count(&result.DecisionsToday)
	db.Model(&model.AuditLog{}).Where("created_at >= ?", today).Count(&result.AuditEventsToday)
	completed := result.PassedInspections + result.FailedInspections
	if completed > 0 {
		result.FirstPassYield = float64(result.PassedInspections) / float64(completed) * 100
	}
	var risky []model.ProductionBatch
	if err := db.Preload("Inspections").Where("status IN ?", []string{"hold", "rework", "running"}).Order("updated_at DESC").Limit(10).Find(&risky).Error; err != nil {
		return nil, err
	}
	for _, batch := range risky {
		var failed, pending, retest int64
		for _, sample := range batch.Inspections {
			if sample.Result == "fail" {
				failed++
			}
			if sample.Result == "pending" {
				pending++
			}
			if sample.RetestStatus == "requested" {
				retest++
			}
		}
		if batch.Status == "running" && failed == 0 && pending == 0 && retest == 0 {
			continue
		}
		result.Risks = append(result.Risks, dto.QualityRisk{BatchID: batch.ID, BatchNo: batch.BatchNo, Status: string(batch.Status), Failed: failed, Pending: pending, Retest: retest, HoldReason: batch.HoldReason, LastModified: batch.UpdatedAt.Format(time.RFC3339)})
	}
	return result, nil
}
