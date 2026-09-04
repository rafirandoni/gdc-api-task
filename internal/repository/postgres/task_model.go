package postgres

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type TaskModel struct {
	bun.BaseModel `bun:"table:tasks,alias:t"`

	ID             uuid.UUID `bun:"type:uuid,pk"`
	UserID         *int64    `bun:"user_id"`
	IdempotencyKey string    `bun:"idempotency_key,notnull"`
	Title          string    `bun:",notnull"`
	Description    *string
	Status         string    `bun:",notnull"`
	CreatedAt      time.Time `bun:",notnull"`
	UpdatedAt      *time.Time
	DeletedAt      *time.Time `bun:",soft_delete,nullzero"`
}
