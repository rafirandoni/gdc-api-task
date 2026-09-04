package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"api-task/internal/domain"
	"api-task/internal/domain/entity"
	"api-task/internal/domain/repository"
)

const (
	maxIdempotencyKeyLen = 150
	maxTitleRunes        = 255
	maxDescriptionRunes  = 100000
	idempotentExpiry     = 24 * time.Hour
)

type taskUsecase struct {
	tasks  repository.TaskRepository
	users  repository.UserRepository
	logger zerolog.Logger
}

func NewTaskUsecase(tasks repository.TaskRepository, users repository.UserRepository, logger zerolog.Logger) TaskUsecase {
	return &taskUsecase{tasks: tasks, users: users, logger: logger}
}

func (u *taskUsecase) Create(ctx context.Context, idempotencyKey, title string, description *string, status *entity.TaskStatus) (*entity.Task, bool, error) {
	uid, err := uuid.NewV7()
	if err != nil {
		return nil, false, fmt.Errorf("create task: failed to generate uuid: %w", err)
	}

	idemKey := strings.TrimSpace(idempotencyKey)
	if idemKey == "" || len(idemKey) > maxIdempotencyKeyLen {
		return nil, false, fmt.Errorf("create task: invalid idempotency_key: %w", domain.ErrInvalidInput)
	}

	_, err = uuid.Parse(idemKey)
	if err != nil {
		return nil, false, fmt.Errorf("create task: invalid idempotency_key format: %w", domain.ErrInvalidInput)
	}

	title = strings.TrimSpace(title)
	if title == "" || utf8.RuneCountInString(title) > maxTitleRunes {
		return nil, false, fmt.Errorf("create task: invalid title: %w", domain.ErrInvalidInput)
	}

	if err := validateDescription(description); err != nil {
		return nil, false, err
	}

	taskStatus := entity.TaskBacklog
	if status != nil {
		taskStatus = *status
	}

	task := &entity.Task{
		ID:             uid,
		IdempotencyKey: idemKey,
		Title:          title,
		Description:    description,
		Status:         taskStatus,
	}

	result, isReplay, err := u.tasks.Create(ctx, task, idempotentExpiry)
	if err != nil {
		return nil, false, fmt.Errorf("create task: %w", err)
	}

	if isReplay {
		u.logger.Info().Str("task_id", result.ID.String()).Str("idempotency_key", idemKey).Msg("task create replayed")
	} else {
		u.logger.Info().Str("task_id", result.ID.String()).Msg("task created")
	}
	return result, isReplay, nil
}

func (u *taskUsecase) Get(ctx context.Context, id uuid.UUID) (*entity.Task, error) {
	task, err := u.tasks.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	return task, nil
}

func (u *taskUsecase) List(ctx context.Context, filter repository.TaskListFilter) ([]*entity.Task, int, error) {
	if filter.Page < 1 {
		return nil, 0, fmt.Errorf("list tasks: page must be >= 1: %w", domain.ErrInvalidInput)
	}

	if filter.Limit < 1 || filter.Limit > 100 {
		return nil, 0, fmt.Errorf("list tasks: limit must be between 1 and 100: %w", domain.ErrInvalidInput)
	}

	if filter.AssigneeID != nil && *filter.AssigneeID < 1 {
		return nil, 0, fmt.Errorf("list tasks: invalid user_id: %w", domain.ErrInvalidInput)
	}

	filter.Offset = (filter.Page - 1) * filter.Limit

	tasks, total, err := u.tasks.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, total, nil
}

func (u *taskUsecase) Update(ctx context.Context, id uuid.UUID, title *string, description *string, status *entity.TaskStatus) (*entity.Task, error) {
	if title == nil && description == nil && status == nil {
		return nil, fmt.Errorf("update task: nothing to update: %w", domain.ErrInvalidInput)
	}

	task, err := u.tasks.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	if title != nil {
		trimmed := strings.TrimSpace(*title)
		if trimmed == "" || utf8.RuneCountInString(trimmed) > maxTitleRunes {
			return nil, fmt.Errorf("update task: invalid title: %w", domain.ErrInvalidInput)
		}
		task.Title = trimmed
	}

	if description != nil {
		if err := validateDescription(description); err != nil {
			return nil, err
		}
		task.Description = description
	}

	if status != nil {
		task.Status = *status
	}

	if err := u.tasks.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	u.logger.Info().Str("task_id", task.ID.String()).Msg("task updated")
	return task, nil
}

func (u *taskUsecase) Assign(ctx context.Context, id uuid.UUID, assigneeID *int64) (*entity.Task, error) {
	if assigneeID != nil {
		if *assigneeID < 1 {
			return nil, fmt.Errorf("assign task: invalid user_id: %w", domain.ErrInvalidInput)
		}

		if _, err := u.users.GetByID(ctx, *assigneeID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, fmt.Errorf("assign task: assignee does not exist: %w", domain.ErrInvalidInput)
			}
			return nil, fmt.Errorf("assign task: verify assignee: %w", err)
		}
	}

	task, err := u.tasks.Assign(ctx, id, assigneeID)
	if err != nil {
		return nil, fmt.Errorf("assign task: %w", err)
	}

	u.logger.Info().Msgf("assign task: send notification to user: %d", assigneeID)

	return task, nil
}

func (u *taskUsecase) Delete(ctx context.Context, id uuid.UUID) error {
	if err := u.tasks.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	u.logger.Info().Str("task_id", id.String()).Msg("task deleted")
	return nil
}

func validateDescription(description *string) error {
	if description != nil && utf8.RuneCountInString(*description) > maxDescriptionRunes {
		return fmt.Errorf("description is too long: %w", domain.ErrInvalidInput)
	}

	return nil
}
