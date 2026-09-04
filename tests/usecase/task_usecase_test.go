package usecase_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"api-task/internal/domain"
	"api-task/internal/domain/entity"
	"api-task/internal/domain/repository"
	"api-task/internal/testkit"
	"api-task/internal/usecase"
)

func newTaskUsecase(t *testing.T) (usecase.TaskUsecase, *testkit.InMemoryTaskRepository, *testkit.InMemoryUserRepository) {
	t.Helper()
	tasks := testkit.NewInMemoryTaskRepository()
	users := testkit.NewInMemoryUserRepository()
	uc := usecase.NewTaskUsecase(tasks, users, zerolog.New(io.Discard))
	return uc, tasks, users
}

func strPtr(s string) *string { return &s }

func generateUUID(t *testing.T) string {
	idemKey, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("uuid generate failed %+v", err)
	}

	return idemKey.String()
}

func TestTaskCreateSuccess(t *testing.T) {
	uc, tasks, _ := newTaskUsecase(t)
	ctx := context.Background()

	idemKey := generateUUID(t)
	status := entity.TaskOnProgress
	desc := strPtr("first task")
	task, replayed, err := uc.Create(ctx, idemKey, "Implement login", desc, &status)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if replayed {
		t.Error("replayed = true, want false for a fresh create")
	}
	if task.ID == uuid.Nil {
		t.Error("task id must be set")
	}
	if task.Title != "Implement login" || task.Description == nil || *task.Description != "first task" {
		t.Errorf("task content = %+v", task)
	}
	if task.Status != entity.TaskOnProgress {
		t.Errorf("status = %q, want ON_PROGRESS", task.Status)
	}
	if task.IdempotencyKey != idemKey {
		t.Errorf("idempotency key = %q", task.IdempotencyKey)
	}
	if task.CreatedAt.IsZero() {
		t.Error("created_at must be set")
	}

	stored, err := tasks.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("task not stored: %v", err)
	}
	if stored.Status != entity.TaskOnProgress {
		t.Errorf("stored status = %q", stored.Status)
	}
}

func TestTaskList(t *testing.T) {
	uc, _, users := newTaskUsecase(t)
	ctx := context.Background()

	uid, err := users.Seed("owner@example.com")
	if err != nil {
		t.Fatalf("Seed() error = %v", err)
	}

	for range 3 {
		if _, _, err := uc.Create(ctx, generateUUID(t), "task", nil, nil); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	done := entity.TaskFinished
	if _, _, err := uc.Create(ctx, generateUUID(t), "finished task", nil, &done); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	all, total, err := uc.List(ctx, repository.TaskListFilter{Page: 1, Limit: 100})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 4 || len(all) != 4 {
		t.Errorf("all: total=%d len=%d, want 4/4", total, len(all))
	}

	finished, total, err := uc.List(ctx, repository.TaskListFilter{Page: 1, Limit: 100, Status: &done})
	if err != nil {
		t.Fatalf("List(status) error = %v", err)
	}
	if total != 1 || finished[0].Status != entity.TaskFinished {
		t.Errorf("status filter: total=%d", total)
	}

	assigned, total, err := uc.List(ctx, repository.TaskListFilter{Page: 1, Limit: 100, AssigneeID: &uid})
	if err != nil {
		t.Fatalf("List(user) error = %v", err)
	}
	if total != 0 || len(assigned) != 0 {
		t.Errorf("user filter before assignment: total=%d, want 0", total)
	}
}

func TestTaskUpdate(t *testing.T) {
	uc, _, _ := newTaskUsecase(t)
	ctx := context.Background()

	task, _, err := uc.Create(ctx, generateUUID(t), "old title", nil, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	desc := strPtr("now described")
	title := "new title"
	status := entity.TaskOnReview
	updated, err := uc.Update(ctx, task.ID, &title, desc, &status)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Title != "new title" || updated.Description == nil || *updated.Description != "now described" || updated.Status != entity.TaskOnReview {
		t.Errorf("updated task = %+v", updated)
	}
	if updated.UpdatedAt == nil {
		t.Error("updated_at should be set after update")
	}
}

func TestTaskAssignWritesLog(t *testing.T) {
	uc, tasks, users := newTaskUsecase(t)
	ctx := context.Background()

	owner, err := users.Seed("owner@example.com")
	if err != nil {
		t.Fatalf("Seed() error = %v", err)
	}
	other, err := users.Seed("other@example.com")
	if err != nil {
		t.Fatalf("Seed() error = %v", err)
	}

	task, _, err := uc.Create(ctx, generateUUID(t), "do the thing", strPtr("details"), nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	assigned, err := uc.Assign(ctx, task.ID, &owner)
	if err != nil {
		t.Fatalf("Assign() error = %v", err)
	}
	if assigned.UserID == nil || *assigned.UserID != owner {
		t.Fatalf("user_id = %v, want %d", assigned.UserID, owner)
	}
	logs := tasks.Logs()
	if len(logs) != 1 {
		t.Fatalf("logs after assign = %d, want 1", len(logs))
	}
	if logs[0].TaskID != task.ID || logs[0].Title != "do the thing" || logs[0].Status != entity.TaskBacklog {
		t.Errorf("log snapshot = %+v", logs[0])
	}
	if logs[0].Description == nil || *logs[0].Description != "details" {
		t.Errorf("log description = %v", logs[0].Description)
	}

	reassign, err := uc.Assign(ctx, task.ID, &owner)
	if err != nil {
		t.Fatalf("re-assign same user error = %v", err)
	}
	if reassign.UserID == nil || *reassign.UserID != owner {
		t.Fatalf("re-assign same user changed user_id = %v", reassign.UserID)
	}
	if got := len(tasks.Logs()); got != 1 {
		t.Errorf("logs after no-op assign = %d, want 1 (no log on unchanged assignee)", got)
	}

	if _, err := uc.Assign(ctx, task.ID, &other); err != nil {
		t.Fatalf("assign to other error = %v", err)
	}
	if got := len(tasks.Logs()); got != 2 {
		t.Errorf("logs after assign to another user = %d, want 2", got)
	}

	unassigned, err := uc.Assign(ctx, task.ID, nil)
	if err != nil {
		t.Fatalf("unassign error = %v", err)
	}
	if unassigned.UserID != nil {
		t.Errorf("user_id after unassign = %v, want nil", unassigned.UserID)
	}
	if got := len(tasks.Logs()); got != 3 {
		t.Errorf("logs after unassign = %d, want 3", got)
	}
}

func TestTaskDelete(t *testing.T) {
	uc, tasks, _ := newTaskUsecase(t)
	ctx := context.Background()

	task, _, err := uc.Create(ctx, generateUUID(t), "title", nil, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := uc.Delete(ctx, task.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := uc.Get(ctx, task.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get after delete error = %v, want ErrNotFound", err)
	}
	if err := uc.Delete(ctx, task.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("double delete error = %v, want ErrNotFound", err)
	}
	if err := uc.Delete(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("delete unknown error = %v, want ErrNotFound", err)
	}
	if got := len(tasks.Logs()); got != 0 {
		t.Errorf("delete should not write logs, got %d", got)
	}
}
