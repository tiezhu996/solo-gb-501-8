package dto

import "sterile-packaging-release-control/backend/internal/constants"

type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=80"`
	Password string `json:"password" binding:"required,min=6,max=128"`
}

type UserView struct {
	ID          uint           `json:"id"`
	Username    string         `json:"username"`
	DisplayName string         `json:"displayName"`
	Role        constants.Role `json:"role"`
	Permissions []string       `json:"permissions"`
}

type LoginResponse struct {
	Token     string   `json:"token"`
	ExpiresAt int64    `json:"expiresAt"`
	User      UserView `json:"user"`
}
