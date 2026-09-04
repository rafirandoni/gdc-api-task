package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"api-task/internal/domain"
	"api-task/internal/domain/entity"
	"api-task/internal/domain/repository"
)

const (
	maxEmailLen = 254
	minPassword = 8
	maxPassword = 72
)

type userUsecase struct {
	repo   repository.UserRepository
	logger zerolog.Logger
}

func NewUserUsecase(repo repository.UserRepository, logger zerolog.Logger) UserUsecase {
	return &userUsecase{repo: repo, logger: logger}
}

func (u *userUsecase) Register(ctx context.Context, email, password string) (*entity.User, error) {
	email = normalizeEmail(email)

	if err := validateEmailAndPassword(email, password); err != nil {
		return nil, err
	}

	_, err := u.repo.GetByEmail(ctx, email)
	if err == nil {
		return nil, fmt.Errorf("register: %w", domain.ErrAlreadyExists)
	}

	if !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("register: check email: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("register: hash password: %w", err)
	}

	user := &entity.User{
		Email:    email,
		Password: string(hash),
	}

	if err := u.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("register: %w", err)
	}

	u.logger.Info().Int64("user_id", user.ID).Msg("user registered")
	return user, nil
}

func (u *userUsecase) GetProfile(ctx context.Context, id int64) (*entity.User, error) {
	user, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	return user, nil
}

func (u *userUsecase) ListUsers(ctx context.Context, page, limit int) ([]*entity.User, int, error) {
	if page < 1 {
		return nil, 0, fmt.Errorf("list users: page must be >= 1: %w", domain.ErrInvalidInput)
	}
	if limit < 1 || limit > 100 {
		return nil, 0, fmt.Errorf("list users: limit must be between 1 and 100: %w", domain.ErrInvalidInput)
	}

	users, total, err := u.repo.List(ctx, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	return users, total, nil
}

func (u *userUsecase) UpdateProfile(ctx context.Context, id int64, email, password *string) (*entity.User, error) {
	if email == nil && password == nil {
		return nil, fmt.Errorf("update profile: nothing to update: %w", domain.ErrInvalidInput)
	}

	user, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	if email != nil {
		normalizedEmail := normalizeEmail(*email)
		if !isValidEmail(normalizedEmail) {
			return nil, fmt.Errorf("update profile: invalid email: %w", domain.ErrInvalidInput)
		}
		if normalizedEmail != user.Email {
			existing, err := u.repo.GetByEmail(ctx, normalizedEmail)
			if err != nil && !errors.Is(err, domain.ErrNotFound) {
				return nil, fmt.Errorf("update profile: check email: %w", err)
			}
			if existing != nil && existing.ID != id {
				return nil, fmt.Errorf("update profile: %w", domain.ErrAlreadyExists)
			}
			user.Email = normalizedEmail
		}
	}

	if password != nil {
		if err := validatePassword(*password); err != nil {
			return nil, err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("update profile: hash password: %w", err)
		}
		user.Password = string(hash)
	}

	if err := u.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	u.logger.Info().Int64("user_id", user.ID).Msg("user profile updated")
	return user, nil
}

func (u *userUsecase) DeleteAccount(ctx context.Context, id int64) error {
	if err := u.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete account: %w", err)
	}

	u.logger.Info().Int64("user_id", id).Msg("user account deleted")
	return nil
}

func validateEmailAndPassword(email, password string) error {
	if !isValidEmail(email) {
		return fmt.Errorf("register: invalid email: %w", domain.ErrInvalidInput)
	}

	if err := validatePassword(password); err != nil {
		return err
	}

	return nil
}

func validatePassword(password string) error {
	passwordLen := len(password)
	if passwordLen < minPassword || passwordLen > maxPassword {
		return fmt.Errorf("password must be %d-%d characters: %w", minPassword, maxPassword, domain.ErrInvalidInput)
	}
	return nil
}

func isValidEmail(email string) bool {
	if email == "" || len(email) > maxEmailLen {
		return false
	}

	parsed, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}

	return parsed.Address == email
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
