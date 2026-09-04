package postgres

import (
	"time"

	"github.com/uptrace/bun"
)

type RoleModel struct {
	bun.BaseModel `bun:"table:role,alias:r"`

	ID        int64     `bun:",pk,autoincrement"`
	Label     string    `bun:",notnull"`
	Status    string    `bun:",notnull"`
	CreatedAt time.Time `bun:",notnull"`
	UpdatedAt *time.Time
	DeletedAt *time.Time `bun:",soft_delete,nullzero"`
}
