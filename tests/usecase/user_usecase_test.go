package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"api-task/internal/domain"
	"api-task/internal/testkit"
	"api-task/internal/usecase"
)

func newUsecase(t *testing.T) (usecase.UserUsecase, *testkit.InMemoryUserRepository) {
	t.Helper()
	repo := testkit.NewInMemoryUserRepository()
	uc := usecase.NewUserUsecase(repo, zerolog.New(io.Discard))
	return uc, repo
}

func TestRegisterSuccess(t *testing.T) {
	uc, repo := newUsecase(t)

	user, err := uc.Register(context.Background(), " Ada@Example.com ", "password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if user.Email != "ada@example.com" {
		t.Errorf("email = %q, want normalized lowercase", user.Email)
	}
	if user.ID < 1 {
		t.Errorf("id = %d, want a positive DB-assigned id", user.ID)
	}
	if user.Password == "password" {
		t.Error("password must be hashed")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("password")); err != nil {
		t.Errorf("stored password does not verify: %v", err)
	}
	if user.CreatedAt.IsZero() {
		t.Error("created_at must be set")
	}
	if user.UpdatedAt != nil {
		t.Error("updated_at should be nil for a fresh user")
	}

	stored, err := repo.GetByEmail(context.Background(), "ada@example.com")
	if err != nil {
		t.Fatalf("user not stored: %v", err)
	}
	if stored.ID != user.ID {
		t.Errorf("stored id = %d, want %d", stored.ID, user.ID)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	uc, _ := newUsecase(t)
	ctx := context.Background()

	if _, err := uc.Register(ctx, "ada@example.com", "password"); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	_, err := uc.Register(ctx, "ada@example.com", "other password")
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("second Register() error = %v, want ErrAlreadyExists", err)
	}
}

func TestRegisterEmailReservedAfterDelete(t *testing.T) {
	uc, _ := newUsecase(t)
	ctx := context.Background()

	user, err := uc.Register(ctx, "ada@example.com", "password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := uc.DeleteAccount(ctx, user.ID); err != nil {
		t.Fatalf("DeleteAccount() error = %v", err)
	}

	_, err = uc.Register(ctx, "ada@example.com", "password")
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("re-register after delete error = %v, want ErrAlreadyExists (email stays reserved)", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
	}{
		{name: "bad email", email: "not-an-email", password: "password"},
		{name: "short password", email: "ada@example.com", password: "short"},
		{name: "oversized password", email: "ada@example.com", password: string(make([]byte, 73))},
	}
	uc, _ := newUsecase(t)
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.Register(ctx, tt.email, tt.password)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("Register() error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestGetProfileNotFound(t *testing.T) {
	uc, _ := newUsecase(t)

	_, err := uc.GetProfile(context.Background(), 999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetProfile() error = %v, want ErrNotFound", err)
	}
}

func TestListUsersPagination(t *testing.T) {
	uc, _ := newUsecase(t)
	ctx := context.Background()

	for i := range 5 {
		_, err := uc.Register(ctx, fmt.Sprintf("user%d@example.com", i), "password")
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	users, total, err := uc.ListUsers(ctx, 1, 2)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(users) != 2 {
		t.Errorf("page size returned %d items, want 2", len(users))
	}

	users2, _, err := uc.ListUsers(ctx, 3, 2)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(users2) != 1 {
		t.Errorf("last page returned %d items, want 1", len(users2))
	}
}

func TestListUsersInvalidPagination(t *testing.T) {
	uc, _ := newUsecase(t)
	ctx := context.Background()

	if _, _, err := uc.ListUsers(ctx, 0, 20); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("page 0 error = %v, want ErrInvalidInput", err)
	}
	if _, _, err := uc.ListUsers(ctx, 1, 0); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("limit 0 error = %v, want ErrInvalidInput", err)
	}
	if _, _, err := uc.ListUsers(ctx, 1, 101); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("limit 101 error = %v, want ErrInvalidInput", err)
	}
}

func TestUpdateProfileEmail(t *testing.T) {
	uc, _ := newUsecase(t)
	ctx := context.Background()

	user, err := uc.Register(ctx, "ada@example.com", "password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	newEmail := "ada.lovelace@example.com"
	updated, err := uc.UpdateProfile(ctx, user.ID, &newEmail, nil)
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if updated.Email != newEmail {
		t.Errorf("email = %q, want %q", updated.Email, newEmail)
	}
	if updated.UpdatedAt == nil {
		t.Error("UpdatedAt should be set after update")
	}
}

func TestUpdateProfilePassword(t *testing.T) {
	uc, _ := newUsecase(t)
	ctx := context.Background()

	user, err := uc.Register(ctx, "ada@example.com", "password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	newPassword := "another password"
	updated, err := uc.UpdateProfile(ctx, user.ID, nil, &newPassword)
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte(newPassword)); err != nil {
		t.Errorf("new password does not verify: %v", err)
	}
}

func TestUpdateProfileEmailConflict(t *testing.T) {
	uc, _ := newUsecase(t)
	ctx := context.Background()

	first, err := uc.Register(ctx, "ada@example.com", "password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if _, err := uc.Register(ctx, "grace@example.com", "password"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	taken := "grace@example.com"
	_, err = uc.UpdateProfile(ctx, first.ID, &taken, nil)
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("UpdateProfile() error = %v, want ErrAlreadyExists", err)
	}
}

func TestUpdateProfileNothingToUpdate(t *testing.T) {
	uc, _ := newUsecase(t)
	ctx := context.Background()

	user, err := uc.Register(ctx, "ada@example.com", "password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	_, err = uc.UpdateProfile(ctx, user.ID, nil, nil)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("UpdateProfile() error = %v, want ErrInvalidInput", err)
	}
}

func TestUpdateProfileInvalidPassword(t *testing.T) {
	uc, _ := newUsecase(t)
	ctx := context.Background()

	user, err := uc.Register(ctx, "ada@example.com", "password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	weak := "short"
	_, err = uc.UpdateProfile(ctx, user.ID, nil, &weak)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("UpdateProfile() error = %v, want ErrInvalidInput", err)
	}
}

func TestDeleteAccountAndUpdateAfterDelete(t *testing.T) {
	uc, _ := newUsecase(t)
	ctx := context.Background()

	user, err := uc.Register(ctx, "ada@example.com", "password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := uc.DeleteAccount(ctx, user.ID); err != nil {
		t.Fatalf("DeleteAccount() error = %v", err)
	}

	if _, err := uc.GetProfile(ctx, user.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetProfile after delete error = %v, want ErrNotFound", err)
	}

	newEmail := "renamed@example.com"
	if _, err := uc.UpdateProfile(ctx, user.ID, &newEmail, nil); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdateProfile after delete error = %v, want ErrNotFound", err)
	}

	if err := uc.DeleteAccount(ctx, user.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("DeleteAccount twice error = %v, want ErrNotFound", err)
	}
}
