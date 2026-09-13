package service

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"sterile-packaging-release-control/backend/internal/dto"
	"sterile-packaging-release-control/backend/internal/model"
	"sterile-packaging-release-control/backend/internal/repository"
	"sterile-packaging-release-control/backend/internal/util"
)

type LineService interface {
	List(context.Context, dto.PageQuery) (dto.PageResult[model.PackagingLine], error)
	Get(context.Context, uint) (*model.PackagingLine, error)
	Create(context.Context, Actor, dto.CreateLineRequest) (*model.PackagingLine, error)
	Update(context.Context, Actor, uint, dto.UpdateLineRequest) (*model.PackagingLine, error)
}

type lineService struct {
	repo  repository.LineRepository
	audit AuditService
	tx    repository.Transactor
}

func NewLineService(repo repository.LineRepository, audit AuditService, tx repository.Transactor) LineService {
	return &lineService{repo: repo, audit: audit, tx: tx}
}

func (s *lineService) List(ctx context.Context, query dto.PageQuery) (dto.PageResult[model.PackagingLine], error) {
	query = query.Normalize()
	items, total, err := s.repo.List(ctx, query)
	return dto.PageResult[model.PackagingLine]{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, err
}

func (s *lineService) Get(ctx context.Context, id uint) (*model.PackagingLine, error) {
	return s.repo.Find(ctx, id)
}

func (s *lineService) Create(ctx context.Context, actor Actor, input dto.CreateLineRequest) (*model.PackagingLine, error) {
	var line *model.PackagingLine
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
		if _, err := s.repo.FindByCode(txCtx, input.Code); err == nil {
			return util.Conflict("产线编码已存在")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		line = &model.PackagingLine{Code: input.Code, Name: input.Name, Team: input.Team, EquipmentStatus: input.EquipmentStatus, Location: input.Location, Active: true}
		line.Normalize()
		if err := line.Validate(); err != nil {
			return util.BadRequest(err.Error())
		}
		if err := s.repo.Create(txCtx, line); err != nil {
			return err
		}
		return s.audit.Record(txCtx, actor, "line.created", "PackagingLine", line.ID, nil, line)
	})
	return line, err
}

func (s *lineService) Update(ctx context.Context, actor Actor, id uint, input dto.UpdateLineRequest) (*model.PackagingLine, error) {
	var line *model.PackagingLine
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		var err error
		line, err = s.repo.FindForUpdate(txCtx, id)
		if err != nil {
			return err
		}
		before := *line
		if input.Name != nil {
			line.Name = strings.TrimSpace(*input.Name)
		}
		if input.Team != nil {
			line.Team = strings.TrimSpace(*input.Team)
		}
		if input.EquipmentStatus != nil {
			line.EquipmentStatus = *input.EquipmentStatus
		}
		if input.Location != nil {
			line.Location = strings.TrimSpace(*input.Location)
		}
		if input.Active != nil {
			if !*input.Active {
				count, countErr := s.repo.CountActiveBatches(txCtx, id)
				if countErr != nil {
					return countErr
				}
				if count > 0 {
					return util.Conflict("产线存在未结束批次，不能停用")
				}
			}
			line.Active = *input.Active
		}
		line.Normalize()
		if err := line.Validate(); err != nil {
			return util.BadRequest(err.Error())
		}
		if err := s.repo.Save(txCtx, line); err != nil {
			return err
		}
		return s.audit.Record(txCtx, actor, "line.updated", "PackagingLine", line.ID, before, line)
	})
	return line, err
}
