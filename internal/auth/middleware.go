package auth

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/pop/erp_meta/internal/db"
	"github.com/pop/erp_meta/internal/models"
)

type tenantKey struct{}

func TenantFromContext(ctx context.Context) (models.Tenant, bool) {
	t, ok := ctx.Value(tenantKey{}).(models.Tenant)
	return t, ok
}

func Middleware(tenantRepo *db.TenantRepo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing X-API-Key header",
			})
		}

		tenant, err := tenantRepo.GetByAPIKey(c.Context(), apiKey)
		if err != nil {
			if err == db.ErrNotFound {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "invalid API key",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "authentication failed",
			})
		}

		c.SetUserContext(context.WithValue(c.UserContext(), tenantKey{}, tenant))
		return c.Next()
	}
}

func GetTenant(c *fiber.Ctx) (models.Tenant, error) {
	tenant, ok := TenantFromContext(c.UserContext())
	if !ok {
		return models.Tenant{}, fiber.NewError(fiber.StatusUnauthorized, "tenant not found in context")
	}
	return tenant, nil
}
