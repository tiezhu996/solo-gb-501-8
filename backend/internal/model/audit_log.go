package model

import "time"

type AuditLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `gorm:"index" json:"createdAt"`
	RequestID   string    `gorm:"size:64;index;not null" json:"requestId"`
	ActorID     uint      `gorm:"index" json:"actorId"`
	ActorName   string    `gorm:"size:100" json:"actorName"`
	Action      string    `gorm:"size:80;index;not null" json:"action"`
	EntityType  string    `gorm:"size:80;index;not null" json:"entityType"`
	EntityID    uint      `gorm:"index" json:"entityId"`
	BeforeState string    `gorm:"type:text" json:"beforeState"`
	AfterState  string    `gorm:"type:text" json:"afterState"`
	IPAddress   string    `gorm:"size:64" json:"ipAddress"`
}
