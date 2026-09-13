package dto

type CreateBatchRequest struct {
	BatchNo          string `json:"batchNo" binding:"required,min=3,max=50"`
	Specification    string `json:"specification" binding:"required,min=2,max=180"`
	ResponsibleTeam  string `json:"responsibleTeam" binding:"required,min=2,max=80"`
	PackagingLineID  uint   `json:"packagingLineId" binding:"required"`
	PlannedQuantity  int    `json:"plannedQuantity" binding:"required,min=1"`
	ProducedQuantity int    `json:"producedQuantity" binding:"omitempty,min=0"`
}

type UpdateBatchRequest struct {
	Specification    *string `json:"specification" binding:"omitempty,min=2,max=180"`
	ResponsibleTeam  *string `json:"responsibleTeam" binding:"omitempty,min=2,max=80"`
	PackagingLineID  *uint   `json:"packagingLineId"`
	PlannedQuantity  *int    `json:"plannedQuantity" binding:"omitempty,min=1"`
	ProducedQuantity *int    `json:"producedQuantity" binding:"omitempty,min=0"`
	HoldReason       *string `json:"holdReason" binding:"omitempty,max=500"`
}
