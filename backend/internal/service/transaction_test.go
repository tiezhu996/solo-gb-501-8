package service

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	"sterile-packaging-release-control/backend/internal/dto"
	"sterile-packaging-release-control/backend/internal/model"
	"sterile-packaging-release-control/backend/internal/repository"
)

type trackingTransactor struct{ active bool }

func (t *trackingTransactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	t.active = true
	defer func() { t.active = false }()
	return fn(ctx)
}

type trackingLineRepository struct {
	tx      *trackingTransactor
	created bool
}

func (r *trackingLineRepository) List(context.Context, dto.PageQuery) ([]model.PackagingLine, int64, error) {
	return nil, 0, nil
}
func (r *trackingLineRepository) Find(context.Context, uint) (*model.PackagingLine, error) {
	return nil, errors.New("unused")
}
func (r *trackingLineRepository) FindForUpdate(context.Context, uint) (*model.PackagingLine, error) {
	return nil, errors.New("unused")
}
func (r *trackingLineRepository) FindByCode(context.Context, string) (*model.PackagingLine, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *trackingLineRepository) Create(_ context.Context, line *model.PackagingLine) error {
	if !r.tx.active {
		return errors.New("create ran outside transaction")
	}
	line.ID = 1
	r.created = true
	return nil
}
func (r *trackingLineRepository) Save(context.Context, *model.PackagingLine) error { return nil }
func (r *trackingLineRepository) CountActiveBatches(context.Context, uint) (int64, error) {
	return 0, nil
}

type failingAuditService struct {
	tx  *trackingTransactor
	err error
}

func (s failingAuditService) Record(context.Context, Actor, string, string, uint, any, any) error {
	if !s.tx.active {
		return errors.New("audit ran outside transaction")
	}
	return s.err
}
func (s failingAuditService) List(context.Context, repository.AuditFilter) (dto.PageResult[model.AuditLog], error) {
	return dto.PageResult[model.AuditLog]{}, nil
}

func TestLineCreateGroupsMutationAndAuditInTransaction(t *testing.T) {
	tx := &trackingTransactor{}
	repo := &trackingLineRepository{tx: tx}
	auditFailure := errors.New("audit unavailable")
	svc := NewLineService(repo, failingAuditService{tx: tx, err: auditFailure}, tx)
	_, err := svc.Create(context.Background(), Actor{ID: 1, Name: "admin", RequestID: "req-1"}, dto.CreateLineRequest{
		Code: "PKG-T01", Name: "测试包装线", Team: "测试班", EquipmentStatus: "ready", Location: "A 区",
	})
	if !errors.Is(err, auditFailure) {
		t.Fatalf("got %v, want audit failure", err)
	}
	if !repo.created {
		t.Fatal("business mutation was not attempted before audit")
	}
}
