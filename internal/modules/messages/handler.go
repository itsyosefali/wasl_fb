package messages

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/pop/erp_meta/internal/auth"
	"github.com/pop/erp_meta/internal/crypto"
	"github.com/pop/erp_meta/internal/db"
	"github.com/pop/erp_meta/internal/events"
	"github.com/pop/erp_meta/internal/models"
	"github.com/pop/erp_meta/internal/providers"
)

type Handler struct {
	messageRepo *db.MessageRepo
	pageRepo    *db.PageRepo
	contactRepo *db.ContactRepo
	store       *events.Store
	registry    *providers.Registry
	encryptor   *crypto.Encryptor
}

func NewHandler(
	messageRepo *db.MessageRepo,
	pageRepo *db.PageRepo,
	contactRepo *db.ContactRepo,
	store *events.Store,
	registry *providers.Registry,
	encryptor *crypto.Encryptor,
) *Handler {
	return &Handler{
		messageRepo: messageRepo,
		pageRepo:    pageRepo,
		contactRepo: contactRepo,
		store:       store,
		registry:    registry,
		encryptor:   encryptor,
	}
}

type sendRequest struct {
	Channel     string `json:"channel"`
	PageID      string `json:"page_id"`
	RecipientID string `json:"recipient_id"`
	Text        string `json:"text"`
}

func (h *Handler) Register(r fiber.Router) {
	r.Post("/send", h.Send)
	r.Get("/", h.List)
	r.Get("/:id", h.Get)
}

func (h *Handler) Send(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	var req sendRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.PageID == "" || req.RecipientID == "" || req.Text == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "page_id, recipient_id, and text are required"})
	}
	channel := providers.ResolveChannel(req.Channel)

	page, token, err := h.resolvePage(c, tenant.ID, req.PageID)
	if err != nil {
		if te, ok := err.(pageError); ok {
			return c.Status(te.status).JSON(fiber.Map{"error": te.msg})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	provider, err := h.registry.Get(channel)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	externalID, err := provider.SendMessage(c.Context(), providers.SendMessageRequest{
		PageID: page.MetaPageID, RecipientID: req.RecipientID, Text: req.Text, AccessToken: token,
	})
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}

	contact, err := h.contactRepo.Upsert(c.Context(), models.Contact{
		TenantID: tenant.ID, Platform: string(channel), ExternalID: req.RecipientID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	ev, err := h.store.Append(c.Context(), events.AppendInput{
		TenantID:      tenant.ID,
		EventType:     events.MessageSent,
		Channel:       channel,
		AggregateType: events.AggregateMessage,
		AggregateID:   externalID,
		Payload: map[string]any{
			"page_id":      page.ID.String(),
			"contact_id":   contact.ID.String(),
			"external_id":  externalID,
			"recipient_id": req.RecipientID,
			"text":         req.Text,
			"direction":    models.DirectionOut,
			"message":      map[string]string{"text": req.Text, "mid": externalID},
		},
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	message, err := h.messageRepo.GetByID(c.Context(), tenant.ID, ev.ID)
	if err != nil {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"event_id": ev.ID, "external_id": externalID})
	}
	return c.Status(fiber.StatusCreated).JSON(message)
}

func (h *Handler) List(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	messages, err := h.messageRepo.ListByTenant(c.Context(), tenant.ID, 50)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": messages})
}

func (h *Handler) Get(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid message id"})
	}
	message, err := h.messageRepo.GetByID(c.Context(), tenant.ID, id)
	if err != nil {
		if err == db.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "message not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(message)
}

func (h *Handler) resolvePage(c *fiber.Ctx, tenantID uuid.UUID, pageID string) (models.Page, string, error) {
	pageUUID, err := uuid.Parse(pageID)
	if err != nil {
		page, perr := h.pageRepo.GetByMetaPageID(c.Context(), pageID)
		if perr != nil {
			return models.Page{}, "", pageError{status: fiber.StatusBadRequest, msg: "invalid page_id"}
		}
		if page.TenantID != tenantID {
			return models.Page{}, "", pageError{status: fiber.StatusForbidden, msg: "page not found"}
		}
		pageUUID = page.ID
	}
	page, err := h.pageRepo.GetByID(c.Context(), tenantID, pageUUID)
	if err != nil {
		return models.Page{}, "", pageError{status: fiber.StatusNotFound, msg: "page not found"}
	}
	token, err := h.encryptor.Decrypt(page.AccessToken)
	if err != nil {
		return models.Page{}, "", pageError{status: fiber.StatusInternalServerError, msg: "failed to decrypt token"}
	}
	return page, token, nil
}

type pageError struct {
	status int
	msg    string
}

func (e pageError) Error() string { return e.msg }
