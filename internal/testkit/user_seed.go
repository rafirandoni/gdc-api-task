package testkit

import (
	"fmt"
	"time"

	"api-task/internal/domain"
	"api-task/internal/domain/entity"
)

func (r *InMemoryUserRepository) Seed(email string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byEmail[email]; ok {
		return 0, fmt.Errorf("seed user: %w", domain.ErrAlreadyExists)
	}
	u := entity.User{
		ID:        r.nextID,
		Email:     email,
		Password:  "seeded",
		CreatedAt: time.Now(),
	}
	r.nextID++
	r.byID[u.ID] = u
	r.byEmail[email] = u.ID
	return u.ID, nil
}
