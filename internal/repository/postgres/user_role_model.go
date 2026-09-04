package postgres

import (
	"time"

	"github.com/uptrace/bun"
)

type UserRoleModel struct {
	bun.BaseModel `bun:"table:user_role,alias:ur"`

	ID        int64     `bun:",pk,autoincrement"`
	UserID    int64     `bun:"user_id,notnull"`
	RoleID    int64     `bun:"role_id,notnull"`
	Status    string    `bun:",notnull"`
	CreatedAt time.Time `bun:",notnull"`
	UpdatedAt *time.Time
	DeletedAt *time.Time `bun:",soft_delete,nullzero"`
}
