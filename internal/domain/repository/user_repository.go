package repository

import (
	"context"

	"api-task/internal/domain/entity"
)

type UserRepository interface {
	Create(ctx context.Context, u *entity.User) error
	GetByID(ctx context.Context, id int64) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	List(ctx context.Context, limit, offset int) ([]*entity.User, int, error)
	Update(ctx context.Context, u *entity.User) error
	Delete(ctx context.Context, id int64) error
}
