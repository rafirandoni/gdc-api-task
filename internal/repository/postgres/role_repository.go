package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"api-task/internal/domain"
	"api-task/internal/domain/entity"
	"api-task/internal/domain/repository"
)

const (
	RoleStatusActive = "ACTIVE"
	UserRoleActive   = "ACTIVE"
)

type roleRepository struct {
	db *bun.DB
}

func NewRoleRepository(db *bun.DB) repository.RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) ListActive(ctx context.Context) ([]*entity.Role, error) {
	var models []RoleModel
	if err := r.db.NewSelect().
		Model(&models).
		Where("r.status = ?", RoleStatusActive).
		Order("r.label ASC").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	roles := make([]*entity.Role, 0, len(models))
	for i := range models {
		roles = append(roles, toRoleEntity(&models[i]))
	}
	return roles, nil
}

func (r *roleRepository) GetActiveByLabel(ctx context.Context, label string) (*entity.Role, error) {
	var m RoleModel
	err := r.db.NewSelect().
		Model(&m).
		Where("r.label = ?", label).
		Where("r.status = ?", RoleStatusActive).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get role by label: %w", domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get role by label: %w", err)
	}
	return toRoleEntity(&m), nil
}

func (r *roleRepository) RolesByUser(ctx context.Context, userID int64) ([]*entity.Role, error) {
	var models []RoleModel
	err := r.db.NewSelect().
		Model(&models).
		Join("JOIN user_role AS ur ON ur.role_id = r.id AND ur.deleted_at IS NULL").
		Where("ur.user_id = ?", userID).
		Where("ur.status = ?", UserRoleActive).
		Where("r.status = ?", RoleStatusActive).
		Order("r.label ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("roles by user: %w", err)
	}

	roles := make([]*entity.Role, 0, len(models))
	for i := range models {
		roles = append(roles, toRoleEntity(&models[i]))
	}
	return roles, nil
}

func (r *roleRepository) AssignRole(ctx context.Context, userID, roleID int64) error {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		m := &UserRoleModel{
			UserID:    userID,
			RoleID:    roleID,
			Status:    UserRoleActive,
			CreatedAt: time.Now(),
		}
		_, err := tx.NewInsert().
			Model(m).
			On("CONFLICT (user_id, role_id) DO UPDATE SET status = 'ACTIVE', deleted_at = NULL, updated_at = now()").
			Exec(ctx)
		if err != nil {
			return mapInsertError(err, "assign role")
		}
		return nil
	})
	return err
}

func (r *roleRepository) RevokeRole(ctx context.Context, userID, roleID int64) error {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		res, err := tx.NewDelete().
			Model(&UserRoleModel{}).
			Where("ur.user_id = ?", userID).
			Where("ur.role_id = ?", roleID).
			ForceDelete().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("revoke role: %w", err)
		}
		return ensureAffected(res, "revoke role")
	})
	return err
}

func toRoleEntity(m *RoleModel) *entity.Role {
	return &entity.Role{
		ID:        m.ID,
		Label:     m.Label,
		Status:    m.Status,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}
