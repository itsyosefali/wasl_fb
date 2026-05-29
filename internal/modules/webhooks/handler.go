package webhooks

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/pop/erp_meta/internal/auth"
	"github.com/pop/erp_meta/internal/db"
	"github.com/pop/erp_meta/internal/models"
)

type Handler struct {
	webhookRepo *db.WebhookRepo
}

func NewHandler(webhookRepo *db.WebhookRepo) *Handler {
	return &Handler{webhookRepo: webhookRepo}
}

type createRequest struct {
	URL string `json:"url"`
}

func (h *Handler) Register(r fiber.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Delete("/:id", h.Delete)
}

func (h *Handler) List(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	webhooks, err := h.webhookRepo.ListByTenant(c.Context(), tenant.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	for i := range webhooks {
		webhooks[i].Secret = ""
	}
	return c.JSON(fiber.Map{"data": webhooks})
}

func (h *Handler) Create(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	var req createRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "url is required"})
	}

	secret, err := generateSecret()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate secret"})
	}

	webhook, err := h.webhookRepo.Create(c.Context(), models.Webhook{
		TenantID: tenant.ID,
		URL:      req.URL,
		Secret:   secret,
		Enabled:  true,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(webhook)
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid webhook id"})
	}
	if err := h.webhookRepo.Delete(c.Context(), tenant.ID, id); err != nil {
		if err == db.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "webhook not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func generateSecret() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
