package entity

import (
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID             uuid.UUID
	UserID         *int64
	IdempotencyKey string
	Title          string
	Description    *string
	Status         TaskStatus
	CreatedAt      time.Time
	UpdatedAt      *time.Time
	DeletedAt      *time.Time
}
