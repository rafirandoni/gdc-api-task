package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"api-task/internal/domain"
)

type loginPayload struct {
	Token     string `json:"token"`
	TokenType string `json:"token_type"`
	ExpiresIn int64  `json:"expires_in"`
	User      struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

func registerAndPassword(t *testing.T, ta *testApp, email string) (int64, string) {
	t.Helper()
	const password = "password"
	id := registerUser(t, ta, email)
	return id, password
}

func TestLoginSuccess(t *testing.T) {
	ta := newTestApp(t)
	email := "ada@example.com"
	id, password := registerAndPassword(t, ta, email)

	body := `{"email":"ada@example.com","password":"password"}`
	httpRes, resWrapper := ta.doOpen(t, http.MethodPost, "/api/v1/auth/login", body)
	if httpRes.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200 (body: %s)", httpRes.StatusCode, resWrapper.Data)
	}
	var payload loginPayload
	if err := json.Unmarshal(resWrapper.Data, &payload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	if payload.Token == "" || payload.TokenType != "Bearer" || payload.ExpiresIn <= 0 {
		t.Errorf("login payload = %+v", payload)
	}

	if payload.User.ID != id || payload.User.Email != email {
		t.Errorf("user = %+v", payload.User)
	}

	if password == "" {
		t.Fatal("unreachable")
	}

	accessRights, err := ta.Token.Verify(context.Background(), payload.Token)
	if err != nil {
		t.Fatalf("verify login token: %v", err)
	}

	if accessRights.UserID != id {
		t.Errorf("accessRights user id = %d, want %d", accessRights.UserID, id)
	}

	if !accessRights.HasRole(domain.RoleUser) {
		t.Errorf("access rights roles = %v, want implicit USER", accessRights.Roles)
	}

	meResp, _ := ta.doToken(t, payload.Token, http.MethodGet, "/api/v1/users/me", "")
	if meResp.StatusCode != http.StatusOK {
		t.Errorf("GET /users/me with login token status = %d, want 200", meResp.StatusCode)
	}
}

func TestLoginBadCredentials(t *testing.T) {
	ta := newTestApp(t)
	registerAndPassword(t, ta, "rafi@example.com")

	for _, body := range []string{
		`{"email":"rafi@example.com","password":"wrong password"}`,
		`{"email":"nobody@example.com","password":"correct"}`,
	} {
		httpRes, resWrapper := ta.doOpen(t, http.MethodPost, "/api/v1/auth/login", body)
		if httpRes.StatusCode != http.StatusUnauthorized {
			t.Errorf("body %q: status = %d, want 401", body, httpRes.StatusCode)
		}
		if resWrapper.Success {
			t.Errorf("body %q: success = true", body)
		}
	}
}

func TestLoginValidation(t *testing.T) {
	ta := newTestApp(t)

	httpRes, _ := ta.doOpen(t, http.MethodPost, "/api/v1/auth/login", `{"email":"not-an-email","password":"123123"}`)
	if httpRes.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid email status = %d, want 400", httpRes.StatusCode)
	}
	httpRes, _ = ta.doOpen(t, http.MethodPost, "/api/v1/auth/login", `{"email":`)
	if httpRes.StatusCode != http.StatusBadRequest {
		t.Errorf("malformed body status = %d, want 400", httpRes.StatusCode)
	}
}
