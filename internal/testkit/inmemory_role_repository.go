package testkit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"api-task/internal/domain"
	"api-task/internal/domain/entity"
	"api-task/internal/domain/repository"
)

var _ repository.RoleRepository = (*InMemoryRoleRepository)(nil)

type InMemoryRoleRepository struct {
	mu       sync.Mutex
	byLabel  map[string]entity.Role
	members  map[int64]map[int64]bool
	nextRole int64
}

func NewInMemoryRoleRepository() *InMemoryRoleRepository {
	r := &InMemoryRoleRepository{
		byLabel:  make(map[string]entity.Role),
		members:  make(map[int64]map[int64]bool),
		nextRole: 1,
	}
	r.mustSeed(domain.RoleUser)
	r.mustSeed(domain.RoleAdmin)
	return r
}

func (r *InMemoryRoleRepository) mustSeed(label string) {
	now := time.Now()
	r.byLabel[label] = entity.Role{
		ID:        r.nextRole,
		Label:     label,
		Status:    "ACTIVE",
		CreatedAt: now,
	}
	r.nextRole++
}

func (r *InMemoryRoleRepository) ListActive(ctx context.Context) ([]*entity.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	roles := make([]*entity.Role, 0, len(r.byLabel))
	for _, role := range r.byLabel {
		role := role
		roles = append(roles, &role)
	}
	return roles, nil
}

func (r *InMemoryRoleRepository) GetActiveByLabel(ctx context.Context, label string) (*entity.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	role, ok := r.byLabel[label]
	if !ok || role.Status != "ACTIVE" {
		return nil, fmt.Errorf("get role by label: %w", domain.ErrNotFound)
	}
	return &role, nil
}

func (r *InMemoryRoleRepository) RolesByUser(ctx context.Context, userID int64) ([]*entity.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var roles []*entity.Role
	for _, role := range r.byLabel {
		if r.members[userID][role.ID] {
			role := role
			roles = append(roles, &role)
		}
	}
	return roles, nil
}

func (r *InMemoryRoleRepository) AssignRole(ctx context.Context, userID, roleID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byLabel[roleLabel(r.byLabel, roleID)]; !ok {
		return fmt.Errorf("assign role: %w", domain.ErrNotFound)
	}
	if r.members[userID] == nil {
		r.members[userID] = make(map[int64]bool)
	}
	r.members[userID][roleID] = true
	return nil
}

func (r *InMemoryRoleRepository) RevokeRole(ctx context.Context, userID, roleID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.members[userID][roleID] {
		return fmt.Errorf("revoke role: %w", domain.ErrNotFound)
	}
	delete(r.members[userID], roleID)
	return nil
}

func roleLabel(roles map[string]entity.Role, id int64) string {
	for _, role := range roles {
		if role.ID == id {
			return role.Label
		}
	}
	return ""
}
