package testkit

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"api-task/internal/domain"
	"api-task/internal/domain/entity"
	"api-task/internal/domain/repository"
)

var _ repository.TaskRepository = (*InMemoryTaskRepository)(nil)

type InMemoryTaskRepository struct {
	mu      sync.Mutex
	tasks   map[uuid.UUID]entity.Task
	deleted map[uuid.UUID]bool
	logs    []entity.TaskLog
}

func NewInMemoryTaskRepository() *InMemoryTaskRepository {
	return &InMemoryTaskRepository{
		tasks:   make(map[uuid.UUID]entity.Task),
		deleted: make(map[uuid.UUID]bool),
	}
}

func (r *InMemoryTaskRepository) Create(ctx context.Context, t *entity.Task, replayWindow time.Duration) (*entity.Task, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if latest := r.latestByIdempotencyKey(t.IdempotencyKey); latest != nil {
		if !latest.CreatedAt.Before(time.Now().Add(-replayWindow)) {
			return r.cloneTask(latest), true, nil
		}
	}

	t.CreatedAt = time.Now()
	r.tasks[t.ID] = *t
	return r.cloneTask(t), false, nil
}

func (r *InMemoryTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.tasks[id]
	if !ok || r.deleted[id] {
		return nil, fmt.Errorf("get task by id: %w", domain.ErrNotFound)
	}
	return r.cloneTask(&t), nil
}

func (r *InMemoryTaskRepository) List(ctx context.Context, filter repository.TaskListFilter) ([]*entity.Task, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	all := r.sortedTasksLocked()
	active := all[:0]
	for _, t := range all {
		if r.deleted[t.ID] {
			continue
		}
		if filter.Status != nil && t.Status != *filter.Status {
			continue
		}
		if filter.AssigneeID != nil && (t.UserID == nil || *t.UserID != *filter.AssigneeID) {
			continue
		}
		active = append(active, t)
	}
	total := len(active)

	if filter.Offset >= total {
		return []*entity.Task{}, total, nil
	}
	end := filter.Offset + filter.Limit
	if end > total {
		end = total
	}
	page := make([]*entity.Task, 0, end-filter.Offset)
	for _, t := range active[filter.Offset:end] {
		page = append(page, r.cloneTask(&t))
	}
	return page, total, nil
}

func (r *InMemoryTaskRepository) Update(ctx context.Context, t *entity.Task) error {
	now := time.Now()
	t.UpdatedAt = &now

	r.mu.Lock()
	defer r.mu.Unlock()

	stored, ok := r.tasks[t.ID]
	if !ok || r.deleted[t.ID] {
		return fmt.Errorf("update task: %w", domain.ErrNotFound)
	}
	t.CreatedAt = stored.CreatedAt
	r.tasks[t.ID] = *t
	return nil
}

func (r *InMemoryTaskRepository) Assign(ctx context.Context, id uuid.UUID, assigneeID *int64) (*entity.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stored, ok := r.tasks[id]
	if !ok || r.deleted[id] {
		return nil, fmt.Errorf("assign task: %w", domain.ErrNotFound)
	}
	if sameValue(stored.UserID, assigneeID) {
		return r.cloneTask(&stored), nil
	}

	now := time.Now()
	stored.UserID = assigneeID
	stored.UpdatedAt = &now
	r.tasks[id] = stored

	r.logs = append(r.logs, entity.TaskLog{
		ID:          uuid.New(),
		TaskID:      id,
		Title:       stored.Title,
		Description: cloneStringPtr(stored.Description),
		Status:      stored.Status,
		CreatedAt:   now,
	})

	return r.cloneTask(&stored), nil
}

func (r *InMemoryTaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tasks[id]; !ok || r.deleted[id] {
		return fmt.Errorf("delete task: %w", domain.ErrNotFound)
	}
	r.deleted[id] = true
	return nil
}

func (r *InMemoryTaskRepository) Logs() []entity.TaskLog {
	r.mu.Lock()
	defer r.mu.Unlock()

	copies := make([]entity.TaskLog, len(r.logs))
	copy(copies, r.logs)
	return copies
}

func (r *InMemoryTaskRepository) SetCreatedAt(id uuid.UUID, at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if t, ok := r.tasks[id]; ok {
		t.CreatedAt = at
		r.tasks[id] = t
	}
}

func (r *InMemoryTaskRepository) latestByIdempotencyKey(key string) *entity.Task {
	var latest *entity.Task
	for _, t := range r.sortedTasksLocked() {
		if t.IdempotencyKey == key && !r.deleted[t.ID] {
			latest = &t
			break
		}
	}
	return latest
}

func (r *InMemoryTaskRepository) sortedTasksLocked() []entity.Task {
	all := make([]entity.Task, 0, len(r.tasks))
	for _, t := range r.tasks {
		all = append(all, t)
	}
	sort.Slice(all, func(i, j int) bool {
		if !all[i].CreatedAt.Equal(all[j].CreatedAt) {
			return all[i].CreatedAt.After(all[j].CreatedAt)
		}
		return all[i].ID.String() > all[j].ID.String()
	})
	return all
}

func (r *InMemoryTaskRepository) cloneTask(t *entity.Task) *entity.Task {
	copy := *t
	if t.UserID != nil {
		uid := *t.UserID
		copy.UserID = &uid
	}
	if t.Description != nil {
		desc := *t.Description
		copy.Description = &desc
	}
	if t.UpdatedAt != nil {
		at := *t.UpdatedAt
		copy.UpdatedAt = &at
	}
	if t.DeletedAt != nil {
		at := *t.DeletedAt
		copy.DeletedAt = &at
	}
	return &copy
}

func sameValue(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func cloneStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}
