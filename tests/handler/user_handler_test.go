package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"api-task/internal/domain"
)

func registerUser(t *testing.T, ta *testApp, email string) int64 {
	t.Helper()
	body := fmt.Sprintf(`{"email":%q,"password":"password"}`, email)
	httpRes, resWrapper := ta.doOpen(t, http.MethodPost, "/api/v1/users", body)
	if httpRes.StatusCode != http.StatusCreated {
		t.Fatalf("register %s: status = %d", email, httpRes.StatusCode)
	}
	var created struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(resWrapper.Data, &created); err != nil {
		t.Fatalf("decode created user: %v", err)
	}
	return created.ID
}

func TestRegisterReturnsCreated(t *testing.T) {
	ta := newTestApp(t)

	httpRes, resWrapper := ta.doOpen(t, http.MethodPost, "/api/v1/users", `{"email":"ada@example.com","password":"password"}`)
	if httpRes.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", httpRes.StatusCode, resWrapper.Data)
	}
	if !resWrapper.Success {
		t.Error("success = false, want true")
	}
	var user struct {
		ID        int64  `json:"id"`
		Email     string `json:"email"`
		CreatedAt string `json:"created_at"`
	}
	if err := json.Unmarshal(resWrapper.Data, &user); err != nil {
		t.Fatalf("decode user: %v", err)
	}
	if user.ID < 1 || user.Email != "ada@example.com" || user.CreatedAt == "" {
		t.Errorf("unexpected user payload: %+v", user)
	}
}

func TestGetMe(t *testing.T) {
	ta := newTestApp(t)
	id := registerUser(t, ta, "ada@example.com")

	httpRes, resWrapper := ta.doAs(t, id, []string{domain.RoleUser}, http.MethodGet, "/api/v1/users/me", "")
	if httpRes.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", httpRes.StatusCode)
	}
	var user struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(resWrapper.Data, &user); err != nil {
		t.Fatalf("decode user: %v", err)
	}
	if user.ID != id || user.Email != "ada@example.com" {
		t.Errorf("user = %+v", user)
	}
}

func TestGetProfileSelf(t *testing.T) {
	ta := newTestApp(t)
	id := registerUser(t, ta, "ada@example.com")

	httpRes, resWrapper := ta.doAs(t, id, []string{domain.RoleUser}, http.MethodGet, "/api/v1/users/"+strconv.FormatInt(id, 10), "")
	if httpRes.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", httpRes.StatusCode)
	}
	if !resWrapper.Success {
		t.Error("success = false, want true")
	}
}

func TestGetOtherUserProfileForbidden(t *testing.T) {
	ta := newTestApp(t)
	registerUser(t, ta, "ada@example.com")
	other := registerUser(t, ta, "bob@example.com")

	httpRes, _ := ta.doAs(t, 1, []string{domain.RoleUser}, http.MethodGet, "/api/v1/users/"+strconv.FormatInt(other, 10), "")
	if httpRes.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", httpRes.StatusCode)
	}
}
