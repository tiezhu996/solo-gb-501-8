package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"sterile-packaging-release-control/backend/internal/constants"
	"sterile-packaging-release-control/backend/internal/dto"
	"sterile-packaging-release-control/backend/internal/model"
	"sterile-packaging-release-control/backend/internal/repository"
	"sterile-packaging-release-control/backend/internal/util"
)

type ReleaseService interface {
	List(context.Context, dto.PageQuery, string) (dto.PageResult[model.ReleaseDecision], error)
	Get(context.Context, uint) (*model.ReleaseDecision, error)
	Decide(context.Context, Actor, dto.CreateReleaseDecisionRequest) (*model.ReleaseDecision, error)
}

type releaseService struct {
	repo           repository.ReleaseRepository
	batchRepo      repository.BatchRepository
	inspectionRepo repository.InspectionRepository
	audit          AuditService
	tx             repository.Transactor
}

func NewReleaseService(repo repository.ReleaseRepository, batchRepo repository.BatchRepository, inspectionRepo repository.InspectionRepository, audit AuditService, tx repository.Transactor) ReleaseService {
	return &releaseService{repo: repo, batchRepo: batchRepo, inspectionRepo: inspectionRepo, audit: audit, tx: tx}
}

func (s *releaseService) List(ctx context.Context, query dto.PageQuery, decision string) (dto.PageResult[model.ReleaseDecision], error) {
	query = query.Normalize()
	items, total, err := s.repo.List(ctx, query, decision)
	return dto.PageResult[model.ReleaseDecision]{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, err
}

func (s *releaseService) Get(ctx context.Context, id uint) (*model.ReleaseDecision, error) {
	return s.repo.Find(ctx, id)
}

func (s *releaseService) Decide(ctx context.Context, actor Actor, input dto.CreateReleaseDecisionRequest) (*model.ReleaseDecision, error) {
	if !input.Decision.Valid() {
		return nil, util.BadRequest("无效的放行决定")
	}
	var decision *model.ReleaseDecision
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		batch, err := s.batchRepo.FindForUpdate(txCtx, input.ProductionBatchID)
		if err != nil {
			return err
		}
		if batch.Status == constants.BatchStatusDraft {
			return util.Conflict("草稿批次不能审批")
		}
		if batch.Status == constants.BatchStatusReleased {
			return util.Conflict("批次已经放行")
		}
		incomplete, err := s.inspectionRepo.CountIncomplete(txCtx, batch.ID)
		if err != nil {
			return err
		}
		failed, err := s.inspectionRepo.CountByResult(txCtx, batch.ID, "fail")
		if err != nil {
			return err
		}
		if input.Decision == constants.DecisionRelease {
			if len(batch.Inspections) == 0 {
				return util.Conflict("批次至少需要一项检验结果")
			}
			if incomplete > 0 {
				return util.Conflict("仍有待完成或待复测的检验")
			}
			if failed > 0 {
				return util.Conflict("存在不合格检验，不能放行")
			}
		}
		before := *batch
		switch input.Decision {
		case constants.DecisionRelease:
			batch.Status = constants.BatchStatusReleased
			now := time.Now()
			batch.CompletedAt = &now
		case constants.DecisionQuarantine:
			batch.Status = constants.BatchStatusHold
			batch.HoldReason = strings.TrimSpace(input.Reason)
		case constants.DecisionRework:
			batch.Status = constants.BatchStatusRework
			batch.HoldReason = strings.TrimSpace(input.Reason)
		}
		decision = &model.ReleaseDecision{
			ProductionBatchID: batch.ID, Decision: input.Decision, ApproverID: actor.ID,
			ApproverName: actor.Name, Reason: strings.TrimSpace(input.Reason), EffectiveAt: time.Now(),
			InspectionSummary: fmt.Sprintf("共 %d 项检验，%d 项不合格，%d 项待处理", len(batch.Inspections), failed, incomplete),
		}
		decision.Normalize()
		if err := decision.Validate(); err != nil {
			return util.BadRequest(err.Error())
		}
		if err := s.repo.CreateWithBatch(txCtx, decision, batch); err != nil {
			return err
		}
		return s.audit.Record(txCtx, actor, "release.decided", "ProductionBatch", batch.ID, before, batch)
	})
	if err != nil {
		return nil, err
	}
	return s.repo.Find(ctx, decision.ID)
}
