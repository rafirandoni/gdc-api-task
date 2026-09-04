package usecase

import (
	"context"

	"api-task/internal/domain"
	"api-task/internal/domain/entity"
)

type AuthUsecase interface {
	Login(ctx context.Context, email, password string) (domain.Token, *entity.User, error)
}
