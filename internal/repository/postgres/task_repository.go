package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"api-task/internal/domain"
	"api-task/internal/domain/entity"
	"api-task/internal/domain/repository"
)

type taskRepository struct {
	db *bun.DB
}

func NewTaskRepository(db *bun.DB) repository.TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(ctx context.Context, t *entity.Task, idempotentExpiry time.Duration) (*entity.Task, bool, error) {
	var (
		result *entity.Task
		replay bool
	)

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtext(?))", t.IdempotencyKey); err != nil {
			return fmt.Errorf("acquire idempotency lock: %w", err)
		}

		var existing TaskModel
		err := tx.NewSelect().
			Model(&existing).
			Where("t.idempotency_key = ?", t.IdempotencyKey).
			Where("t.created_at >= now() - make_interval(secs => ?)", int64(idempotentExpiry/time.Second)).
			Order("t.created_at DESC").
			Order("t.id DESC").
			Limit(1).
			Scan(ctx)
		switch {
		case err == nil:
			replay = true
			result = toTaskEntity(&existing)
			return nil
		case errors.Is(err, sql.ErrNoRows):
		default:
			return fmt.Errorf("check idempotency: %w", err)
		}

		t.CreatedAt = time.Now()
		if _, err := tx.NewInsert().Model(toTaskModel(t)).Exec(ctx); err != nil {
			return mapInsertError(err, "create task")
		}
		result = t
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return result, replay, nil
}

func (r *taskRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Task, error) {
	var m TaskModel
	err := r.db.NewSelect().Model(&m).Where("t.id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get task by id: %w", domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get task by id: %w", err)
	}
	return toTaskEntity(&m), nil
}

func (r *taskRepository) List(ctx context.Context, filter repository.TaskListFilter) ([]*entity.Task, int, error) {
	query := func(q *bun.SelectQuery) *bun.SelectQuery {
		return applyTaskFilters(q, filter)
	}

	var models []TaskModel
	if err := query(r.db.NewSelect().Model(&models).Order("t.created_at DESC").Order("t.id DESC")).
		Limit(filter.Limit).
		Offset(filter.Offset).
		Scan(ctx); err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}

	total, err := query(r.db.NewSelect().Model((*TaskModel)(nil))).Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	tasks := make([]*entity.Task, 0, len(models))
	for i := range models {
		tasks = append(tasks, toTaskEntity(&models[i]))
	}
	return tasks, total, nil
}

func applyTaskFilters(q *bun.SelectQuery, filter repository.TaskListFilter) *bun.SelectQuery {
	if filter.Status != nil {
		q = q.Where("t.status = ?", string(*filter.Status))
	}
	if filter.AssigneeID != nil {
		q = q.Where("t.user_id = ?", *filter.AssigneeID)
	}
	return q
}

func (r *taskRepository) Update(ctx context.Context, t *entity.Task) error {
	now := time.Now()
	t.UpdatedAt = &now

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		res, err := tx.NewUpdate().Model(toTaskModel(t)).WherePK().Exec(ctx)
		if err != nil {
			return mapInsertError(err, "update task")
		}
		return ensureAffected(res, "update task")
	})
	return err
}

func (r *taskRepository) Assign(ctx context.Context, id uuid.UUID, assigneeID *int64) (*entity.Task, error) {
	var out *entity.Task

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var m TaskModel
		err := tx.NewSelect().Model(&m).Where("t.id = ?", id).For("UPDATE").Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("assign task: %w", domain.ErrNotFound)
		}
		if err != nil {
			return fmt.Errorf("assign task: read: %w", err)
		}

		if sameAssignee(m.UserID, assigneeID) {
			out = toTaskEntity(&m)
			return nil
		}

		if assigneeID != nil {
			var user UserModel
			err := tx.NewSelect().Model(&user).Where("u.id = ?", *assigneeID).Scan(ctx)
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("assign task: assignee does not exist: %w", domain.ErrInvalidInput)
			}
			if err != nil {
				return fmt.Errorf("assign task: verify assignee: %w", err)
			}
		}

		now := time.Now()
		m.UserID = assigneeID
		m.UpdatedAt = &now

		if _, err := tx.NewUpdate().Model(&m).WherePK().Exec(ctx); err != nil {
			return fmt.Errorf("assign task: persist: %w", err)
		}

		logRow := &TaskLogModel{
			ID:          uuid.New(),
			TaskID:      m.ID,
			UserId:      assigneeID,
			Title:       m.Title,
			Description: m.Description,
			Status:      m.Status,
			CreatedAt:   now,
		}
		if _, err := tx.NewInsert().Model(logRow).Exec(ctx); err != nil {
			return fmt.Errorf("assign task: write log: %w", err)
		}

		out = toTaskEntity(&m)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func sameAssignee(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func (r *taskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		res, err := tx.NewDelete().Model(&TaskModel{ID: id}).WherePK().Exec(ctx)
		if err != nil {
			return fmt.Errorf("delete task: %w", err)
		}
		return ensureAffected(res, "delete task")
	})
	return err
}

func toTaskModel(t *entity.Task) *TaskModel {
	return &TaskModel{
		ID:             t.ID,
		UserID:         t.UserID,
		IdempotencyKey: t.IdempotencyKey,
		Title:          t.Title,
		Description:    t.Description,
		Status:         string(t.Status),
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
		DeletedAt:      t.DeletedAt,
	}
}

func toTaskEntity(m *TaskModel) *entity.Task {
	return &entity.Task{
		ID:             m.ID,
		UserID:         m.UserID,
		IdempotencyKey: m.IdempotencyKey,
		Title:          m.Title,
		Description:    m.Description,
		Status:         entity.TaskStatus(m.Status),
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
		DeletedAt:      m.DeletedAt,
	}
}
