package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"api-task/internal/domain/entity"
)

type TaskListFilter struct {
	Page       int
	Limit      int
	Offset     int
	Title      *string
	Status     *entity.TaskStatus
	AssigneeID *int64
}

type TaskRepository interface {
	Create(ctx context.Context, t *entity.Task, idempotentExpiry time.Duration) (*entity.Task, bool, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Task, error)
	List(ctx context.Context, filter TaskListFilter) ([]*entity.Task, int, error)
	Update(ctx context.Context, t *entity.Task) error
	Assign(ctx context.Context, id uuid.UUID, assigneeID *int64) (*entity.Task, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
