package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"sterile-packaging-release-control/backend/internal/constants"
	"sterile-packaging-release-control/backend/internal/dto"
	"sterile-packaging-release-control/backend/internal/model"
	"sterile-packaging-release-control/backend/internal/repository"
	"sterile-packaging-release-control/backend/internal/util"
)

type BatchService interface {
	List(context.Context, repository.BatchFilter) (dto.PageResult[model.ProductionBatch], error)
	Get(context.Context, uint) (*model.ProductionBatch, error)
	Create(context.Context, Actor, dto.CreateBatchRequest) (*model.ProductionBatch, error)
	Update(context.Context, Actor, uint, dto.UpdateBatchRequest) (*model.ProductionBatch, error)
	Transition(context.Context, Actor, uint, constants.BatchStatus, string) (*model.ProductionBatch, error)
	Overview(context.Context) (*dto.QualityOverview, error)
}

type batchService struct {
	repo     repository.BatchRepository
	lineRepo repository.LineRepository
	audit    AuditService
	tx       repository.Transactor
}

func NewBatchService(repo repository.BatchRepository, lineRepo repository.LineRepository, audit AuditService, tx repository.Transactor) BatchService {
	return &batchService{repo: repo, lineRepo: lineRepo, audit: audit, tx: tx}
}

func (s *batchService) List(ctx context.Context, filter repository.BatchFilter) (dto.PageResult[model.ProductionBatch], error) {
	query := filter.PageQuery.Normalize()
	filter.PageQuery = query
	items, total, err := s.repo.List(ctx, filter)
	return dto.PageResult[model.ProductionBatch]{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, err
}

func (s *batchService) Get(ctx context.Context, id uint) (*model.ProductionBatch, error) {
	return s.repo.Find(ctx, id)
}

func (s *batchService) Overview(ctx context.Context) (*dto.QualityOverview, error) {
	return s.repo.Overview(ctx)
}

func (s *batchService) Create(ctx context.Context, actor Actor, input dto.CreateBatchRequest) (*model.ProductionBatch, error) {
	var batch *model.ProductionBatch
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		input.BatchNo = strings.ToUpper(strings.TrimSpace(input.BatchNo))
		if _, err := s.repo.FindByNumber(txCtx, input.BatchNo); err == nil {
			return util.Conflict("批次号已存在")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		line, err := s.lineRepo.FindForUpdate(txCtx, input.PackagingLineID)
		if err != nil {
			return err
		}
		if !line.Active || line.EquipmentStatus == "fault" || line.EquipmentStatus == "maintenance" {
			return util.Conflict("目标产线当前不可用于生产")
		}
		batch = &model.ProductionBatch{
			BatchNo: input.BatchNo, Specification: input.Specification, Status: constants.BatchStatusDraft,
			ResponsibleTeam: input.ResponsibleTeam, PackagingLineID: input.PackagingLineID,
			PlannedQuantity: input.PlannedQuantity, ProducedQuantity: input.ProducedQuantity,
		}
		batch.Normalize()
		if err := batch.Validate(); err != nil {
			return util.BadRequest(err.Error())
		}
		if err := s.repo.Create(txCtx, batch); err != nil {
			return err
		}
		return s.audit.Record(txCtx, actor, "batch.created", "ProductionBatch", batch.ID, nil, batch)
	})
	if err != nil {
		return nil, err
	}
	return s.repo.Find(ctx, batch.ID)
}

func (s *batchService) Update(ctx context.Context, actor Actor, id uint, input dto.UpdateBatchRequest) (*model.ProductionBatch, error) {
	var batch *model.ProductionBatch
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		var err error
		batch, err = s.repo.FindForUpdate(txCtx, id)
		if err != nil {
			return err
		}
		if batch.Status == constants.BatchStatusReleased {
			return util.Conflict("已放行批次不可编辑")
		}
		before := *batch
		if input.Specification != nil {
			batch.Specification = strings.TrimSpace(*input.Specification)
		}
		if input.ResponsibleTeam != nil {
			batch.ResponsibleTeam = strings.TrimSpace(*input.ResponsibleTeam)
		}
		if input.PackagingLineID != nil {
			if batch.Status != constants.BatchStatusDraft {
				return util.Conflict("只有草稿批次可以更换产线")
			}
			line, lineErr := s.lineRepo.FindForUpdate(txCtx, *input.PackagingLineID)
			if lineErr != nil {
				return lineErr
			}
			if !line.Active {
				return util.Conflict("目标产线已停用")
			}
			batch.PackagingLineID = *input.PackagingLineID
		}
		if input.PlannedQuantity != nil {
			batch.PlannedQuantity = *input.PlannedQuantity
		}
		if input.ProducedQuantity != nil {
			if *input.ProducedQuantity > batch.PlannedQuantity*2 {
				return util.BadRequest("实际数量异常，请复核")
			}
			batch.ProducedQuantity = *input.ProducedQuantity
		}
		if input.HoldReason != nil {
			batch.HoldReason = strings.TrimSpace(*input.HoldReason)
		}
		batch.Normalize()
		if err := batch.Validate(); err != nil {
			return util.BadRequest(err.Error())
		}
		if err := s.repo.Save(txCtx, batch); err != nil {
			return err
		}
		return s.audit.Record(txCtx, actor, "batch.updated", "ProductionBatch", batch.ID, before, batch)
	})
	if err != nil {
		return nil, err
	}
	return s.repo.Find(ctx, batch.ID)
}

func (s *batchService) Transition(ctx context.Context, actor Actor, id uint, next constants.BatchStatus, reason string) (*model.ProductionBatch, error) {
	var batch *model.ProductionBatch
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		var err error
		batch, err = s.repo.FindForUpdate(txCtx, id)
		if err != nil {
			return err
		}
		if next == constants.BatchStatusReleased {
			return util.Forbidden("批次放行只能通过放行审批完成")
		}
		if err := constants.ValidateBatchTransition(batch.Status, next); err != nil {
			return util.Conflict(err.Error())
		}
		if next == constants.BatchStatusHold && strings.TrimSpace(reason) == "" {
			return util.BadRequest("暂停批次必须填写原因")
		}
		before := *batch
		now := time.Now()
		if next == constants.BatchStatusRunning && batch.StartedAt == nil {
			batch.StartedAt = &now
		}
		if next == constants.BatchStatusHold {
			batch.HoldReason = strings.TrimSpace(reason)
		}
		if next == constants.BatchStatusRunning {
			batch.HoldReason = ""
		}
		batch.Status = next
		batch.Normalize()
		if err := batch.Validate(); err != nil {
			return util.BadRequest(err.Error())
		}
		if err := s.repo.Save(txCtx, batch); err != nil {
			return err
		}
		return s.audit.Record(txCtx, actor, "batch.transitioned", "ProductionBatch", batch.ID, before, batch)
	})
	if err != nil {
		return nil, err
	}
	return s.repo.Find(ctx, batch.ID)
}
