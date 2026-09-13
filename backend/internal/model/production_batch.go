package model

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"sterile-packaging-release-control/backend/internal/constants"
)

var batchNumberPattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{2,49}$`)

type ProductionBatch struct {
	Base
	BatchNo          string                `gorm:"size:50;uniqueIndex;not null" json:"batchNo"`
	Specification    string                `gorm:"size:180;not null" json:"specification"`
	Status           constants.BatchStatus `gorm:"size:20;index;not null;default:'draft'" json:"status"`
	ResponsibleTeam  string                `gorm:"size:80;not null" json:"responsibleTeam"`
	PackagingLineID  uint                  `gorm:"index;not null" json:"packagingLineId"`
	PackagingLine    PackagingLine         `json:"packagingLine,omitempty"`
	PlannedQuantity  int                   `gorm:"not null;default:0" json:"plannedQuantity"`
	ProducedQuantity int                   `gorm:"not null;default:0" json:"producedQuantity"`
	StartedAt        *time.Time            `json:"startedAt"`
	CompletedAt      *time.Time            `json:"completedAt"`
	HoldReason       string                `gorm:"size:500" json:"holdReason"`
	Inspections      []InspectionSample    `json:"inspections,omitempty"`
	Decisions        []ReleaseDecision     `json:"decisions,omitempty"`
}

func (b *ProductionBatch) Normalize() {
	b.BatchNo = strings.ToUpper(strings.TrimSpace(b.BatchNo))
	b.Specification = strings.TrimSpace(b.Specification)
	b.ResponsibleTeam = strings.TrimSpace(b.ResponsibleTeam)
	b.HoldReason = strings.TrimSpace(b.HoldReason)
}

func (b ProductionBatch) Validate() error {
	if !batchNumberPattern.MatchString(b.BatchNo) {
		return fmt.Errorf("batch number must contain 3-50 uppercase letters, numbers or hyphens")
	}
	if len([]rune(b.Specification)) < 2 || len([]rune(b.Specification)) > 180 {
		return fmt.Errorf("specification must contain 2-180 characters")
	}
	if len([]rune(b.ResponsibleTeam)) < 2 || len([]rune(b.ResponsibleTeam)) > 80 {
		return fmt.Errorf("responsible team must contain 2-80 characters")
	}
	if !b.Status.Valid() {
		return fmt.Errorf("unsupported batch status: %s", b.Status)
	}
	if b.PackagingLineID == 0 {
		return fmt.Errorf("packaging line is required")
	}
	if b.PlannedQuantity < 1 {
		return fmt.Errorf("planned quantity must be positive")
	}
	if b.ProducedQuantity < 0 {
		return fmt.Errorf("produced quantity cannot be negative")
	}
	if b.ProducedQuantity > b.PlannedQuantity*2 {
		return fmt.Errorf("produced quantity exceeds the allowed deviation")
	}
	if len([]rune(b.HoldReason)) > 500 {
		return fmt.Errorf("hold reason cannot exceed 500 characters")
	}
	if b.Status == constants.BatchStatusHold && b.HoldReason == "" {
		return fmt.Errorf("hold reason is required when a batch is on hold")
	}
	return nil
}

func (b ProductionBatch) CompletionPercent() float64 {
	if b.PlannedQuantity <= 0 {
		return 0
	}
	percent := float64(b.ProducedQuantity) / float64(b.PlannedQuantity) * 100
	if percent > 100 {
		return 100
	}
	return percent
}

func (b ProductionBatch) InspectionSummary() (total, passed, failed, pending, retest int) {
	for _, sample := range b.Inspections {
		total++
		switch sample.Result {
		case "pass":
			passed++
		case "fail":
			failed++
		default:
			pending++
		}
		if sample.RetestStatus == "requested" {
			retest++
		}
	}
	return
}

func (b ProductionBatch) ReadyForRelease() (bool, string) {
	if b.Status == constants.BatchStatusDraft {
		return false, "batch has not started"
	}
	if b.Status == constants.BatchStatusReleased {
		return false, "batch is already released"
	}
	total, _, failed, pending, retest := b.InspectionSummary()
	if total == 0 {
		return false, "at least one inspection is required"
	}
	if pending > 0 {
		return false, "pending inspections remain"
	}
	if failed > 0 {
		return false, "failed inspections remain"
	}
	if retest > 0 {
		return false, "requested retests remain"
	}
	return true, ""
}

func (b ProductionBatch) Mutable() bool {
	return b.Status != constants.BatchStatusReleased
}
