package dto

import "sterile-packaging-release-control/backend/internal/constants"

type CreateInspectionRequest struct {
	ProductionBatchID uint   `json:"productionBatchId" binding:"required"`
	SampleCode        string `json:"sampleCode" binding:"required,min=3,max=50"`
	SamplingPosition  string `json:"samplingPosition" binding:"required,max=120"`
	InspectionItem    string `json:"inspectionItem" binding:"required,max=120"`
	AcceptanceRange   string `json:"acceptanceRange" binding:"required,max=100"`
	Notes             string `json:"notes" binding:"max=1000"`
}

type CompleteInspectionRequest struct {
	Result        string `json:"result" binding:"required,oneof=pass fail"`
	MeasuredValue string `json:"measuredValue" binding:"required,max=100"`
	Notes         string `json:"notes" binding:"max=1000"`
	RequestRetest bool   `json:"requestRetest"`
}

type CreateReleaseDecisionRequest struct {
	ProductionBatchID uint                   `json:"productionBatchId" binding:"required"`
	Decision          constants.DecisionType `json:"decision" binding:"required"`
	Reason            string                 `json:"reason" binding:"required,min=5,max=1000"`
}
