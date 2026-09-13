package model

import (
	"fmt"
	"regexp"
	"strings"
)

var lineCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{1,31}$`)

type PackagingLine struct {
	Base
	Code            string            `gorm:"size:32;uniqueIndex;not null" json:"code"`
	Name            string            `gorm:"size:100;not null" json:"name"`
	Team            string            `gorm:"size:80;not null" json:"team"`
	EquipmentStatus string            `gorm:"size:24;not null;default:'ready'" json:"equipmentStatus"`
	Location        string            `gorm:"size:120" json:"location"`
	Active          bool              `gorm:"not null;default:true" json:"active"`
	Batches         []ProductionBatch `json:"batches,omitempty"`
}

func (l *PackagingLine) Normalize() {
	l.Code = strings.ToUpper(strings.TrimSpace(l.Code))
	l.Name = strings.TrimSpace(l.Name)
	l.Team = strings.TrimSpace(l.Team)
	l.EquipmentStatus = strings.ToLower(strings.TrimSpace(l.EquipmentStatus))
	l.Location = strings.TrimSpace(l.Location)
}

func (l PackagingLine) Validate() error {
	if !lineCodePattern.MatchString(l.Code) {
		return fmt.Errorf("line code must contain 2-32 uppercase letters, numbers or hyphens")
	}
	if len([]rune(l.Name)) < 2 || len([]rune(l.Name)) > 100 {
		return fmt.Errorf("line name must contain 2-100 characters")
	}
	if len([]rune(l.Team)) < 2 || len([]rune(l.Team)) > 80 {
		return fmt.Errorf("team must contain 2-80 characters")
	}
	if len([]rune(l.Location)) > 120 {
		return fmt.Errorf("location cannot exceed 120 characters")
	}
	if !l.HasKnownEquipmentStatus() {
		return fmt.Errorf("unsupported equipment status: %s", l.EquipmentStatus)
	}
	return nil
}

func (l PackagingLine) HasKnownEquipmentStatus() bool {
	switch l.EquipmentStatus {
	case "ready", "running", "maintenance", "fault":
		return true
	default:
		return false
	}
}

func (l PackagingLine) CanAcceptBatch() bool {
	if !l.Active {
		return false
	}
	return l.EquipmentStatus == "ready" || l.EquipmentStatus == "running"
}

func (l PackagingLine) OpenBatchCount() int {
	count := 0
	for _, batch := range l.Batches {
		if batch.Status != "released" {
			count++
		}
	}
	return count
}
