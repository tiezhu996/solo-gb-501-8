package model

import (
	"fmt"
	"strings"
	"time"

	"sterile-packaging-release-control/backend/internal/constants"
)

type ReleaseDecision struct {
	Base
	ProductionBatchID uint                   `gorm:"index;not null" json:"productionBatchId"`
	ProductionBatch   ProductionBatch        `json:"productionBatch,omitempty"`
	Decision          constants.DecisionType `gorm:"size:24;index;not null" json:"decision"`
	ApproverID        uint                   `gorm:"index;not null" json:"approverId"`
	ApproverName      string                 `gorm:"size:100;not null" json:"approverName"`
	Reason            string                 `gorm:"size:1000;not null" json:"reason"`
	EffectiveAt       time.Time              `gorm:"not null" json:"effectiveAt"`
	InspectionSummary string                 `gorm:"size:1000" json:"inspectionSummary"`
}

func (d *ReleaseDecision) Normalize() {
	d.ApproverName = strings.TrimSpace(d.ApproverName)
	d.Reason = strings.TrimSpace(d.Reason)
	d.InspectionSummary = strings.TrimSpace(d.InspectionSummary)
}

func (d ReleaseDecision) Validate() error {
	if d.ProductionBatchID == 0 {
		return fmt.Errorf("production batch is required")
	}
	if !d.Decision.Valid() {
		return fmt.Errorf("unsupported release decision: %s", d.Decision)
	}
	if d.ApproverID == 0 || d.ApproverName == "" {
		return fmt.Errorf("approver identity is required")
	}
	if len([]rune(d.ApproverName)) > 100 {
		return fmt.Errorf("approver name cannot exceed 100 characters")
	}
	if len([]rune(d.Reason)) < 5 || len([]rune(d.Reason)) > 1000 {
		return fmt.Errorf("decision reason must contain 5-1000 characters")
	}
	if d.EffectiveAt.IsZero() {
		return fmt.Errorf("effective time is required")
	}
	if len([]rune(d.InspectionSummary)) > 1000 {
		return fmt.Errorf("inspection summary cannot exceed 1000 characters")
	}
	return nil
}
