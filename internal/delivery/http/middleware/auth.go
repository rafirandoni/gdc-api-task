package middleware

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"

	"api-task/internal/domain"
)

const accessRightsKey = "auth.access_rights"

func Auth(tm domain.TokenManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		accessRights, err := authenticate(c, tm)
		if err != nil {
			return err
		}
		c.Locals(accessRightsKey, accessRights)
		return c.Next()
	}
}

func authenticate(c fiber.Ctx, tm domain.TokenManager) (domain.AccessRights, error) {
	rawToken, found := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
	if !found || strings.TrimSpace(rawToken) == "" {
		return domain.AccessRights{}, fmt.Errorf("missing bearer token: %w", domain.ErrUnauthorized)
	}

	accessRights, err := tm.Verify(c, strings.TrimSpace(rawToken))
	if err != nil {
		return domain.AccessRights{}, fmt.Errorf("invalid token: %w", domain.ErrUnauthorized)
	}

	return accessRights, nil
}

func AccessRights(c fiber.Ctx) (domain.AccessRights, bool) {
	accessRights, ok := c.Locals(accessRightsKey).(domain.AccessRights)
	return accessRights, ok
}

func RequireSelfOrAdmin(c fiber.Ctx, id int64) error {
	accessRights, ok := AccessRights(c)
	if !ok {
		return fmt.Errorf("authentication required: %w", domain.ErrUnauthorized)
	}
	if accessRights.UserID == id || accessRights.HasRole(domain.RoleAdmin) {
		return nil
	}
	return fmt.Errorf("cannot access another user's resource: %w", domain.ErrForbidden)
}

func RequireAdmin(c fiber.Ctx) error {
	accessRights, ok := AccessRights(c)
	if !ok {
		return fmt.Errorf("authentication required: %w", domain.ErrUnauthorized)
	}
	if !accessRights.HasRole(domain.RoleAdmin) {
		return fmt.Errorf("admin role required: %w", domain.ErrForbidden)
	}
	return nil
}
