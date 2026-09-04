package usecase_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"api-task/internal/domain"
	"api-task/internal/domain/repository"
	"api-task/internal/platform/auth"
	"api-task/internal/testkit"
	"api-task/internal/usecase"
)

const authTestSecret = "0123456789abcdef0123456789abcdef"

type authStack struct {
	auth      usecase.AuthUsecase
	roles     usecase.RoleUsecase
	users     repository.UserRepository
	tokens    domain.TokenManager
	rolesRepo *testkit.InMemoryRoleRepository
}

func newAuthStack(t *testing.T) authStack {
	t.Helper()
	users := testkit.NewInMemoryUserRepository()
	roles := testkit.NewInMemoryRoleRepository()

	tokens, err := auth.NewManager(authTestSecret, time.Hour)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	log := zerolog.New(io.Discard)
	return authStack{
		auth:      usecase.NewAuthUsecase(users, roles, tokens, log),
		roles:     usecase.NewRoleUsecase(users, roles, log),
		users:     users,
		tokens:    tokens,
		rolesRepo: roles,
	}
}

func (s authStack) registerMember(t *testing.T) int64 {
	t.Helper()
	userUC := usecase.NewUserUsecase(s.users, zerolog.New(io.Discard))
	user, err := userUC.Register(context.Background(), "member@example.com", "password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	return user.ID
}

func TestAuthLoginAndRoleLifecycle(t *testing.T) {
	s := newAuthStack(t)
	ctx := context.Background()
	id := s.registerMember(t)

	token, _, err := s.auth.Login(ctx, "member@example.com", "password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if token.Value == "" {
		t.Fatal("token value is empty")
	}
	if _, _, err := s.auth.Login(ctx, "member@example.com", "wrong password"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("wrong password error = %v, want ErrUnauthorized", err)
	}
	if _, _, err := s.auth.Login(ctx, "ghost@example.com", "password"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("unknown email error = %v, want ErrUnauthorized", err)
	}

	if err := s.roles.GrantRole(ctx, id, "ADMIN"); err != nil {
		t.Fatalf("GrantRole() error = %v", err)
	}
	if err := s.roles.GrantRole(ctx, id, "ADMIN"); err != nil {
		t.Fatalf("idempotent GrantRole() error = %v", err)
	}

	token, _, err = s.auth.Login(ctx, "member@example.com", "password")
	if err != nil {
		t.Fatalf("Login() after grant error = %v", err)
	}
	accessRights, err := s.tokens.Verify(ctx, token.Value)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !accessRights.HasRole(domain.RoleAdmin) {
		t.Errorf("roles after grant = %v, want ADMIN", accessRights.Roles)
	}

	if err := s.roles.RevokeRole(ctx, id, "ADMIN"); err != nil {
		t.Fatalf("RevokeRole() error = %v", err)
	}
	if err := s.roles.RevokeRole(ctx, id, "ADMIN"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("double revoke error = %v, want ErrNotFound", err)
	}
}

func TestRoleGrantValidation(t *testing.T) {
	s := newAuthStack(t)
	ctx := context.Background()
	id := s.registerMember(t)

	if err := s.roles.GrantRole(ctx, id, "SUPERUSER"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("unknown role error = %v, want ErrInvalidInput", err)
	}
	if err := s.roles.GrantRole(ctx, 999999, "ADMIN"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("missing user error = %v, want ErrNotFound", err)
	}
	if err := s.roles.RevokeRole(ctx, id, "ADMIN"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("revoke unassigned error = %v, want ErrNotFound", err)
	}
}
