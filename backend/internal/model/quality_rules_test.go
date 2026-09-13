package model

import (
	"testing"
	"time"

	"sterile-packaging-release-control/backend/internal/constants"
)

func validBatch() ProductionBatch {
	return ProductionBatch{
		BatchNo: "B20260822-TEST", Specification: "无菌屏障袋", Status: constants.BatchStatusRunning,
		ResponsibleTeam: "验证班", PackagingLineID: 1, PlannedQuantity: 1000, ProducedQuantity: 800,
	}
}

func TestBatchReadyForRelease(t *testing.T) {
	batch := validBatch()
	if ready, _ := batch.ReadyForRelease(); ready {
		t.Fatal("batch without inspections cannot release")
	}
	batch.Inspections = []InspectionSample{{Result: "pass", RetestStatus: "none"}}
	if ready, reason := batch.ReadyForRelease(); !ready {
		t.Fatalf("passed batch should release: %s", reason)
	}
	batch.Inspections = append(batch.Inspections, InspectionSample{Result: "fail", RetestStatus: "requested"})
	if ready, _ := batch.ReadyForRelease(); ready {
		t.Fatal("failed batch cannot release")
	}
}

func TestInspectionValidation(t *testing.T) {
	now := time.Now()
	sample := InspectionSample{
		ProductionBatchID: 1, SampleCode: "SAMPLE-001", SamplingPosition: "中段",
		InspectionItem: "热封强度", Result: "pass", MeasuredValue: "1.7 N",
		AcceptanceRange: ">= 1.5 N", RetestStatus: "none", InspectedAt: &now,
	}
	if err := sample.ValidateDefinition(); err != nil {
		t.Fatalf("valid sample rejected: %v", err)
	}
	sample.Result = "pending"
	if err := sample.ValidateDefinition(); err == nil {
		t.Fatal("pending sample with timestamp must fail")
	}
}

func TestNextRetestStatus(t *testing.T) {
	cases := []struct {
		name          string
		previous      string
		result        string
		requestRetest bool
		want          string
	}{
		{"initial pass needs no retest", "none", "pass", false, "none"},
		{"initial fail waits for retest", "none", "fail", false, "requested"},
		{"retest pass completes retest", "requested", "pass", false, "completed"},
		{"retest fail keeps waiting for another retest", "requested", "fail", false, "requested"},
		{"explicit retest request wins", "requested", "pass", true, "requested"},
	}
	for _, tc := range cases {
		if got := NextRetestStatus(tc.previous, tc.result, tc.requestRetest); got != tc.want {
			t.Errorf("%s: NextRetestStatus(%q, %q, %v) = %q, want %q", tc.name, tc.previous, tc.result, tc.requestRetest, got, tc.want)
		}
	}
}
