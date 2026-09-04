package usecase

import (
	"context"

	"github.com/google/uuid"

	"api-task/internal/domain/entity"
	"api-task/internal/domain/repository"
)

type TaskUsecase interface {
	Create(ctx context.Context, idempotencyKey, title string, description *string, status *entity.TaskStatus) (*entity.Task, bool, error)
	Get(ctx context.Context, id uuid.UUID) (*entity.Task, error)
	List(ctx context.Context, filter repository.TaskListFilter) ([]*entity.Task, int, error)
	Update(ctx context.Context, id uuid.UUID, title *string, description *string, status *entity.TaskStatus) (*entity.Task, error)
	Assign(ctx context.Context, id uuid.UUID, assigneeID *int64) (*entity.Task, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
