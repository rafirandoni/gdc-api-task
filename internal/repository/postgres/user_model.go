package postgres

import (
	"time"

	"github.com/uptrace/bun"
)

type UserModel struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID        int64     `bun:",pk,autoincrement"`
	Email     string    `bun:",notnull,unique"`
	Password  string    `bun:",notnull"`
	CreatedAt time.Time `bun:",notnull"`
	UpdatedAt *time.Time
	DeletedAt *time.Time `bun:",soft_delete,nullzero"`
}
