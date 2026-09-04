package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"api-task/internal/domain"
	"api-task/internal/domain/entity"
	"api-task/internal/domain/repository"
)

type authUsecase struct {
	users  repository.UserRepository
	roles  repository.RoleRepository
	tokens domain.TokenManager
	logger zerolog.Logger
}

func NewAuthUsecase(users repository.UserRepository, roles repository.RoleRepository, tokens domain.TokenManager, logger zerolog.Logger) AuthUsecase {
	return &authUsecase{users: users, roles: roles, tokens: tokens, logger: logger}
}

func (u *authUsecase) Login(ctx context.Context, email, password string) (domain.Token, *entity.User, error) {
	email = normalizeEmail(email)
	if !isValidEmail(email) {
		return domain.Token{}, nil, fmt.Errorf("login: invalid email: %w", domain.ErrInvalidInput)
	}

	user, err := u.users.GetByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Token{}, nil, fmt.Errorf("login: %w", domain.ErrUnauthorized)
	}
	if err != nil {
		return domain.Token{}, nil, fmt.Errorf("login: lookup user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return domain.Token{}, nil, fmt.Errorf("login: %w", domain.ErrUnauthorized)
	}

	roles, err := u.roles.RolesByUser(ctx, user.ID)
	if err != nil {
		return domain.Token{}, nil, fmt.Errorf("login: resolve roles: %w", err)
	}

	labels := make([]string, 0, len(roles))
	for _, r := range roles {
		labels = append(labels, r.Label)
	}
	if len(labels) == 0 {
		labels = []string{domain.RoleUser}
	}

	token, err := u.tokens.Issue(ctx, user.ID, labels)
	if err != nil {
		return domain.Token{}, nil, fmt.Errorf("login: issue token: %w", err)
	}

	u.logger.Info().Int64("user_id", user.ID).Msg("user logged in")
	return token, user, nil
}
