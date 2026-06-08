package conversations

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/pop/erp_meta/internal/auth"
	"github.com/pop/erp_meta/internal/db"
)

type Handler struct {
	conversationRepo *db.ConversationRepo
}

func NewHandler(conversationRepo *db.ConversationRepo) *Handler {
	return &Handler{conversationRepo: conversationRepo}
}

func (h *Handler) Register(r fiber.Router) {
	r.Get("/", h.List)
	r.Get("/:contact_id/messages", h.Messages)
}

func (h *Handler) List(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	conversations, err := h.conversationRepo.ListByTenant(c.Context(), tenant.ID, pageFilter(c), 50)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if conversations == nil {
		conversations = []db.Conversation{}
	}
	return c.JSON(fiber.Map{"data": conversations})
}

func (h *Handler) Messages(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	contactID, err := uuid.Parse(c.Params("contact_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid contact_id"})
	}
	pageIDStr := c.Query("page_id")
	if pageIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "page_id query parameter required"})
	}
	pageID, err := uuid.Parse(pageIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid page_id"})
	}

	rows, err := h.conversationRepo.ListMessages(c.Context(), tenant.ID, contactID, pageID, 100)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	type msg struct {
		ID         uuid.UUID `json:"id"`
		ExternalID string    `json:"external_id,omitempty"`
		Direction  string    `json:"direction"`
		Message    string    `json:"message"`
		CreatedAt  string    `json:"created_at"`
	}
	out := make([]msg, 0, len(rows))
	for _, m := range rows {
		out = append(out, msg{
			ID: m.ID, ExternalID: m.ExternalID, Direction: m.Direction,
			Message: m.Message, CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return c.JSON(fiber.Map{"data": out})
}

func pageFilter(c *fiber.Ctx) *uuid.UUID {
	pageIDStr := c.Query("page_id")
	if pageIDStr == "" {
		return nil
	}
	pageID, err := uuid.Parse(pageIDStr)
	if err != nil {
		return nil
	}
	return &pageID
}
