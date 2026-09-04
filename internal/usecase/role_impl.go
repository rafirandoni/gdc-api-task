package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog"

	"api-task/internal/domain"
	"api-task/internal/domain/entity"
	"api-task/internal/domain/repository"
)

type roleUsecase struct {
	users  repository.UserRepository
	roles  repository.RoleRepository
	logger zerolog.Logger
}

func NewRoleUsecase(users repository.UserRepository, roles repository.RoleRepository, logger zerolog.Logger) RoleUsecase {
	return &roleUsecase{users: users, roles: roles, logger: logger}
}

func (u *roleUsecase) ListRoles(ctx context.Context) ([]*entity.Role, error) {
	roles, err := u.roles.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	return roles, nil
}

func (u *roleUsecase) GrantRole(ctx context.Context, userID int64, roleLabel string) error {
	role, err := u.resolveActiveRole(ctx, roleLabel)
	if err != nil {
		return err
	}
	if _, err := u.users.GetByID(ctx, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("grant role: %w", domain.ErrNotFound)
		}
		return fmt.Errorf("grant role: verify user: %w", err)
	}

	if err := u.roles.AssignRole(ctx, userID, role.ID); err != nil {
		return fmt.Errorf("grant role: %w", err)
	}

	u.logger.Info().Int64("user_id", userID).Str("role", role.Label).Msg("role granted")
	return nil
}

func (u *roleUsecase) RevokeRole(ctx context.Context, userID int64, roleLabel string) error {
	role, err := u.resolveActiveRole(ctx, roleLabel)
	if err != nil {
		return err
	}
	if _, err := u.users.GetByID(ctx, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("revoke role: %w", domain.ErrNotFound)
		}
		return fmt.Errorf("revoke role: verify user: %w", err)
	}

	if err := u.roles.RevokeRole(ctx, userID, role.ID); err != nil {
		return fmt.Errorf("revoke role: %w", err)
	}

	u.logger.Info().Int64("user_id", userID).Str("role", role.Label).Msg("role revoked")
	return nil
}

func (u *roleUsecase) resolveActiveRole(ctx context.Context, label string) (*entity.Role, error) {
	normalized := strings.ToUpper(strings.TrimSpace(label))
	if normalized == "" {
		return nil, fmt.Errorf("role label is required: %w", domain.ErrInvalidInput)
	}
	role, err := u.roles.GetActiveByLabel(ctx, normalized)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("unknown or inactive role %q: %w", normalized, domain.ErrInvalidInput)
	}
	if err != nil {
		return nil, fmt.Errorf("resolve role: %w", err)
	}
	return role, nil
}
