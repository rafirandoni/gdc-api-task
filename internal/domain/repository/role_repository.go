package repository

import (
	"context"

	"api-task/internal/domain/entity"
)

type RoleRepository interface {
	ListActive(ctx context.Context) ([]*entity.Role, error)
	GetActiveByLabel(ctx context.Context, label string) (*entity.Role, error)
	RolesByUser(ctx context.Context, userID int64) ([]*entity.Role, error)
	AssignRole(ctx context.Context, userID, roleID int64) error
	RevokeRole(ctx context.Context, userID, roleID int64) error
}
