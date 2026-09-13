package service

import (
	"context"
	"encoding/json"
	"fmt"

	"sterile-packaging-release-control/backend/internal/dto"
	"sterile-packaging-release-control/backend/internal/model"
	"sterile-packaging-release-control/backend/internal/repository"
)

type Actor struct {
	ID        uint
	Name      string
	RequestID string
	IP        string
}

type AuditService interface {
	Record(context.Context, Actor, string, string, uint, any, any) error
	List(context.Context, repository.AuditFilter) (dto.PageResult[model.AuditLog], error)
}

type auditService struct{ repo repository.AuditRepository }

func NewAuditService(repo repository.AuditRepository) AuditService {
	return &auditService{repo: repo}
}

func (s *auditService) Record(ctx context.Context, actor Actor, action, entityType string, entityID uint, before, after any) error {
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return fmt.Errorf("serialize audit before state: %w", err)
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		return fmt.Errorf("serialize audit after state: %w", err)
	}
	log := &model.AuditLog{
		RequestID: actor.RequestID, ActorID: actor.ID, ActorName: actor.Name,
		Action: action, EntityType: entityType, EntityID: entityID,
		BeforeState: string(beforeJSON), AfterState: string(afterJSON), IPAddress: actor.IP,
	}
	return s.repo.Create(ctx, log)
}

func (s *auditService) List(ctx context.Context, filter repository.AuditFilter) (dto.PageResult[model.AuditLog], error) {
	query := filter.PageQuery.Normalize()
	filter.PageQuery = query
	items, total, err := s.repo.List(ctx, filter)
	return dto.PageResult[model.AuditLog]{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, err
}
