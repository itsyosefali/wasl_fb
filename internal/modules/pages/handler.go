package pages

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
	pageRepo  *db.PageRepo
	store     *events.Store
	encryptor *crypto.Encryptor
}

func NewHandler(pageRepo *db.PageRepo, store *events.Store, encryptor *crypto.Encryptor) *Handler {
	return &Handler{
		pageRepo:  pageRepo,
		store:     store,
		encryptor: encryptor,
	}
}

type connectRequest struct {
	MetaPageID  string `json:"meta_page_id"`
	Name        string `json:"name"`
	AccessToken string `json:"access_token"`
}

func (h *Handler) Register(r fiber.Router) {
	r.Get("/", h.List)
	r.Get("/:id", h.Get)
	r.Post("/connect", h.Connect)
	r.Delete("/:id", h.Delete)
}

func (h *Handler) List(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	pages, err := h.pageRepo.ListByTenant(c.Context(), tenant.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if pages == nil {
		pages = []models.Page{}
	}
	for i := range pages {
		pages[i].AccessToken = ""
	}
	return c.JSON(fiber.Map{"data": pages})
}

func (h *Handler) Get(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid page id"})
	}
	page, err := h.pageRepo.GetByID(c.Context(), tenant.ID, id)
	if err != nil {
		if err == db.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "page not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	page.AccessToken = ""
	return c.JSON(page)
}

func (h *Handler) Connect(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	var req connectRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.MetaPageID == "" || req.AccessToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "meta_page_id and access_token are required"})
	}

	encrypted, err := h.encryptor.Encrypt(req.AccessToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to encrypt token"})
	}

	page, err := h.pageRepo.Create(c.Context(), models.Page{
		TenantID:    tenant.ID,
		MetaPageID:  req.MetaPageID,
		Name:        req.Name,
		AccessToken: encrypted,
		Status:      models.PageStatusActive,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	_, err = h.store.Append(c.Context(), events.AppendInput{
		TenantID:      tenant.ID,
		EventType:     events.PageConnected,
		Channel:       providers.ChannelFacebook,
		AggregateType: events.AggregatePage,
		AggregateID:   page.MetaPageID,
		Payload: map[string]any{
			"page_id":     page.MetaPageID,
			"name":        page.Name,
			"internal_id": page.ID.String(),
		},
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	page.AccessToken = ""
	return c.Status(fiber.StatusCreated).JSON(page)
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid page id"})
	}
	if err := h.pageRepo.Delete(c.Context(), tenant.ID, id); err != nil {
		if err == db.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "page not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
