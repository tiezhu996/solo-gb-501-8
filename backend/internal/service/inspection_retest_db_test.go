package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"gorm.io/gorm"

	"sterile-packaging-release-control/backend/internal/constants"
	"sterile-packaging-release-control/backend/internal/dto"
	"sterile-packaging-release-control/backend/internal/model"
	"sterile-packaging-release-control/backend/internal/repository"
	"sterile-packaging-release-control/backend/internal/util"
)

// 本文件中的用例针对真实持久化流程：真实 PostgreSQL、真实 Transactor、真实
// GORM 仓库和真实 SELECT ... FOR UPDATE 行锁。只有这样，"生产事务被绕过"和
// "行锁失效"这两类故障才会真实复现为测试失败，而不是被内存 fake 掩盖。

var (
	pgServer  *embeddedpostgres.EmbeddedPostgres
	pgOnce    sync.Once
	pgErr     error
	sharedDB  *gorm.DB
	pgStarted bool
)

func TestMain(m *testing.M) {
	code := m.Run()
	if pgStarted && pgServer != nil {
		_ = pgServer.Stop()
	}
	os.Exit(code)
}

func testDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	pgOnce.Do(func() {
		pgServer = embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
			Version("16.4.0").
			Port(55439).
			Username("postgres").
			Password("postgres").
			Database("postgres"))
		if err := pgServer.Start(); err != nil {
			pgErr = fmt.Errorf("start embedded postgres: %w", err)
			return
		}
		pgStarted = true
		db, err := util.OpenDatabase("postgres://postgres:postgres@localhost:55439/postgres?sslmode=disable")
		if err != nil {
			pgErr = err
			return
		}
		if err := util.Migrate(db); err != nil {
			pgErr = fmt.Errorf("migrate: %w", err)
			return
		}
		sharedDB = db
	})
	if pgErr != nil {
		t.Fatalf("test database unavailable: %v", pgErr)
	}
	resetTables(t, sharedDB)
	return sharedDB
}

func resetTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	err := db.Exec("TRUNCATE packaging_lines, production_batches, inspection_samples, inspection_retests, release_decisions, audit_logs RESTART IDENTITY CASCADE").Error
	if err != nil {
		t.Fatalf("reset tables: %v", err)
	}
}

func seedLineBatchSample(t *testing.T, db *gorm.DB) (uint, uint) {
	t.Helper()
	line := &model.PackagingLine{
		Code: "PKG-DB-T01", Name: "数据库测试包装线", Team: "验证班",
		EquipmentStatus: "ready", Location: "A 区", Active: true,
	}
	if err := db.Create(line).Error; err != nil {
		t.Fatalf("seed line: %v", err)
	}
	batch := &model.ProductionBatch{
		BatchNo: "DB-TEST-001", Specification: "无菌屏障袋", Status: constants.BatchStatusRunning,
		ResponsibleTeam: "验证班", PackagingLineID: line.ID, PlannedQuantity: 1000, ProducedQuantity: 800,
	}
	if err := db.Create(batch).Error; err != nil {
		t.Fatalf("seed batch: %v", err)
	}
	sample := &model.InspectionSample{
		ProductionBatchID: batch.ID, SampleCode: "DB-TEST-S1", SamplingPosition: "中段",
		InspectionItem: "热封强度", AcceptanceRange: ">= 1.50 N/15mm", Result: "pending", RetestStatus: "none",
	}
	if err := db.Create(sample).Error; err != nil {
		t.Fatalf("seed sample: %v", err)
	}
	return batch.ID, sample.ID
}

func realInspectionService(db *gorm.DB, retestRepo repository.RetestRepository) InspectionService {
	if retestRepo == nil {
		retestRepo = repository.NewRetestRepository(db)
	}
	return NewInspectionService(
		repository.NewInspectionRepository(db),
		retestRepo,
		repository.NewBatchRepository(db),
		NewAuditService(repository.NewAuditRepository(db)),
		repository.NewTransactor(db),
	)
}

func countRows(t *testing.T, db *gorm.DB, modelValue any, where string, args ...any) int64 {
	t.Helper()
	var count int64
	if err := db.Model(modelValue).Where(where, args...).Count(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return count
}

// 并发完成同一复测：在真实行锁下只有一个请求成功，其余得到 409，历史只追加
// 一条。delayedSaveRepo 把"读-改-写"窗口拉宽，保证并发交错确定性发生：行锁
// 失效时所有请求都读到待复测状态并全部写入，产生多个成功和多条历史（或唯一
// 索引冲突的 500），本用例随之失败；行锁正常时锁在 FindForUpdate 阶段就已
// 串行，延迟不影响结果。
func TestConcurrentCompleteKeepsSingleHistory(t *testing.T) {
	db := testDatabase(t)
	_, sampleID := seedLineBatchSample(t, db)
	svc := NewInspectionService(
		delayedSaveRepo{inner: repository.NewInspectionRepository(db), delay: 75 * time.Millisecond},
		repository.NewRetestRepository(db),
		repository.NewBatchRepository(db),
		NewAuditService(repository.NewAuditRepository(db)),
		repository.NewTransactor(db),
	)
	ctx := context.Background()

	if _, err := svc.Complete(ctx, testActor, sampleID, dto.CompleteInspectionRequest{
		Result: "fail", MeasuredValue: "1.33 N/15mm", Notes: "首检不合格，等待复测",
	}); err != nil {
		t.Fatalf("seed failing first inspection: %v", err)
	}

	const workers = 8
	var wg sync.WaitGroup
	errs := make([]error, workers)
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = svc.Complete(ctx, testActor, sampleID, dto.CompleteInspectionRequest{
				Result: "pass", MeasuredValue: "1.65 N/15mm", Notes: "复测合格",
			})
		}(i)
	}
	close(start)
	wg.Wait()

	var succeeded, conflicts int
	for _, err := range errs {
		if err == nil {
			succeeded++
			continue
		}
		var apiErr *util.APIError
		if !errors.As(err, &apiErr) || apiErr.Status != http.StatusConflict {
			t.Fatalf("loser must get 409 conflict, got: %v", err)
		}
		conflicts++
	}
	if succeeded != 1 || conflicts != workers-1 {
		t.Fatalf("concurrent complete: %d succeeded, %d conflicts; want 1 and %d", succeeded, conflicts, workers-1)
	}
	if got := countRows(t, db, &model.InspectionRetest{}, "inspection_sample_id = ?", sampleID); got != 2 {
		t.Fatalf("retest history rows = %d, want 2 (initial + single concurrent retest)", got)
	}
	var sample model.InspectionSample
	if err := db.First(&sample, sampleID).Error; err != nil {
		t.Fatal(err)
	}
	if sample.Result != "pass" || sample.RetestStatus != "completed" {
		t.Fatalf("final state = (%s, %s), want (pass, completed)", sample.Result, sample.RetestStatus)
	}
}

// 历史写入失败时，真实事务必须让样本状态、既有历史和审计记录一起回滚。
// 生产事务被绕过时，样本更新会脱离事务直接落库，下面的数据库断言随之失败。
func TestCompleteRollsBackWhenRetestHistoryFails(t *testing.T) {
	historyFailure := errors.New("retest history store unavailable")

	t.Run("first inspection", func(t *testing.T) {
		db := testDatabase(t)
		_, sampleID := seedLineBatchSample(t, db)
		svc := realInspectionService(db, failingRetestCreate{inner: repository.NewRetestRepository(db), err: historyFailure})

		_, err := svc.Complete(context.Background(), testActor, sampleID, dto.CompleteInspectionRequest{
			Result: "fail", MeasuredValue: "1.30 N/15mm", Notes: "首检不合格",
		})
		if !errors.Is(err, historyFailure) {
			t.Fatalf("got %v, want injected history failure", err)
		}
		var sample model.InspectionSample
		if err := db.First(&sample, sampleID).Error; err != nil {
			t.Fatal(err)
		}
		if sample.Result != "pending" || sample.MeasuredValue != "" || sample.InspectedAt != nil || sample.InspectorName != "" {
			t.Fatalf("sample state leaked from rolled back transaction: %+v", sample)
		}
		if got := countRows(t, db, &model.InspectionRetest{}, "inspection_sample_id = ?", sampleID); got != 0 {
			t.Fatalf("retest history rows = %d, want 0 after rollback", got)
		}
		if got := countRows(t, db, &model.AuditLog{}, "1=1"); got != 0 {
			t.Fatalf("audit rows = %d, want 0 after rollback", got)
		}
	})

	t.Run("retest keeps earlier rounds", func(t *testing.T) {
		db := testDatabase(t)
		_, sampleID := seedLineBatchSample(t, db)
		if _, err := realInspectionService(db, nil).Complete(context.Background(), testActor, sampleID, dto.CompleteInspectionRequest{
			Result: "fail", MeasuredValue: "1.30 N/15mm", Notes: "首检不合格",
		}); err != nil {
			t.Fatalf("seed failing first inspection: %v", err)
		}

		svc := realInspectionService(db, failingRetestCreate{inner: repository.NewRetestRepository(db), err: historyFailure})
		_, err := svc.Complete(context.Background(), testActor, sampleID, dto.CompleteInspectionRequest{
			Result: "pass", MeasuredValue: "1.66 N/15mm", Notes: "复测合格",
		})
		if !errors.Is(err, historyFailure) {
			t.Fatalf("got %v, want injected history failure", err)
		}
		var sample model.InspectionSample
		if err := db.First(&sample, sampleID).Error; err != nil {
			t.Fatal(err)
		}
		if sample.Result != "fail" || sample.RetestStatus != "requested" || sample.MeasuredValue != "1.30 N/15mm" {
			t.Fatalf("sample state leaked from rolled back retest: %+v", sample)
		}
		if got := countRows(t, db, &model.InspectionRetest{}, "inspection_sample_id = ?", sampleID); got != 1 {
			t.Fatalf("retest history rows = %d, want only the first round after rollback", got)
		}
		if got := countRows(t, db, &model.AuditLog{}, "entity_type = ? AND entity_id = ?", "InspectionSample", sampleID); got != 1 {
			t.Fatalf("audit rows = %d, want only the first completed round", got)
		}
	})
}

// delayedSaveRepo 包装真实检验仓库，仅在 Save 前加入固定延迟，把并发事务的
// "读-改-写"窗口拉宽到确定性交错；其余方法直接走真实持久化。
type delayedSaveRepo struct {
	inner repository.InspectionRepository
	delay time.Duration
}

func (r delayedSaveRepo) List(ctx context.Context, filter repository.InspectionFilter) ([]model.InspectionSample, int64, error) {
	return r.inner.List(ctx, filter)
}

func (r delayedSaveRepo) Find(ctx context.Context, id uint) (*model.InspectionSample, error) {
	return r.inner.Find(ctx, id)
}

func (r delayedSaveRepo) FindForUpdate(ctx context.Context, id uint) (*model.InspectionSample, error) {
	return r.inner.FindForUpdate(ctx, id)
}

func (r delayedSaveRepo) FindByCode(ctx context.Context, code string) (*model.InspectionSample, error) {
	return r.inner.FindByCode(ctx, code)
}

func (r delayedSaveRepo) Create(ctx context.Context, sample *model.InspectionSample) error {
	return r.inner.Create(ctx, sample)
}

func (r delayedSaveRepo) Save(ctx context.Context, sample *model.InspectionSample) error {
	time.Sleep(r.delay)
	return r.inner.Save(ctx, sample)
}

func (r delayedSaveRepo) CountByResult(ctx context.Context, batchID uint, result string) (int64, error) {
	return r.inner.CountByResult(ctx, batchID, result)
}

func (r delayedSaveRepo) CountIncomplete(ctx context.Context, batchID uint) (int64, error) {
	return r.inner.CountIncomplete(ctx, batchID)
}

// failingRetestCreate 包装真实仓库，仅注入 Create 失败，其余行为走真实持久化。
type failingRetestCreate struct {
	inner repository.RetestRepository
	err   error
}

func (r failingRetestCreate) Create(context.Context, *model.InspectionRetest) error {
	return r.err
}

func (r failingRetestCreate) ListBySample(ctx context.Context, sampleID uint) ([]model.InspectionRetest, error) {
	return r.inner.ListBySample(ctx, sampleID)
}

func (r failingRetestCreate) CountBySample(ctx context.Context, sampleID uint) (int64, error) {
	return r.inner.CountBySample(ctx, sampleID)
}
