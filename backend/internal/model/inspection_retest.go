package model

import (
	"fmt"
	"strings"
	"time"
)

// InspectionRetest 记录一次已完成的检验轮次：首检（round=initial）或历次复测
// （round=retest）。每次录入结果都会追加一条记录，样本上的同名字段始终保存
// 最新一次结论，放行判定只读取样本的最新结论，历史记录仅用于追溯。
type InspectionRetest struct {
	Base
	InspectionSampleID uint             `gorm:"index;not null;uniqueIndex:uk_retest_sample_sequence" json:"inspectionSampleId"`
	InspectionSample   InspectionSample `json:"-"`
	Round              string           `gorm:"size:20;not null" json:"round"`
	Sequence           int              `gorm:"not null;uniqueIndex:uk_retest_sample_sequence" json:"sequence"`
	Result             string           `gorm:"size:20;not null" json:"result"`
	MeasuredValue      string           `gorm:"size:100;not null" json:"measuredValue"`
	InspectorID        uint             `gorm:"index" json:"inspectorId"`
	InspectorName      string           `gorm:"size:100" json:"inspectorName"`
	InspectedAt        *time.Time       `json:"inspectedAt"`
	Notes              string           `gorm:"size:1000" json:"notes"`
}

func (r *InspectionRetest) Normalize() {
	r.Round = strings.ToLower(strings.TrimSpace(r.Round))
	r.Result = strings.ToLower(strings.TrimSpace(r.Result))
	r.MeasuredValue = strings.TrimSpace(r.MeasuredValue)
	r.InspectorName = strings.TrimSpace(r.InspectorName)
	r.Notes = strings.TrimSpace(r.Notes)
}

func (r InspectionRetest) Validate() error {
	if r.InspectionSampleID == 0 {
		return fmt.Errorf("inspection sample is required")
	}
	switch r.Round {
	case "initial", "retest":
	default:
		return fmt.Errorf("unsupported inspection round: %s", r.Round)
	}
	if r.Sequence < 1 {
		return fmt.Errorf("sequence must be positive")
	}
	switch r.Result {
	case "pass", "fail":
	default:
		return fmt.Errorf("unsupported inspection result: %s", r.Result)
	}
	if r.MeasuredValue == "" || len([]rune(r.MeasuredValue)) > 100 {
		return fmt.Errorf("measured value must contain 1-100 characters")
	}
	if r.InspectedAt == nil {
		return fmt.Errorf("inspection timestamp is required")
	}
	if len([]rune(r.Notes)) > 1000 {
		return fmt.Errorf("notes cannot exceed 1000 characters")
	}
	return nil
}
