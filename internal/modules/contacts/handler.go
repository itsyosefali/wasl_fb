package contacts

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/pop/erp_meta/internal/auth"
	"github.com/pop/erp_meta/internal/db"
)

type Handler struct {
	contactRepo *db.ContactRepo
}

func NewHandler(contactRepo *db.ContactRepo) *Handler {
	return &Handler{contactRepo: contactRepo}
}

func (h *Handler) Register(r fiber.Router) {
	r.Get("/", h.List)
	r.Get("/:id", h.Get)
}

func (h *Handler) List(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	contacts, err := h.contactRepo.ListByTenant(c.Context(), tenant.ID, 50)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": contacts})
}

func (h *Handler) Get(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid contact id"})
	}
	contact, err := h.contactRepo.GetByID(c.Context(), tenant.ID, id)
	if err != nil {
		if err == db.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "contact not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(contact)
}
