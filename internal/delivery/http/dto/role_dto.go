package dto

import (
	"time"

	"api-task/internal/domain/entity"
)

type GrantRoleRequest struct {
	Role string `json:"role"`
}

type RoleResponse struct {
	ID        int64     `json:"id"`
	Label     string    `json:"label"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func NewRoleResponse(r *entity.Role) RoleResponse {
	return RoleResponse{
		ID:        r.ID,
		Label:     r.Label,
		Status:    r.Status,
		CreatedAt: r.CreatedAt,
	}
}
