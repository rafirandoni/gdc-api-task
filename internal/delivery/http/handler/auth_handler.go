package handler

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	"api-task/internal/delivery/http/dto"
	"api-task/internal/delivery/http/response"
	"api-task/internal/domain"
	"api-task/internal/usecase"
)

type AuthHandler struct {
	usecase usecase.AuthUsecase
}

func NewAuthHandler(uc usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{usecase: uc}
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return fmt.Errorf("bind login request: %w", domain.ErrInvalidInput)
	}

	token, user, err := h.usecase.Login(c, req.Email, req.Password)
	if err != nil {
		return err
	}

	body := dto.LoginResponse{
		Token:     token.Value,
		TokenType: "Bearer",
		ExpiresIn: int64(time.Until(token.ExpiresAt).Seconds()),
		User:      dto.NewUserResponse(user),
	}
	return response.Success(c, fiber.StatusOK, body)
}
