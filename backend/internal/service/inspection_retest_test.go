package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"testing"

	"gorm.io/gorm"

	"sterile-packaging-release-control/backend/internal/constants"
	"sterile-packaging-release-control/backend/internal/dto"
	"sterile-packaging-release-control/backend/internal/model"
	"sterile-packaging-release-control/backend/internal/repository"
	"sterile-packaging-release-control/backend/internal/util"
)

// memEnv 是一个内存版数据环境：memTransactor 串行执行事务（模拟行锁），并在
// 事务出错时恢复快照（模拟回滚），因此可以在不启动数据库的前提下检验复测
// 历史、状态流转、并发去重和事务回滚语义。
type memEnv struct {
	mu           sync.Mutex
	txMu         sync.Mutex
	batches      map[uint]*model.ProductionBatch
	samples      map[uint]*model.InspectionSample
	retests      map[uint][]model.InspectionRetest
	audits       []model.AuditLog
	decisions    map[uint]*model.ReleaseDecision
	nextBatchID  uint
	nextSampleID uint
	nextRecordID uint
	nextDecideID uint
}

func newMemEnv() *memEnv {
	return &memEnv{
		batches:   make(map[uint]*model.ProductionBatch),
		samples:   make(map[uint]*model.InspectionSample),
		retests:   make(map[uint][]model.InspectionRetest),
		decisions: make(map[uint]*model.ReleaseDecision),
	}
}

func cloneBatchValue(b *model.ProductionBatch) *model.ProductionBatch {
	c := *b
	if b.StartedAt != nil {
		t := *b.StartedAt
		c.StartedAt = &t
	}
	if b.CompletedAt != nil {
		t := *b.CompletedAt
		c.CompletedAt = &t
	}
	c.Inspections = nil
	c.Decisions = nil
	return &c
}

func cloneSampleValue(s *model.InspectionSample) *model.InspectionSample {
	c := *s
	if s.InspectedAt != nil {
		t := *s.InspectedAt
		c.InspectedAt = &t
	}
	c.Retests = nil
	c.ProductionBatch = model.ProductionBatch{}
	return &c
}

func (e *memEnv) retestCopy(sampleID uint) []model.InspectionRetest {
	records := make([]model.InspectionRetest, len(e.retests[sampleID]))
	copy(records, e.retests[sampleID])
	sort.Slice(records, func(i, j int) bool { return records[i].Sequence < records[j].Sequence })
	return records
}

func (e *memEnv) inspectionsOf(batchID uint) []model.InspectionSample {
	var items []model.InspectionSample
	for _, sample := range e.samples {
		if sample.ProductionBatchID == batchID {
			items = append(items, *cloneSampleValue(sample))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

type memSnapshot struct {
	batches      map[uint]*model.ProductionBatch
	samples      map[uint]*model.InspectionSample
	retests      map[uint][]model.InspectionRetest
	audits       []model.AuditLog
	decisions    map[uint]*model.ReleaseDecision
	nextRecordID uint
	nextDecideID uint
}

func (e *memEnv) snapshot() memSnapshot {
	snap := memSnapshot{
		batches:      make(map[uint]*model.ProductionBatch, len(e.batches)),
		samples:      make(map[uint]*model.InspectionSample, len(e.samples)),
		retests:      make(map[uint][]model.InspectionRetest, len(e.retests)),
		audits:       append([]model.AuditLog(nil), e.audits...),
		decisions:    make(map[uint]*model.ReleaseDecision, len(e.decisions)),
		nextRecordID: e.nextRecordID,
		nextDecideID: e.nextDecideID,
	}
	for id, batch := range e.batches {
		snap.batches[id] = cloneBatchValue(batch)
	}
	for id, sample := range e.samples {
		snap.samples[id] = cloneSampleValue(sample)
	}
	for id, records := range e.retests {
		snap.retests[id] = append([]model.InspectionRetest(nil), records...)
	}
	for id, decision := range e.decisions {
		c := *decision
		snap.decisions[id] = &c
	}
	return snap
}

func (e *memEnv) restore(snap memSnapshot) {
	e.batches = snap.batches
	e.samples = snap.samples
	e.retests = snap.retests
	e.audits = snap.audits
	e.decisions = snap.decisions
	e.nextRecordID = snap.nextRecordID
	e.nextDecideID = snap.nextDecideID
}

// memTransactor 串行执行事务函数（等价于相关行被 SELECT ... FOR UPDATE 锁定），
// 出错时整体恢复快照，与真实数据库事务回滚一致。
type memTransactor struct{ env *memEnv }

func (t *memTransactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	t.env.txMu.Lock()
	defer t.env.txMu.Unlock()
	t.env.mu.Lock()
	snap := t.env.snapshot()
	t.env.mu.Unlock()
	if err := fn(ctx); err != nil {
		t.env.mu.Lock()
		t.env.restore(snap)
		t.env.mu.Unlock()
		return err
	}
	return nil
}

type memInspectionRepo struct{ env *memEnv }

func (r *memInspectionRepo) List(context.Context, repository.InspectionFilter) ([]model.InspectionSample, int64, error) {
	return nil, 0, errors.New("unused")
}

func (r *memInspectionRepo) Find(_ context.Context, id uint) (*model.InspectionSample, error) {
	r.env.mu.Lock()
	defer r.env.mu.Unlock()
	sample, ok := r.env.samples[id]
	if !ok {
		return nil, fmt.Errorf("sample %d: %w", id, gorm.ErrRecordNotFound)
	}
	c := cloneSampleValue(sample)
	c.Retests = r.env.retestCopy(id)
	return c, nil
}

func (r *memInspectionRepo) FindForUpdate(ctx context.Context, id uint) (*model.InspectionSample, error) {
	return r.Find(ctx, id)
}

func (r *memInspectionRepo) FindByCode(_ context.Context, code string) (*model.InspectionSample, error) {
	r.env.mu.Lock()
	defer r.env.mu.Unlock()
	for _, sample := range r.env.samples {
		if sample.SampleCode == code {
			return cloneSampleValue(sample), nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *memInspectionRepo) Create(_ context.Context, sample *model.InspectionSample) error {
	r.env.mu.Lock()
	defer r.env.mu.Unlock()
	r.env.nextSampleID++
	sample.ID = r.env.nextSampleID
	r.env.samples[sample.ID] = cloneSampleValue(sample)
	return nil
}

func (r *memInspectionRepo) Save(_ context.Context, sample *model.InspectionSample) error {
	r.env.mu.Lock()
	defer r.env.mu.Unlock()
	if _, ok := r.env.samples[sample.ID]; !ok {
		return fmt.Errorf("sample %d: %w", sample.ID, gorm.ErrRecordNotFound)
	}
	r.env.samples[sample.ID] = cloneSampleValue(sample)
	return nil
}

func (r *memInspectionRepo) CountByResult(_ context.Context, batchID uint, result string) (int64, error) {
	r.env.mu.Lock()
	defer r.env.mu.Unlock()
	var count int64
	for _, sample := range r.env.samples {
		if sample.ProductionBatchID == batchID && sample.Result == result {
			count++
		}
	}
	return count, nil
}

func (r *memInspectionRepo) CountIncomplete(_ context.Context, batchID uint) (int64, error) {
	r.env.mu.Lock()
	defer r.env.mu.Unlock()
	var count int64
	for _, sample := range r.env.samples {
		if sample.ProductionBatchID == batchID && (sample.Result == "pending" || sample.RetestStatus == "requested") {
			count++
		}
	}
	return count, nil
}

type memRetestRepo struct {
	env       *memEnv
	createErr error
}

func (r *memRetestRepo) Create(_ context.Context, record *model.InspectionRetest) error {
	r.env.mu.Lock()
	defer r.env.mu.Unlock()
	if r.createErr != nil {
		return r.createErr
	}
	for _, existing := range r.env.retests[record.InspectionSampleID] {
		if existing.Sequence == record.Sequence {
			return fmt.Errorf("duplicate sequence %d for sample %d", record.Sequence, record.InspectionSampleID)
		}
	}
	r.env.nextRecordID++
	record.ID = r.env.nextRecordID
	r.env.retests[record.InspectionSampleID] = append(r.env.retests[record.InspectionSampleID], *record)
	return nil
}

func (r *memRetestRepo) ListBySample(_ context.Context, sampleID uint) ([]model.InspectionRetest, error) {
	r.env.mu.Lock()
	defer r.env.mu.Unlock()
	return r.env.retestCopy(sampleID), nil
}

func (r *memRetestRepo) CountBySample(_ context.Context, sampleID uint) (int64, error) {
	r.env.mu.Lock()
	defer r.env.mu.Unlock()
	return int64(len(r.env.retests[sampleID])), nil
}

type memBatchRepo struct{ env *memEnv }

func (r *memBatchRepo) List(context.Context, repository.BatchFilter) ([]model.ProductionBatch, int64, error) {
	return nil, 0, errors.New("unused")
}

func (r *memBatchRepo) Find(_ context.Context, id uint) (*model.ProductionBatch, error) {
	r.env.mu.Lock()
	defer r.env.mu.Unlock()
	batch, ok := r.env.batches[id]
	if !ok {
		return nil, fmt.Errorf("batch %d: %w", id, gorm.ErrRecordNotFound)
	}
	c := cloneBatchValue(batch)
	c.Inspections = r.env.inspectionsOf(id)
	return c, nil
}

func (r *memBatchRepo) FindForUpdate(ctx context.Context, id uint) (*model.ProductionBatch, error) {
	return r.Find(ctx, id)
}

func (r *memBatchRepo) FindByNumber(context.Context, string) (*model.ProductionBatch, error) {
	return nil, errors.New("unused")
}

func (r *memBatchRepo) Create(_ context.Context, batch *model.ProductionBatch) error {
	r.env.mu.Lock()
	defer r.env.mu.Unlock()
	r.env.nextBatchID++
	batch.ID = r.env.nextBatchID
	r.env.batches[batch.ID] = cloneBatchValue(batch)
	return nil
}

func (r *memBatchRepo) Save(_ context.Context, batch *model.ProductionBatch) error {
	r.env.mu.Lock()
	defer r.env.mu.Unlock()
	if _, ok := r.env.batches[batch.ID]; !ok {
		return fmt.Errorf("batch %d: %w", batch.ID, gorm.ErrRecordNotFound)
	}
	r.env.batches[batch.ID] = cloneBatchValue(batch)
	return nil
}

func (r *memBatchRepo) Overview(context.Context) (*dto.QualityOverview, error) {
	return nil, errors.New("unused")
}

type memReleaseRepo struct{ env *memEnv }

func (r *memReleaseRepo) List(context.Context, dto.PageQuery, string) ([]model.ReleaseDecision, int64, error) {
	return nil, 0, errors.New("unused")
}

func (r *memReleaseRepo) Find(_ context.Context, id uint) (*model.ReleaseDecision, error) {
	r.env.mu.Lock()
	defer r.env.mu.Unlock()
	decision, ok := r.env.decisions[id]
	if !ok {
		return nil, fmt.Errorf("decision %d: %w", id, gorm.ErrRecordNotFound)
	}
	c := *decision
	return &c, nil
}

func (r *memReleaseRepo) LatestForBatch(context.Context, uint) (*model.ReleaseDecision, error) {
	return nil, errors.New("unused")
}

func (r *memReleaseRepo) CreateWithBatch(_ context.Context, decision *model.ReleaseDecision, batch *model.ProductionBatch) error {
	r.env.mu.Lock()
	defer r.env.mu.Unlock()
	r.env.nextDecideID++
	decision.ID = r.env.nextDecideID
	c := *decision
	r.env.decisions[decision.ID] = &c
	r.env.batches[batch.ID] = cloneBatchValue(batch)
	return nil
}

type memAudit struct{ env *memEnv }

func (a memAudit) Record(_ context.Context, actor Actor, action, entityType string, entityID uint, _, _ any) error {
	a.env.mu.Lock()
	defer a.env.mu.Unlock()
	a.env.audits = append(a.env.audits, model.AuditLog{
		ActorID: actor.ID, ActorName: actor.Name, Action: action, EntityType: entityType, EntityID: entityID,
	})
	return nil
}

func (a memAudit) List(context.Context, repository.AuditFilter) (dto.PageResult[model.AuditLog], error) {
	return dto.PageResult[model.AuditLog]{}, nil
}

func (e *memEnv) inspectionService() InspectionService {
	return NewInspectionService(&memInspectionRepo{env: e}, &memRetestRepo{env: e}, &memBatchRepo{env: e}, memAudit{env: e}, &memTransactor{env: e})
}

func (e *memEnv) releaseService() ReleaseService {
	return NewReleaseService(&memReleaseRepo{env: e}, &memBatchRepo{env: e}, &memInspectionRepo{env: e}, memAudit{env: e}, &memTransactor{env: e})
}

func (e *memEnv) addBatch(t *testing.T, status constants.BatchStatus) uint {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	e.nextBatchID++
	batch := &model.ProductionBatch{
		BatchNo: fmt.Sprintf("MEM-BATCH-%03d", e.nextBatchID), Specification: "无菌屏障袋",
		Status: status, ResponsibleTeam: "验证班", PackagingLineID: 1,
		PlannedQuantity: 1000, ProducedQuantity: 800,
	}
	batch.ID = e.nextBatchID
	e.batches[batch.ID] = batch
	return batch.ID
}

func (e *memEnv) addSample(t *testing.T, batchID uint) uint {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	e.nextSampleID++
	sample := &model.InspectionSample{
		ProductionBatchID: batchID, SampleCode: fmt.Sprintf("MEM-S-%03d", e.nextSampleID),
		SamplingPosition: "中段", InspectionItem: "热封强度", AcceptanceRange: ">= 1.50 N/15mm",
		Result: "pending", RetestStatus: "none",
	}
	sample.ID = e.nextSampleID
	e.samples[sample.ID] = sample
	return sample.ID
}

func (e *memEnv) sampleState(id uint) model.InspectionSample {
	e.mu.Lock()
	defer e.mu.Unlock()
	return *cloneSampleValue(e.samples[id])
}

func (e *memEnv) batchState(id uint) model.ProductionBatch {
	e.mu.Lock()
	defer e.mu.Unlock()
	return *cloneBatchValue(e.batches[id])
}

func (e *memEnv) historyOf(t *testing.T, sampleID uint) []model.InspectionRetest {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.retestCopy(sampleID)
}

func (e *memEnv) auditCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.audits)
}

var testActor = Actor{ID: 9, Name: "检验员甲", RequestID: "req-retest-test"}

func completeRound(t *testing.T, svc InspectionService, sampleID uint, result, measured, notes string) *model.InspectionSample {
	t.Helper()
	sample, err := svc.Complete(context.Background(), testActor, sampleID, dto.CompleteInspectionRequest{
		Result: result, MeasuredValue: measured, Notes: notes,
	})
	if err != nil {
		t.Fatalf("complete (%s) failed: %v", result, err)
	}
	return sample
}

func requireConflict(t *testing.T, err error, message string) {
	t.Helper()
	var apiErr *util.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusConflict {
		t.Fatalf("got %v, want 409 conflict (%s)", err, message)
	}
}

// 首检不合格 → 连续复测失败仍可继续 → 最终合格；每轮都追加到历史末尾。
func TestRetestChainAppendsHistoryAndStaysContinueable(t *testing.T) {
	env := newMemEnv()
	batchID := env.addBatch(t, constants.BatchStatusRunning)
	sampleID := env.addSample(t, batchID)
	svc := env.inspectionService()

	rounds := []struct {
		result   string
		measured string
		notes    string
	}{
		{"fail", "1.30 N/15mm", "首检不合格"},
		{"fail", "1.38 N/15mm", "第一次复测仍不合格"},
		{"fail", "1.44 N/15mm", "第二次复测仍不合格"},
		{"pass", "1.66 N/15mm", "第三次复测合格"},
	}
	for i, round := range rounds {
		sample := completeRound(t, svc, sampleID, round.result, round.measured, round.notes)
		if sample.Result != round.result {
			t.Fatalf("round %d: result = %s, want %s", i+1, sample.Result, round.result)
		}
		wantStatus := "requested"
		if i == len(rounds)-1 {
			wantStatus = "completed"
		}
		if sample.RetestStatus != wantStatus {
			t.Fatalf("round %d (%s): retestStatus = %s, want %s", i+1, round.result, sample.RetestStatus, wantStatus)
		}
		if len(sample.Retests) != i+1 {
			t.Fatalf("round %d: history length = %d, want %d", i+1, len(sample.Retests), i+1)
		}
	}

	history := env.historyOf(t, sampleID)
	if len(history) != len(rounds) {
		t.Fatalf("history length = %d, want %d", len(history), len(rounds))
	}
	for i, record := range history {
		if record.Sequence != i+1 {
			t.Errorf("record %d: sequence = %d", i, record.Sequence)
		}
		wantRound := "retest"
		if i == 0 {
			wantRound = "initial"
		}
		if record.Round != wantRound {
			t.Errorf("record %d: round = %s, want %s", i, record.Round, wantRound)
		}
		if record.Result != rounds[i].result || record.MeasuredValue != rounds[i].measured || record.Notes != rounds[i].notes {
			t.Errorf("record %d: got (%s, %s, %s), want (%s, %s, %s)", i,
				record.Result, record.MeasuredValue, record.Notes,
				rounds[i].result, rounds[i].measured, rounds[i].notes)
		}
		if record.InspectorName != testActor.Name || record.InspectedAt == nil {
			t.Errorf("record %d: inspector/timestamp not preserved: %+v", i, record)
		}
	}
	latest := env.sampleState(sampleID)
	if latest.Result != "pass" || latest.MeasuredValue != "1.66 N/15mm" || latest.RetestStatus != "completed" {
		t.Fatalf("latest sample state = (%s, %s, %s), want (pass, 1.66 N/15mm, completed)",
			latest.Result, latest.MeasuredValue, latest.RetestStatus)
	}
}

// 最新结论不合格时放行被阻断；连续复测合格后放行成功，较早不合格记录仅用于追溯。
func TestReleaseFollowsLatestConclusion(t *testing.T) {
	env := newMemEnv()
	batchID := env.addBatch(t, constants.BatchStatusRunning)
	sampleID := env.addSample(t, batchID)
	inspections := env.inspectionService()
	releases := env.releaseService()
	ctx := context.Background()
	decide := func() error {
		_, err := releases.Decide(ctx, testActor, dto.CreateReleaseDecisionRequest{
			ProductionBatchID: batchID, Decision: constants.DecisionRelease, Reason: "验证放行规则",
		})
		return err
	}

	completeRound(t, inspections, sampleID, "fail", "1.31 N/15mm", "首检不合格")
	requireConflict(t, decide(), "latest conclusion fail must block release")
	completeRound(t, inspections, sampleID, "fail", "1.42 N/15mm", "复测仍不合格")
	requireConflict(t, decide(), "retest fail must keep blocking release")
	completeRound(t, inspections, sampleID, "pass", "1.63 N/15mm", "复测合格")

	decision, err := releases.Decide(ctx, testActor, dto.CreateReleaseDecisionRequest{
		ProductionBatchID: batchID, Decision: constants.DecisionRelease, Reason: "最新复测合格，予以放行",
	})
	if err != nil {
		t.Fatalf("release after passing retest failed: %v", err)
	}
	if decision.Decision != constants.DecisionRelease {
		t.Fatalf("decision = %s, want release", decision.Decision)
	}
	if got := env.batchState(batchID).Status; got != constants.BatchStatusReleased {
		t.Fatalf("batch status = %s, want released", got)
	}
	// 较早的不合格记录仍然完整保留，仅用于追溯。
	history := env.historyOf(t, sampleID)
	if len(history) != 3 || history[0].Result != "fail" || history[1].Result != "fail" || history[2].Result != "pass" {
		t.Fatalf("traceability history broken: %+v", history)
	}
	requireConflict(t, decide(), "released batch cannot be decided again")
}

// 已完成（且不在待复测）的样本重复提交被拒绝。
func TestDuplicateCompleteRejected(t *testing.T) {
	env := newMemEnv()
	batchID := env.addBatch(t, constants.BatchStatusRunning)
	svc := env.inspectionService()

	passed := env.addSample(t, batchID)
	completeRound(t, svc, passed, "pass", "1.70 N/15mm", "首检合格")
	_, err := svc.Complete(context.Background(), testActor, passed, dto.CompleteInspectionRequest{
		Result: "pass", MeasuredValue: "1.71 N/15mm",
	})
	requireConflict(t, err, "completed sample cannot be completed again")

	retested := env.addSample(t, batchID)
	completeRound(t, svc, retested, "fail", "1.35 N/15mm", "首检不合格")
	completeRound(t, svc, retested, "pass", "1.62 N/15mm", "复测合格")
	_, err = svc.Complete(context.Background(), testActor, retested, dto.CompleteInspectionRequest{
		Result: "pass", MeasuredValue: "1.64 N/15mm",
	})
	requireConflict(t, err, "sample with completed retest cannot be completed again")

	if got := env.historyOf(t, passed); len(got) != 1 {
		t.Fatalf("passed sample history = %d, want 1", len(got))
	}
	if got := env.historyOf(t, retested); len(got) != 2 {
		t.Fatalf("retested sample history = %d, want 2", len(got))
	}
}

// 并发完成同一复测：只有一个请求成功，历史只追加一条。
func TestConcurrentCompleteKeepsSingleHistory(t *testing.T) {
	env := newMemEnv()
	batchID := env.addBatch(t, constants.BatchStatusRunning)
	sampleID := env.addSample(t, batchID)
	svc := env.inspectionService()
	completeRound(t, svc, sampleID, "fail", "1.33 N/15mm", "首检不合格，等待复测")

	const workers = 8
	var wg sync.WaitGroup
	errs := make([]error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = svc.Complete(context.Background(), testActor, sampleID, dto.CompleteInspectionRequest{
				Result: "pass", MeasuredValue: "1.65 N/15mm", Notes: "复测合格",
			})
		}(i)
	}
	wg.Wait()

	var succeeded, conflicts int
	for _, err := range errs {
		if err == nil {
			succeeded++
			continue
		}
		var apiErr *util.APIError
		if !errors.As(err, &apiErr) || apiErr.Status != http.StatusConflict {
			t.Fatalf("unexpected concurrent error: %v", err)
		}
		conflicts++
	}
	if succeeded != 1 || conflicts != workers-1 {
		t.Fatalf("concurrent complete: %d succeeded, %d conflicts; want 1 and %d", succeeded, conflicts, workers-1)
	}
	history := env.historyOf(t, sampleID)
	if len(history) != 2 {
		t.Fatalf("history length = %d, want 2 (initial + single concurrent retest)", len(history))
	}
	if history[1].Round != "retest" || history[1].Result != "pass" || history[1].Sequence != 2 {
		t.Fatalf("unexpected retest record: %+v", history[1])
	}
	if got := env.sampleState(sampleID); got.Result != "pass" || got.RetestStatus != "completed" {
		t.Fatalf("final state = (%s, %s), want (pass, completed)", got.Result, got.RetestStatus)
	}
}

// 历史写入失败时，样本状态、既有历史和审计记录一起回滚。
func TestCompleteRollsBackWhenRetestHistoryFails(t *testing.T) {
	historyFailure := errors.New("retest history store unavailable")

	t.Run("first inspection", func(t *testing.T) {
		env := newMemEnv()
		batchID := env.addBatch(t, constants.BatchStatusRunning)
		sampleID := env.addSample(t, batchID)
		retestRepo := &memRetestRepo{env: env, createErr: historyFailure}
		svc := NewInspectionService(&memInspectionRepo{env: env}, retestRepo, &memBatchRepo{env: env}, memAudit{env: env}, &memTransactor{env: env})

		_, err := svc.Complete(context.Background(), testActor, sampleID, dto.CompleteInspectionRequest{
			Result: "fail", MeasuredValue: "1.30 N/15mm", Notes: "首检不合格",
		})
		if !errors.Is(err, historyFailure) {
			t.Fatalf("got %v, want injected history failure", err)
		}
		sample := env.sampleState(sampleID)
		if sample.Result != "pending" || sample.MeasuredValue != "" || sample.InspectedAt != nil || sample.InspectorName != "" {
			t.Fatalf("sample state leaked from rolled back transaction: %+v", sample)
		}
		if got := env.historyOf(t, sampleID); len(got) != 0 {
			t.Fatalf("history length = %d, want 0 after rollback", len(got))
		}
		if got := env.auditCount(); got != 0 {
			t.Fatalf("audit records = %d, want 0 after rollback", got)
		}
	})

	t.Run("retest keeps earlier rounds", func(t *testing.T) {
		env := newMemEnv()
		batchID := env.addBatch(t, constants.BatchStatusRunning)
		sampleID := env.addSample(t, batchID)
		completeRound(t, env.inspectionService(), sampleID, "fail", "1.30 N/15mm", "首检不合格")

		retestRepo := &memRetestRepo{env: env, createErr: historyFailure}
		svc := NewInspectionService(&memInspectionRepo{env: env}, retestRepo, &memBatchRepo{env: env}, memAudit{env: env}, &memTransactor{env: env})
		_, err := svc.Complete(context.Background(), testActor, sampleID, dto.CompleteInspectionRequest{
			Result: "pass", MeasuredValue: "1.66 N/15mm", Notes: "复测合格",
		})
		if !errors.Is(err, historyFailure) {
			t.Fatalf("got %v, want injected history failure", err)
		}
		sample := env.sampleState(sampleID)
		if sample.Result != "fail" || sample.RetestStatus != "requested" || sample.MeasuredValue != "1.30 N/15mm" {
			t.Fatalf("sample state leaked from rolled back retest: %+v", sample)
		}
		history := env.historyOf(t, sampleID)
		if len(history) != 1 || history[0].Round != "initial" || history[0].Result != "fail" {
			t.Fatalf("earlier history corrupted by rollback: %+v", history)
		}
		if got := env.auditCount(); got != 1 {
			t.Fatalf("audit records = %d, want only the first completed round", got)
		}
	})
}

// 不同检验员接力复测时，每一轮的检验人和时间都按发生顺序各自保留。
func TestRetestHistoryPreservesPerRoundInspectorAndOrder(t *testing.T) {
	env := newMemEnv()
	batchID := env.addBatch(t, constants.BatchStatusRunning)
	sampleID := env.addSample(t, batchID)
	svc := env.inspectionService()
	another := Actor{ID: 10, Name: "检验员乙", RequestID: "req-retest-test-2"}

	completeRound(t, svc, sampleID, "fail", "1.36 N/15mm", "首检不合格")
	sample, err := svc.Complete(context.Background(), another, sampleID, dto.CompleteInspectionRequest{
		Result: "pass", MeasuredValue: "1.60 N/15mm", Notes: "复测合格",
	})
	if err != nil {
		t.Fatalf("retest by another inspector failed: %v", err)
	}
	if sample.InspectorName != another.Name {
		t.Fatalf("latest inspector = %s, want %s", sample.InspectorName, another.Name)
	}
	history := env.historyOf(t, sampleID)
	if len(history) != 2 || history[0].InspectorName != testActor.Name || history[1].InspectorName != another.Name {
		t.Fatalf("per-round inspector not preserved: %+v", history)
	}
	if history[0].InspectedAt == nil || history[1].InspectedAt == nil ||
		history[1].InspectedAt.Before(*history[0].InspectedAt) {
		t.Fatalf("history timestamps out of order: %+v", history)
	}
}
