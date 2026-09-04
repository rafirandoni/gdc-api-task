package handler

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"

	"api-task/internal/delivery/http/dto"
	"api-task/internal/delivery/http/middleware"
	"api-task/internal/delivery/http/response"
	"api-task/internal/domain"
	"api-task/internal/usecase"
)

type RoleHandler struct {
	usecase usecase.RoleUsecase
}

func NewRoleHandler(uc usecase.RoleUsecase) *RoleHandler {
	return &RoleHandler{usecase: uc}
}

func (h *RoleHandler) List(c fiber.Ctx) error {
	if err := middleware.RequireAdmin(c); err != nil {
		return err
	}

	roles, err := h.usecase.ListRoles(c)
	if err != nil {
		return err
	}

	items := make([]dto.RoleResponse, 0, len(roles))
	for _, r := range roles {
		items = append(items, dto.NewRoleResponse(r))
	}
	return response.Success(c, fiber.StatusOK, items)
}

func (h *RoleHandler) Grant(c fiber.Ctx) error {
	if err := middleware.RequireAdmin(c); err != nil {
		return err
	}

	userID, err := paramID(c)
	if err != nil {
		return err
	}

	var req dto.GrantRoleRequest
	if err := c.Bind().Body(&req); err != nil {
		return fmt.Errorf("bind grant role request: %w", domain.ErrInvalidInput)
	}

	if err := h.usecase.GrantRole(c, userID, req.Role); err != nil {
		return err
	}
	return response.Success(c, fiber.StatusOK, nil)
}

func (h *RoleHandler) Revoke(c fiber.Ctx) error {
	if err := middleware.RequireAdmin(c); err != nil {
		return err
	}

	userID, err := paramID(c)
	if err != nil {
		return err
	}

	label := strings.TrimSpace(fiber.Params[string](c, "label", ""))
	if label == "" {
		return fmt.Errorf("role label is required: %w", domain.ErrInvalidInput)
	}

	if err := h.usecase.RevokeRole(c, userID, label); err != nil {
		return err
	}
	return response.Success(c, fiber.StatusOK, nil)
}
