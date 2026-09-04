package auth_test

import (
	"context"
	"testing"
	"time"

	"api-task/internal/domain"
	"api-task/internal/platform/auth"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func newTestManager(t *testing.T, ttl time.Duration) *auth.Manager {
	t.Helper()
	m, err := auth.NewManager(testSecret, ttl)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	return m
}

func TestManagerIssueAndVerify(t *testing.T) {
	m := newTestManager(t, time.Hour)

	token, err := m.Issue(context.Background(), 42, []string{domain.RoleAdmin, domain.RoleUser})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if token.Value == "" {
		t.Fatal("token value is empty")
	}
	if !token.ExpiresAt.After(time.Now()) {
		t.Error("expires_at should be in the future")
	}

	p, err := m.Verify(context.Background(), token.Value)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if p.UserID != 42 {
		t.Errorf("user id = %d, want 42", p.UserID)
	}
	if !p.HasRole(domain.RoleAdmin) || !p.HasRole(domain.RoleUser) {
		t.Errorf("roles = %v", p.Roles)
	}
}
