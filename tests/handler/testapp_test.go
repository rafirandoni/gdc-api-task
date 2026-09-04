package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"

	"api-task/internal/delivery/http/handler"
	"api-task/internal/delivery/http/middleware"
	"api-task/internal/delivery/http/router"
	"api-task/internal/domain"
	"api-task/internal/platform/auth"
	"api-task/internal/testkit"
	"api-task/internal/usecase"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef"

type resWrapper struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *string         `json:"error"`
	Meta    *meta           `json:"meta"`
}

type meta struct {
	Page        int `json:"page"`
	ItemPerPage int `json:"itemPerPage"`
	Total       int `json:"total"`
}

type testApp struct {
	App   *fiber.App
	Users *testkit.InMemoryUserRepository
	Tasks *testkit.InMemoryTaskRepository
	Roles *testkit.InMemoryRoleRepository
	Token domain.TokenManager
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()

	users := testkit.NewInMemoryUserRepository()
	tasks := testkit.NewInMemoryTaskRepository()
	roles := testkit.NewInMemoryRoleRepository()

	userUC := usecase.NewUserUsecase(users, zerolog.New(io.Discard))
	userHandler := handler.NewUserHandler(userUC)

	taskUC := usecase.NewTaskUsecase(tasks, users, zerolog.New(io.Discard))
	taskHandler := handler.NewTaskHandler(taskUC)

	tokenManager, err := auth.NewManager(testJWTSecret, time.Hour)
	if err != nil {
		t.Fatalf("auth.NewManager() error = %v", err)
	}

	authUC := usecase.NewAuthUsecase(users, roles, tokenManager, zerolog.New(io.Discard))
	authHandler := handler.NewAuthHandler(authUC)

	roleUC := usecase.NewRoleUsecase(users, roles, zerolog.New(io.Discard))
	roleHandler := handler.NewRoleHandler(roleUC)

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.NewErrorHandler(zerolog.New(io.Discard)),
	})
	router.RegisterRoutes(app, router.Handlers{
		Users: userHandler,
		Tasks: taskHandler,
		Auth:  authHandler,
		Roles: roleHandler,
	}, tokenManager)

	return &testApp{App: app, Users: users, Tasks: tasks, Roles: roles, Token: tokenManager}
}

func (ta *testApp) token(t *testing.T, userID int64, roles ...string) string {
	t.Helper()
	token, err := ta.Token.Issue(context.Background(), userID, roles)
	if err != nil {
		t.Fatalf("Token.Issue() error = %v", err)
	}
	return token.Value
}

func (ta *testApp) do(t *testing.T, method, path, body string) (*http.Response, resWrapper) {
	t.Helper()
	return ta.doAs(t, 1, []string{domain.RoleUser}, method, path, body)
}

func (ta *testApp) doAs(t *testing.T, userID int64, roles []string, method, path, body string) (*http.Response, resWrapper) {
	t.Helper()
	return ta.doToken(t, ta.token(t, userID, roles...), method, path, body)
}

func (ta *testApp) doOpen(t *testing.T, method, path, body string) (*http.Response, resWrapper) {
	t.Helper()
	return ta.doToken(t, "", method, path, body)
}

func (ta *testApp) doToken(t *testing.T, token, method, path, body string) (*http.Response, resWrapper) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	httpRes, err := ta.App.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer httpRes.Body.Close()

	var result resWrapper
	if err := json.NewDecoder(httpRes.Body).Decode(&result); err != nil {
		t.Fatalf("decode resWrapper for %s %s: %v", method, path, err)
	}
	return httpRes, result
}
