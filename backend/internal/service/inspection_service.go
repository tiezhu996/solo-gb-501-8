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

type InspectionService interface {
	List(context.Context, repository.InspectionFilter) (dto.PageResult[model.InspectionSample], error)
	Get(context.Context, uint) (*model.InspectionSample, error)
	Create(context.Context, Actor, dto.CreateInspectionRequest) (*model.InspectionSample, error)
	Complete(context.Context, Actor, uint, dto.CompleteInspectionRequest) (*model.InspectionSample, error)
	RequestRetest(context.Context, Actor, uint, string) (*model.InspectionSample, error)
}

type inspectionService struct {
	repo       repository.InspectionRepository
	retestRepo repository.RetestRepository
	batchRepo  repository.BatchRepository
	audit      AuditService
	tx         repository.Transactor
}

func NewInspectionService(repo repository.InspectionRepository, retestRepo repository.RetestRepository, batchRepo repository.BatchRepository, audit AuditService, tx repository.Transactor) InspectionService {
	return &inspectionService{repo: repo, retestRepo: retestRepo, batchRepo: batchRepo, audit: audit, tx: tx}
}

func (s *inspectionService) List(ctx context.Context, filter repository.InspectionFilter) (dto.PageResult[model.InspectionSample], error) {
	query := filter.PageQuery.Normalize()
	filter.PageQuery = query
	items, total, err := s.repo.List(ctx, filter)
	return dto.PageResult[model.InspectionSample]{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, err
}

func (s *inspectionService) Get(ctx context.Context, id uint) (*model.InspectionSample, error) {
	return s.repo.Find(ctx, id)
}

func (s *inspectionService) Create(ctx context.Context, actor Actor, input dto.CreateInspectionRequest) (*model.InspectionSample, error) {
	var sample *model.InspectionSample
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		input.SampleCode = strings.ToUpper(strings.TrimSpace(input.SampleCode))
		if _, err := s.repo.FindByCode(txCtx, input.SampleCode); err == nil {
			return util.Conflict("样本编号已存在")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		batch, err := s.batchRepo.FindForUpdate(txCtx, input.ProductionBatchID)
		if err != nil {
			return err
		}
		if batch.Status == constants.BatchStatusDraft || batch.Status == constants.BatchStatusReleased {
			return util.Conflict("当前批次状态不允许新增检验样本")
		}
		sample = &model.InspectionSample{
			ProductionBatchID: input.ProductionBatchID, SampleCode: input.SampleCode,
			SamplingPosition: input.SamplingPosition, InspectionItem: input.InspectionItem,
			AcceptanceRange: input.AcceptanceRange, Result: "pending", RetestStatus: "none", Notes: input.Notes,
		}
		sample.Normalize()
		if err := sample.ValidateDefinition(); err != nil {
			return util.BadRequest(err.Error())
		}
		if err := s.repo.Create(txCtx, sample); err != nil {
			return err
		}
		return s.audit.Record(txCtx, actor, "inspection.created", "InspectionSample", sample.ID, nil, sample)
	})
	if err != nil {
		return nil, err
	}
	return s.repo.Find(ctx, sample.ID)
}

func (s *inspectionService) Complete(ctx context.Context, actor Actor, id uint, input dto.CompleteInspectionRequest) (*model.InspectionSample, error) {
	current, err := s.repo.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	var sample *model.InspectionSample
	err = s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		batch, err := s.batchRepo.FindForUpdate(txCtx, current.ProductionBatchID)
		if err != nil {
			return err
		}
		if batch.Status == constants.BatchStatusReleased {
			return util.Conflict("已放行批次的检验结果不可修改")
		}
		sample, err = s.repo.FindForUpdate(txCtx, id)
		if err != nil {
			return err
		}
		if sample.Result != "pending" && sample.RetestStatus != "requested" {
			return util.Conflict("检验已经完成")
		}
		before := *sample
		now := time.Now()
		sample.Result = input.Result
		sample.MeasuredValue = strings.TrimSpace(input.MeasuredValue)
		sample.Notes = strings.TrimSpace(input.Notes)
		sample.InspectorID = actor.ID
		sample.InspectorName = actor.Name
		sample.InspectedAt = &now
		if input.RequestRetest || input.Result == "fail" {
			sample.RetestStatus = "requested"
		} else {
			sample.RetestStatus = "none"
		}
		if before.RetestStatus == "requested" && !input.RequestRetest {
			sample.RetestStatus = "completed"
		}
		sample.Normalize()
		if err := sample.ValidateDefinition(); err != nil {
			return util.BadRequest(err.Error())
		}
		if err := s.repo.Save(txCtx, sample); err != nil {
			return err
		}
		if err := s.recordRound(txCtx, sample, before.RetestStatus == "requested"); err != nil {
			return err
		}
		return s.audit.Record(txCtx, actor, "inspection.completed", "InspectionSample", sample.ID, before, sample)
	})
	if err != nil {
		return nil, err
	}
	return s.repo.Find(ctx, sample.ID)
}

// recordRound 把本次完成的检验轮次追加到复测历史：首检记为 initial，复测记为
// retest，序号按样本递增。样本字段保存最新结论，历史记录保证各轮次可追溯。
func (s *inspectionService) recordRound(ctx context.Context, sample *model.InspectionSample, isRetest bool) error {
	count, err := s.retestRepo.CountBySample(ctx, sample.ID)
	if err != nil {
		return err
	}
	round := "initial"
	if isRetest {
		round = "retest"
	}
	record := &model.InspectionRetest{
		InspectionSampleID: sample.ID,
		Round:              round,
		Sequence:           int(count) + 1,
		Result:             sample.Result,
		MeasuredValue:      sample.MeasuredValue,
		InspectorID:        sample.InspectorID,
		InspectorName:      sample.InspectorName,
		InspectedAt:        sample.InspectedAt,
		Notes:              sample.Notes,
	}
	record.Normalize()
	if err := record.Validate(); err != nil {
		return util.BadRequest(err.Error())
	}
	return s.retestRepo.Create(ctx, record)
}

func (s *inspectionService) RequestRetest(ctx context.Context, actor Actor, id uint, reason string) (*model.InspectionSample, error) {
	current, err := s.repo.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(reason) == "" {
		return nil, util.BadRequest("复测原因不能为空")
	}
	var sample *model.InspectionSample
	err = s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		batch, err := s.batchRepo.FindForUpdate(txCtx, current.ProductionBatchID)
		if err != nil {
			return err
		}
		if batch.Status == constants.BatchStatusReleased {
			return util.Conflict("已放行批次不能申请复测")
		}
		sample, err = s.repo.FindForUpdate(txCtx, id)
		if err != nil {
			return err
		}
		if sample.Result != "fail" {
			return util.Conflict("只有不合格结果可以申请复测")
		}
		if sample.RetestStatus == "requested" {
			return util.Conflict("该检验已在等待复测")
		}
		before := *sample
		sample.RetestStatus = "requested"
		sample.Notes = strings.TrimSpace(sample.Notes + "\n复测原因: " + reason)
		sample.Normalize()
		if err := sample.ValidateDefinition(); err != nil {
			return util.BadRequest(err.Error())
		}
		if err := s.repo.Save(txCtx, sample); err != nil {
			return err
		}
		return s.audit.Record(txCtx, actor, "inspection.retest_requested", "InspectionSample", sample.ID, before, sample)
	})
	if err != nil {
		return nil, err
	}
	return s.repo.Find(ctx, sample.ID)
}
