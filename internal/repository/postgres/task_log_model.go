package postgres

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type TaskLogModel struct {
	bun.BaseModel `bun:"table:task_logs,alias:tl"`

	ID          uuid.UUID `bun:"type:uuid,pk"`
	TaskID      uuid.UUID `bun:"task_id,type:uuid,notnull"`
	UserId      *int64    `bun:"user_id"`
	Title       string    `bun:",notnull"`
	Description *string
	Status      string    `bun:",notnull"`
	CreatedAt   time.Time `bun:",notnull"`
	UpdatedAt   *time.Time
}
