package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"api-task/internal/delivery/http/dto"
	"api-task/internal/delivery/http/response"
	"api-task/internal/domain"
	"api-task/internal/domain/entity"
	"api-task/internal/domain/repository"
	"api-task/internal/usecase"
)

type TaskHandler struct {
	usecase usecase.TaskUsecase
}

func NewTaskHandler(uc usecase.TaskUsecase) *TaskHandler {
	return &TaskHandler{usecase: uc}
}

func (h *TaskHandler) Create(c fiber.Ctx) error {
	var req dto.CreateTaskRequest
	if err := c.Bind().Body(&req); err != nil {
		return fmt.Errorf("bind create task request: %w", domain.ErrInvalidInput)
	}

	if idemKey := c.Get("Idempotency-Key", ""); idemKey != "" {
		req.IdempotencyKey = idemKey
	}

	task, existed, err := h.usecase.Create(c, req.IdempotencyKey, req.Title, req.Description, req.Status)
	if err != nil {
		return err
	}

	status := fiber.StatusCreated
	if existed {
		status = fiber.StatusOK
	}
	return response.Success(c, status, dto.NewTaskResponse(task))
}

func (h *TaskHandler) Get(c fiber.Ctx) error {
	id, err := paramUUID(c)
	if err != nil {
		return err
	}

	task, err := h.usecase.Get(c, id)
	if err != nil {
		return err
	}

	return response.Success(c, fiber.StatusOK, dto.NewTaskResponse(task))
}

func (h *TaskHandler) List(c fiber.Ctx) error {
	page := fiber.Query[int](c, "page", 1)
	limit := fiber.Query[int](c, "limit", 20)

	filter := repository.TaskListFilter{
		Page:  page,
		Limit: limit,
	}

	if title := fiber.Query(c, "title", ""); title != "" {
		filter.Title = &title
	}

	if status := fiber.Query(c, "status", ""); status != "" {
		parsed, ok := entity.ParseTaskStatus(status)
		if !ok {
			return fmt.Errorf("list tasks: invalid status filter: %w", domain.ErrInvalidInput)
		}
		filter.Status = &parsed
	}

	if userID := int64(fiber.Query[int](c, "user_id", 0)); userID > 0 {
		filter.AssigneeID = &userID
	}

	tasks, total, err := h.usecase.List(c, filter)
	if err != nil {
		return err
	}

	items := make([]dto.TaskResponse, len(tasks))
	for i, t := range tasks {
		items[i] = dto.NewTaskResponse(t)
	}

	meta := response.Meta{Page: page, ItemPerPage: limit, Total: total}
	return response.SuccessPage(c, fiber.StatusOK, items, meta)
}

func (h *TaskHandler) Update(c fiber.Ctx) error {
	id, err := paramUUID(c)
	if err != nil {
		return err
	}

	var req dto.UpdateTaskRequest
	if err := c.Bind().Body(&req); err != nil {
		return fmt.Errorf("bind update task request: %w", domain.ErrInvalidInput)
	}

	task, err := h.usecase.Update(c, id, req.Title, req.Description, req.Status)
	if err != nil {
		return err
	}

	return response.Success(c, fiber.StatusOK, dto.NewTaskResponse(task))
}

func (h *TaskHandler) Assign(c fiber.Ctx) error {
	id, err := paramUUID(c)
	if err != nil {
		return err
	}

	var req dto.AssignTaskRequest
	if err := c.Bind().Body(&req); err != nil {
		return fmt.Errorf("bind assign task request: %w", domain.ErrInvalidInput)
	}

	task, err := h.usecase.Assign(c, id, req.UserID)
	if err != nil {
		return err
	}

	return response.Success(c, fiber.StatusOK, dto.NewTaskResponse(task))
}

func (h *TaskHandler) Delete(c fiber.Ctx) error {
	id, err := paramUUID(c)
	if err != nil {
		return err
	}

	if err := h.usecase.Delete(c, id); err != nil {
		return err
	}

	return response.Success(c, fiber.StatusOK, nil)
}

func paramUUID(c fiber.Ctx) (uuid.UUID, error) {
	id, err := uuid.Parse(fiber.Params(c, "id", ""))
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse id: %w", domain.ErrInvalidInput)
	}
	return id, nil
}
