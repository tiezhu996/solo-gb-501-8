package model

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var sampleCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{2,49}$`)

type InspectionSample struct {
	Base
	ProductionBatchID uint            `gorm:"index;not null" json:"productionBatchId"`
	ProductionBatch   ProductionBatch `json:"productionBatch,omitempty"`
	SampleCode        string          `gorm:"size:50;uniqueIndex;not null" json:"sampleCode"`
	SamplingPosition  string          `gorm:"size:120;not null" json:"samplingPosition"`
	InspectionItem    string          `gorm:"size:120;not null" json:"inspectionItem"`
	Result            string          `gorm:"size:20;index;not null;default:'pending'" json:"result"`
	MeasuredValue     string          `gorm:"size:100" json:"measuredValue"`
	AcceptanceRange   string          `gorm:"size:100" json:"acceptanceRange"`
	RetestStatus      string          `gorm:"size:20;not null;default:'none'" json:"retestStatus"`
	InspectorID       uint            `gorm:"index" json:"inspectorId"`
	InspectorName     string          `gorm:"size:100" json:"inspectorName"`
	InspectedAt       *time.Time      `json:"inspectedAt"`
	Notes             string          `gorm:"size:1000" json:"notes"`
}

func (s *InspectionSample) Normalize() {
	s.SampleCode = strings.ToUpper(strings.TrimSpace(s.SampleCode))
	s.SamplingPosition = strings.TrimSpace(s.SamplingPosition)
	s.InspectionItem = strings.TrimSpace(s.InspectionItem)
	s.Result = strings.ToLower(strings.TrimSpace(s.Result))
	s.MeasuredValue = strings.TrimSpace(s.MeasuredValue)
	s.AcceptanceRange = strings.TrimSpace(s.AcceptanceRange)
	s.RetestStatus = strings.ToLower(strings.TrimSpace(s.RetestStatus))
	s.InspectorName = strings.TrimSpace(s.InspectorName)
	s.Notes = strings.TrimSpace(s.Notes)
}

func (s InspectionSample) ValidateDefinition() error {
	if s.ProductionBatchID == 0 {
		return fmt.Errorf("production batch is required")
	}
	if !sampleCodePattern.MatchString(s.SampleCode) {
		return fmt.Errorf("sample code must contain 3-50 uppercase letters, numbers or hyphens")
	}
	if s.SamplingPosition == "" || len([]rune(s.SamplingPosition)) > 120 {
		return fmt.Errorf("sampling position must contain 1-120 characters")
	}
	if s.InspectionItem == "" || len([]rune(s.InspectionItem)) > 120 {
		return fmt.Errorf("inspection item must contain 1-120 characters")
	}
	if s.AcceptanceRange == "" || len([]rune(s.AcceptanceRange)) > 100 {
		return fmt.Errorf("acceptance range must contain 1-100 characters")
	}
	if len([]rune(s.Notes)) > 1000 {
		return fmt.Errorf("notes cannot exceed 1000 characters")
	}
	return s.ValidateState()
}

func (s InspectionSample) ValidateState() error {
	switch s.Result {
	case "pending", "pass", "fail":
	default:
		return fmt.Errorf("unsupported inspection result: %s", s.Result)
	}
	switch s.RetestStatus {
	case "none", "requested", "completed":
	default:
		return fmt.Errorf("unsupported retest status: %s", s.RetestStatus)
	}
	if s.Result != "pending" && s.MeasuredValue == "" {
		return fmt.Errorf("measured value is required for a completed inspection")
	}
	if s.Result == "pending" && s.InspectedAt != nil {
		return fmt.Errorf("a pending inspection cannot have an inspection timestamp")
	}
	if s.RetestStatus == "requested" && s.Result != "fail" {
		return fmt.Errorf("retest can only be requested for a failed result")
	}
	return nil
}

func (s InspectionSample) Completed() bool {
	return s.Result == "pass" || s.Result == "fail"
}

func (s InspectionSample) BlocksRelease() bool {
	return s.Result != "pass" || s.RetestStatus == "requested"
}
