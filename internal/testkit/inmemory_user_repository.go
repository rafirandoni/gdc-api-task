package testkit

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"api-task/internal/domain"
	"api-task/internal/domain/entity"
	"api-task/internal/domain/repository"
)

var _ repository.UserRepository = (*InMemoryUserRepository)(nil)

type InMemoryUserRepository struct {
	mu            sync.Mutex
	nextID        int64
	byID          map[int64]entity.User
	byEmail       map[string]int64
	reservedEmail map[string]bool
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		nextID:        1,
		byID:          make(map[int64]entity.User),
		byEmail:       make(map[string]int64),
		reservedEmail: make(map[string]bool),
	}
}

func (r *InMemoryUserRepository) Create(ctx context.Context, u *entity.User) error {
	now := time.Now()
	u.CreatedAt = now

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byEmail[u.Email]; ok || r.reservedEmail[u.Email] {
		return fmt.Errorf("create user: %w", domain.ErrAlreadyExists)
	}
	u.ID = r.nextID
	r.nextID++
	r.byID[u.ID] = *u
	r.byEmail[u.Email] = u.ID
	return nil
}

func (r *InMemoryUserRepository) GetByID(ctx context.Context, id int64) (*entity.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, ok := r.byID[id]
	if !ok {
		return nil, fmt.Errorf("get user by id: %w", domain.ErrNotFound)
	}
	return &u, nil
}

func (r *InMemoryUserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id, ok := r.byEmail[email]
	if !ok {
		return nil, fmt.Errorf("get user by email: %w", domain.ErrNotFound)
	}
	u := r.byID[id]
	return &u, nil
}

func (r *InMemoryUserRepository) List(ctx context.Context, limit, offset int) ([]*entity.User, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	users := make([]*entity.User, 0, len(r.byID))
	for _, u := range r.byID {
		users = append(users, &u)
	}
	total := len(users)

	sort.Slice(users, func(i, j int) bool {
		return users[i].ID > users[j].ID
	})

	if offset >= len(users) {
		return []*entity.User{}, total, nil
	}
	end := offset + limit
	if end > len(users) {
		end = len(users)
	}
	return users[offset:end], total, nil
}

func (r *InMemoryUserRepository) Update(ctx context.Context, u *entity.User) error {
	now := time.Now()
	u.UpdatedAt = &now

	r.mu.Lock()
	defer r.mu.Unlock()

	stored, ok := r.byID[u.ID]
	if !ok {
		return fmt.Errorf("update user: %w", domain.ErrNotFound)
	}

	if owner, exists := r.byEmail[u.Email]; exists && owner != u.ID {
		return fmt.Errorf("update user: %w", domain.ErrAlreadyExists)
	}
	if r.reservedEmail[u.Email] && stored.Email != u.Email {
		return fmt.Errorf("update user: %w", domain.ErrAlreadyExists)
	}

	if stored.Email != u.Email {
		delete(r.byEmail, stored.Email)
		r.byEmail[u.Email] = u.ID
	}

	r.byID[u.ID] = *u
	return nil
}

func (r *InMemoryUserRepository) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, ok := r.byID[id]
	if !ok {
		return fmt.Errorf("delete user: %w", domain.ErrNotFound)
	}
	delete(r.byID, id)
	delete(r.byEmail, u.Email)
	r.reservedEmail[u.Email] = true
	return nil
}
