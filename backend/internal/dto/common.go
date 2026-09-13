package dto

import "time"

type PageQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Search   string `form:"search"`
}

func (q PageQuery) Normalize() PageQuery {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 10
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	return q
}

type PageResult[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

type TransitionRequest struct {
	Status string `json:"status" binding:"required"`
	Reason string `json:"reason"`
}

type StatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type QualityRisk struct {
	BatchID      uint   `json:"batchId"`
	BatchNo      string `json:"batchNo"`
	Status       string `json:"status"`
	Failed       int64  `json:"failed"`
	Pending      int64  `json:"pending"`
	Retest       int64  `json:"retest"`
	HoldReason   string `json:"holdReason"`
	LastModified string `json:"lastModified"`
}

type QualityOverview struct {
	GeneratedAt        time.Time     `json:"generatedAt"`
	TotalLines         int64         `json:"totalLines"`
	AvailableLines     int64         `json:"availableLines"`
	AttentionLines     int64         `json:"attentionLines"`
	TotalBatches       int64         `json:"totalBatches"`
	AwaitingApproval   int64         `json:"awaitingApproval"`
	BatchStatuses      []StatusCount `json:"batchStatuses"`
	TotalInspections   int64         `json:"totalInspections"`
	PassedInspections  int64         `json:"passedInspections"`
	FailedInspections  int64         `json:"failedInspections"`
	PendingInspections int64         `json:"pendingInspections"`
	RequestedRetests   int64         `json:"requestedRetests"`
	DecisionsToday     int64         `json:"decisionsToday"`
	AuditEventsToday   int64         `json:"auditEventsToday"`
	FirstPassYield     float64       `json:"firstPassYield"`
	Risks              []QualityRisk `json:"risks"`
}

func (o QualityOverview) QualitySignal() string {
	if o.FailedInspections > 0 || o.RequestedRetests > 0 {
		return "attention"
	}
	if o.PendingInspections > 0 || o.AttentionLines > 0 {
		return "monitor"
	}
	return "stable"
}

func (r QualityRisk) Severity() string {
	if r.Failed > 0 || r.Retest > 0 {
		return "high"
	}
	if r.Pending > 0 || r.Status == "hold" {
		return "medium"
	}
	return "low"
}
