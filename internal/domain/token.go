package domain

import (
	"context"
	"time"
)

const (
	RoleAdmin = "ADMIN"
	RoleUser  = "USER"
)

type AccessRights struct {
	UserID int64
	Roles  []string
}

func (p AccessRights) HasRole(role string) bool {
	for _, r := range p.Roles {
		if r == role {
			return true
		}
	}
	return false
}

type Token struct {
	Value     string
	ExpiresAt time.Time
}

type TokenManager interface {
	Issue(ctx context.Context, userID int64, roles []string) (Token, error)
	Verify(ctx context.Context, raw string) (AccessRights, error)
}
