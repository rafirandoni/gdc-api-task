package entity

import (
	"time"

	"github.com/google/uuid"
)

type TaskLog struct {
	ID          uuid.UUID
	TaskID      uuid.UUID
	UserID      *int64
	Title       string
	Description *string
	Status      TaskStatus
	CreatedAt   time.Time
	UpdatedAt   *time.Time
}
