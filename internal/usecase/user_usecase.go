package usecase

import (
	"context"

	"api-task/internal/domain/entity"
)

type UserUsecase interface {
	Register(ctx context.Context, email, password string) (*entity.User, error)
	GetProfile(ctx context.Context, id int64) (*entity.User, error)
	ListUsers(ctx context.Context, page, limit int) ([]*entity.User, int, error)
	UpdateProfile(ctx context.Context, id int64, email, password *string) (*entity.User, error)
	DeleteAccount(ctx context.Context, id int64) error
}
