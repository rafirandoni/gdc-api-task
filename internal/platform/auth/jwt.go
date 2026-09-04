package auth

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"api-task/internal/domain"
)

const (
	ISSUER           = "gdcpay-task-api"
	MIN_SECRET_BYTES = 32
)

type Manager struct {
	secret []byte
	ttl    time.Duration
}

type claims struct {
	UserID int64    `json:"uid"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

func NewManager(secret string, ttl time.Duration) (*Manager, error) {
	if len(secret) < MIN_SECRET_BYTES {
		return nil, fmt.Errorf("JWT secret must be at least %d bytes", MIN_SECRET_BYTES)
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("JWT TTL must be positive, got %s", ttl)
	}
	return &Manager{secret: []byte(secret), ttl: ttl}, nil
}

var _ domain.TokenManager = (*Manager)(nil)

func (m *Manager) Issue(ctx context.Context, userID int64, roles []string) (domain.Token, error) {
	now := time.Now()
	expiresAt := now.Add(m.ttl)

	registered := jwt.RegisteredClaims{
		Issuer:    ISSUER,
		Subject:   strconv.FormatInt(userID, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UserID:           userID,
		Roles:            roles,
		RegisteredClaims: registered,
	})

	value, err := token.SignedString(m.secret)
	if err != nil {
		return domain.Token{}, fmt.Errorf("issue token: %w", err)
	}
	return domain.Token{Value: value, ExpiresAt: expiresAt}, nil
}

func (m *Manager) Verify(ctx context.Context, raw string) (domain.AccessRights, error) {
	token, err := jwt.ParseWithClaims(raw, &claims{}, func(t *jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(ISSUER), jwt.WithExpirationRequired())
	if err != nil {
		return domain.AccessRights{}, fmt.Errorf("verify token: %w", domain.ErrUnauthorized)
	}

	parsed, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return domain.AccessRights{}, fmt.Errorf("verify token: %w", domain.ErrUnauthorized)
	}
	return domain.AccessRights{UserID: parsed.UserID, Roles: parsed.Roles}, nil
}
