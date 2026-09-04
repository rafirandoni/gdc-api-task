package entity

import "time"

type Role struct {
	ID        int64
	Label     string
	Status    string
	CreatedAt time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
}
