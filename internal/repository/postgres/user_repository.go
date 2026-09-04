package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/driver/pgdriver"

	"api-task/internal/domain"
	"api-task/internal/domain/entity"
	"api-task/internal/domain/repository"
)

type userRepository struct {
	db *bun.DB
}

func NewUserRepository(db *bun.DB) repository.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u *entity.User) error {
	u.CreatedAt = time.Now()

	m := toUserModel(u)
	if err := r.db.NewInsert().Model(m).Returning("id").Scan(ctx); err != nil {
		return mapInsertError(err, "create user")
	}
	u.ID = m.ID
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*entity.User, error) {
	var m UserModel
	err := r.db.NewSelect().Model(&m).Where("u.id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get user by id: %w", domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return toUserEntity(&m), nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	var m UserModel
	err := r.db.NewSelect().Model(&m).Where("u.email = ?", email).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get user by email: %w", domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return toUserEntity(&m), nil
}

func (r *userRepository) List(ctx context.Context, limit, offset int) ([]*entity.User, int, error) {
	var models []UserModel
	if err := r.db.NewSelect().
		Model(&models).
		Order("u.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx); err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}

	total, err := r.db.NewSelect().Model((*UserModel)(nil)).Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	users := make([]*entity.User, 0, len(models))
	for i := range models {
		users = append(users, toUserEntity(&models[i]))
	}
	return users, total, nil
}

func (r *userRepository) Update(ctx context.Context, u *entity.User) error {
	now := time.Now()
	u.UpdatedAt = &now

	res, err := r.db.NewUpdate().Model(toUserModel(u)).WherePK().Exec(ctx)
	if err != nil {
		return mapInsertError(err, "update user")
	}
	if err := ensureAffected(res, "update user"); err != nil {
		return err
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.NewDelete().Model(&UserModel{ID: id}).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if err := ensureAffected(res, "delete user"); err != nil {
		return err
	}
	return nil
}

func ensureAffected(res sql.Result, op string) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if affected == 0 {
		return fmt.Errorf("%s: %w", op, domain.ErrNotFound)
	}
	return nil
}

func mapInsertError(err error, op string) error {
	var pgErr pgdriver.Error
	if errors.As(err, &pgErr) && pgErr.Field('C') == "23505" {
		return fmt.Errorf("%s: %w", op, domain.ErrAlreadyExists)
	}
	return fmt.Errorf("%s: %w", op, err)
}

func toUserModel(u *entity.User) *UserModel {
	return &UserModel{
		ID:        u.ID,
		Email:     u.Email,
		Password:  u.Password,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
		DeletedAt: u.DeletedAt,
	}
}

func toUserEntity(m *UserModel) *entity.User {
	return &entity.User{
		ID:        m.ID,
		Email:     m.Email,
		Password:  m.Password,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}
