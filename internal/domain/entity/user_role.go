package entity

import "time"

type UserRole struct {
	ID        int64
	UserID    int64
	RoleID    int64
	Status    string
	CreatedAt time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
}
