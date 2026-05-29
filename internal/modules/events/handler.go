package eventmod

import (
	"github.com/gofiber/fiber/v2"
	"github.com/pop/erp_meta/internal/auth"
	"github.com/pop/erp_meta/internal/db"
)

type Handler struct {
	eventRepo *db.EventRepo
}

func NewHandler(eventRepo *db.EventRepo) *Handler {
	return &Handler{eventRepo: eventRepo}
}

func (h *Handler) Register(r fiber.Router) {
	r.Get("/", h.List)
}

func (h *Handler) List(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	events, err := h.eventRepo.ListByTenant(c.Context(), tenant.ID, 50)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": events})
}
