package dto

import (
	"time"

	"api-task/internal/domain/entity"
)

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateUserRequest struct {
	Email    *string `json:"email"`
	Password *string `json:"password"`
}

type UserResponse struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func NewUserResponse(u *entity.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}
