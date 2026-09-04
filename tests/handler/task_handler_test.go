package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"api-task/internal/delivery/http/dto"
	"api-task/internal/domain"
)

type taskPayload struct {
	ID          string  `json:"id"`
	UserID      *int64  `json:"user_id"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
}

func createTask(t *testing.T, ta *testApp, body string) (int, taskPayload) {
	t.Helper()
	httpRes, resWrapper := ta.do(t, http.MethodPost, "/api/v1/tasks", body)
	var task taskPayload
	if err := json.Unmarshal(resWrapper.Data, &task); err != nil {
		t.Fatalf("decode task: %v (body %s)", err, resWrapper.Data)
	}
	return httpRes.StatusCode, task
}

func createTaskBody(key, title string) string {
	return fmt.Sprintf(`{"idempotency_key":%q,"title":%q}`, key, title)
}

func generateUUID(t *testing.T) string {
	idemKey, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("uuid generate failed %+v", err)
	}

	return idemKey.String()
}

func TestTaskCreateReturnsCreated(t *testing.T) {
	ta := newTestApp(t)

	status, task := createTask(t, ta, fmt.Sprintf(`{"idempotency_key":%q,"title":"Implement login","description":"oauth","status":"ON_PROGRESS"}`, generateUUID(t)))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want 201", status)
	}
	if task.ID == "" || task.Title != "Implement login" {
		t.Errorf("task payload = %+v", task)
	}
	if task.Status != "ON_PROGRESS" {
		t.Errorf("status = %q, want ON_PROGRESS", task.Status)
	}
	if task.Description == nil || *task.Description != "oauth" {
		t.Errorf("description = %v", task.Description)
	}
	if task.UserID != nil {
		t.Errorf("user_id = %v, want null on create", task.UserID)
	}
}

func TestTaskCreateConcurrent(t *testing.T) {
	ta := newTestApp(t)

	const workers = 100
	idemKey := generateUUID(t)
	token := ta.token(t, 1, domain.RoleUser)
	body := fmt.Sprintf(`{"idempotency_key":%q,"title":"Implement login","description":"oauth","status":"ON_PROGRESS"}`, idemKey)

	type result struct {
		status  int
		payload taskPayload
		err     error
	}

	results := make(chan result, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			httpRes, err := ta.App.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
			if err != nil {
				results <- result{err: err}
				return
			}
			defer httpRes.Body.Close()

			var res resWrapper
			if err := json.NewDecoder(httpRes.Body).Decode(&res); err != nil {
				results <- result{err: fmt.Errorf("decode response: %w", err)}
				return
			}
			var task taskPayload
			if err := json.Unmarshal(res.Data, &task); err != nil {
				results <- result{err: fmt.Errorf("decode task payload: %w (body %s)", err, res.Data)}
				return
			}
			results <- result{status: httpRes.StatusCode, payload: task}
		}()
	}
	wg.Wait()
	close(results)

	created := 0
	var storedID string
	for res := range results {
		if res.err != nil {
			t.Fatalf("concurrent create: %v", res.err)
		}
		switch res.status {
		case http.StatusCreated:
			created++
		case http.StatusOK:
		default:
			t.Fatalf("create status = %d, want 201 for the first create or 200 for a replay", res.status)
		}
		if res.payload.Title != "Implement login" {
			t.Errorf("payload title = %q, want the original title", res.payload.Title)
		}
		if storedID == "" {
			storedID = res.payload.ID
		} else if res.payload.ID != storedID {
			t.Errorf("replayed task id = %s, want the original %s", res.payload.ID, storedID)
		}
	}
	if created != 1 {
		t.Fatalf("got %d HTTP 201 responses, want exactly 1 (the rest must replay)", created)
	}

	httpRes, resWrapper := ta.do(t, http.MethodGet, "/api/v1/tasks?limit=100", "")
	if httpRes.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d, want 200", httpRes.StatusCode)
	}

	var tasks []dto.TaskResponse
	if err := json.Unmarshal(resWrapper.Data, &tasks); err != nil {
		t.Fatalf("decode task list: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("stored %d tasks for one idempotency key, want 1", len(tasks))
	}
	if tasks[0].ID.String() != storedID {
		t.Errorf("stored task id = %s, want %s", tasks[0].ID.String(), storedID)
	}
}

func TestTaskGet(t *testing.T) {
	ta := newTestApp(t)
	_, task := createTask(t, ta, createTaskBody(generateUUID(t), "title"))

	httpRes, resWrapper := ta.do(t, http.MethodGet, "/api/v1/tasks/"+task.ID, "")
	if httpRes.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", httpRes.StatusCode)
	}
	if !resWrapper.Success {
		t.Error("success = false")
	}
}

func TestTaskListFilters(t *testing.T) {
	ta := newTestApp(t)

	ownerID, err := ta.Users.Seed("owner@example.com")
	if err != nil {
		t.Fatalf("Seed() error = %v", err)
	}

	_, task := createTask(t, ta, createTaskBody(generateUUID(t), "one"))
	createTask(t, ta, createTaskBody(generateUUID(t), "two"))
	assignBody := fmt.Sprintf(`{"user_id":%d}`, ownerID)
	ta.do(t, http.MethodPost, "/api/v1/tasks/"+task.ID+"/assign", assignBody)

	httpRes, resWrapper := ta.do(t, http.MethodGet, "/api/v1/tasks?status=BACKLOG", "")
	if httpRes.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d", httpRes.StatusCode)
	}
	if resWrapper.Meta == nil || resWrapper.Meta.Total != 2 {
		t.Errorf("backlog meta = %+v, want total 2", resWrapper.Meta)
	}

	httpRes, resWrapper = ta.do(t, http.MethodGet, "/api/v1/tasks?user_id="+strconv.FormatInt(ownerID, 10), "")
	if httpRes.StatusCode != http.StatusOK {
		t.Fatalf("list user = %d", httpRes.StatusCode)
	}
	if resWrapper.Meta == nil || resWrapper.Meta.Total != 1 {
		t.Errorf("owner meta = %+v, want total 1", resWrapper.Meta)
	}
}

func TestTaskUpdate(t *testing.T) {
	ta := newTestApp(t)
	_, task := createTask(t, ta, createTaskBody(generateUUID(t), "before"))

	httpRes, resWrapper := ta.do(t, http.MethodPatch, "/api/v1/tasks/"+task.ID, `{"title":"after","status":"FINISHED"}`)
	if httpRes.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d, want 200 (err=%v)", httpRes.StatusCode, *resWrapper.Error)
	}

	var updated taskPayload
	if err := json.Unmarshal(resWrapper.Data, &updated); err != nil {
		t.Fatalf("decode updated task: %v", err)
	}
	if updated.Title != "after" || updated.Status != "FINISHED" {
		t.Errorf("updated = %+v", updated)
	}
}

func TestTaskDelete(t *testing.T) {
	ta := newTestApp(t)
	_, task := createTask(t, ta, createTaskBody(generateUUID(t), "title"))

	httpRes, _ := ta.do(t, http.MethodDelete, "/api/v1/tasks/"+task.ID, "")
	if httpRes.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d, want 200", httpRes.StatusCode)
	}

	httpRes, _ = ta.do(t, http.MethodGet, "/api/v1/tasks/"+task.ID, "")
	if httpRes.StatusCode != http.StatusNotFound {
		t.Errorf("get after delete status = %d, want 404", httpRes.StatusCode)
	}
}
