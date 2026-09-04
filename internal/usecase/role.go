package usecase

import (
	"context"

	"api-task/internal/domain/entity"
)

type RoleUsecase interface {
	ListRoles(ctx context.Context) ([]*entity.Role, error)
	GrantRole(ctx context.Context, userID int64, roleLabel string) error
	RevokeRole(ctx context.Context, userID int64, roleLabel string) error
}
