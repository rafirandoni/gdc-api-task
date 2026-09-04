package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v3"

	"api-task/internal/delivery/http/dto"
	"api-task/internal/delivery/http/middleware"
	"api-task/internal/delivery/http/response"
	"api-task/internal/domain"
	"api-task/internal/usecase"
)

type UserHandler struct {
	usecase usecase.UserUsecase
}

func NewUserHandler(uc usecase.UserUsecase) *UserHandler {
	return &UserHandler{usecase: uc}
}

func (h *UserHandler) Register(c fiber.Ctx) error {
	var req dto.CreateUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return fmt.Errorf("bind create user request: %w", domain.ErrInvalidInput)
	}

	user, err := h.usecase.Register(c, req.Email, req.Password)
	if err != nil {
		return err
	}

	return response.Success(c, fiber.StatusCreated, dto.NewUserResponse(user))
}

func (h *UserHandler) GetMe(c fiber.Ctx) error {
	accessRights, ok := middleware.AccessRights(c)
	if !ok {
		return fmt.Errorf("authentication required: %w", domain.ErrUnauthorized)
	}

	user, err := h.usecase.GetProfile(c, accessRights.UserID)
	if err != nil {
		return err
	}

	return response.Success(c, fiber.StatusOK, dto.NewUserResponse(user))
}

func (h *UserHandler) GetProfile(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return err
	}
	if err := middleware.RequireSelfOrAdmin(c, id); err != nil {
		return err
	}

	user, err := h.usecase.GetProfile(c, id)
	if err != nil {
		return err
	}

	return response.Success(c, fiber.StatusOK, dto.NewUserResponse(user))
}

func (h *UserHandler) List(c fiber.Ctx) error {
	if err := middleware.RequireAdmin(c); err != nil {
		return err
	}

	page := fiber.Query(c, "page", 1)
	limit := fiber.Query(c, "limit", 20)

	users, total, err := h.usecase.ListUsers(c, page, limit)
	if err != nil {
		return err
	}

	items := make([]dto.UserResponse, 0, len(users))
	for _, u := range users {
		items = append(items, dto.NewUserResponse(u))
	}

	meta := response.Meta{Page: page, ItemPerPage: limit, Total: total}
	return response.SuccessPage(c, fiber.StatusOK, items, meta)
}

func (h *UserHandler) UpdateProfile(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return err
	}
	if err := middleware.RequireSelfOrAdmin(c, id); err != nil {
		return err
	}

	var req dto.UpdateUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return fmt.Errorf("bind update user request: %w", domain.ErrInvalidInput)
	}

	user, err := h.usecase.UpdateProfile(c, id, req.Email, req.Password)
	if err != nil {
		return err
	}

	return response.Success(c, fiber.StatusOK, dto.NewUserResponse(user))
}

func (h *UserHandler) DeleteAccount(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return err
	}
	if err := middleware.RequireSelfOrAdmin(c, id); err != nil {
		return err
	}

	if err := h.usecase.DeleteAccount(c, id); err != nil {
		return err
	}

	return response.Success(c, fiber.StatusOK, nil)
}

func paramID(c fiber.Ctx) (int64, error) {
	id := int64(fiber.Params[int](c, "id", 0))
	if id < 1 {
		return 0, fmt.Errorf("parse user id: %w", domain.ErrInvalidInput)
	}
	return id, nil
}
