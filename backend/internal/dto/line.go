package dto

type CreateLineRequest struct {
	Code            string `json:"code" binding:"required,min=2,max=32"`
	Name            string `json:"name" binding:"required,min=2,max=100"`
	Team            string `json:"team" binding:"required,min=2,max=80"`
	EquipmentStatus string `json:"equipmentStatus" binding:"required,oneof=ready running maintenance fault"`
	Location        string `json:"location" binding:"max=120"`
}

type UpdateLineRequest struct {
	Name            *string `json:"name" binding:"omitempty,min=2,max=100"`
	Team            *string `json:"team" binding:"omitempty,min=2,max=80"`
	EquipmentStatus *string `json:"equipmentStatus" binding:"omitempty,oneof=ready running maintenance fault"`
	Location        *string `json:"location" binding:"omitempty,max=120"`
	Active          *bool   `json:"active"`
}
