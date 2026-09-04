package dto

import (
	"time"

	"github.com/google/uuid"

	"api-task/internal/domain/entity"
)

type CreateTaskRequest struct {
	Title          string             `json:"title"`
	Description    *string            `json:"description"`
	Status         *entity.TaskStatus `json:"status"`
	IdempotencyKey string             `json:"idempotency_key"`
}

type UpdateTaskRequest struct {
	Title       *string            `json:"title"`
	Description *string            `json:"description"`
	Status      *entity.TaskStatus `json:"status"`
}

type AssignTaskRequest struct {
	UserID *int64 `json:"user_id"`
}

type TaskResponse struct {
	ID          uuid.UUID  `json:"id"`
	UserID      *int64     `json:"user_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

func NewTaskResponse(t *entity.Task) TaskResponse {
	return TaskResponse{
		ID:          t.ID,
		UserID:      t.UserID,
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
