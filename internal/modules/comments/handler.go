package comments

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
	commentRepo *db.CommentRepo
	pageRepo    *db.PageRepo
	store       *events.Store
	registry    *providers.Registry
	encryptor   *crypto.Encryptor
}

func NewHandler(
	commentRepo *db.CommentRepo,
	pageRepo *db.PageRepo,
	store *events.Store,
	registry *providers.Registry,
	encryptor *crypto.Encryptor,
) *Handler {
	return &Handler{
		commentRepo: commentRepo,
		pageRepo:    pageRepo,
		store:       store,
		registry:    registry,
		encryptor:   encryptor,
	}
}

type replyRequest struct {
	Channel   string `json:"channel"`
	CommentID string `json:"comment_id"`
	PageID    string `json:"page_id"`
	Text      string `json:"text"`
}

type hideRequest struct {
	Channel   string `json:"channel"`
	CommentID string `json:"comment_id"`
	PageID    string `json:"page_id"`
}

func (h *Handler) Register(r fiber.Router) {
	r.Post("/reply", h.Reply)
	r.Post("/private-reply", h.PrivateReply)
	r.Post("/hide", h.Hide)
	r.Get("/", h.List)
}

func (h *Handler) List(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	comments, err := h.commentRepo.ListByTenant(c.Context(), tenant.ID, 50)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": comments})
}

func (h *Handler) Reply(c *fiber.Ctx) error {
	return h.doReply(c, false)
}

func (h *Handler) PrivateReply(c *fiber.Ctx) error {
	return h.doReply(c, true)
}

func (h *Handler) doReply(c *fiber.Ctx, private bool) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	var req replyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.CommentID == "" || req.Text == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "comment_id and text are required"})
	}
	channel := providers.ResolveChannel(req.Channel)

	page, token, err := h.resolveToken(c, tenant.ID, req.PageID, req.CommentID)
	if err != nil {
		if te, ok := err.(tokenError); ok {
			return c.Status(te.status).JSON(fiber.Map{"error": te.msg})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	provider, err := h.registry.Get(channel)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	replyReq := providers.ReplyCommentRequest{CommentID: req.CommentID, Text: req.Text, AccessToken: token}
	var externalID string
	if private {
		externalID, err = provider.PrivateReplyComment(c.Context(), replyReq)
	} else {
		externalID, err = provider.ReplyComment(c.Context(), replyReq)
	}
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}

	_, _ = h.store.Append(c.Context(), events.AppendInput{
		TenantID:      tenant.ID,
		EventType:     events.CommentReplied,
		Channel:       channel,
		AggregateType: events.AggregateComment,
		AggregateID:   externalID,
		Payload: map[string]any{
			"page_id":    page.ID.String(),
			"comment_id": req.CommentID,
			"external_id": externalID,
			"text":       req.Text,
			"private":    private,
		},
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"comment_id": externalID,
		"text":       req.Text,
		"private":    private,
	})
}

func (h *Handler) Hide(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	var req hideRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.CommentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "comment_id is required"})
	}
	channel := providers.ResolveChannel(req.Channel)

	_, token, err := h.resolveToken(c, tenant.ID, req.PageID, req.CommentID)
	if err != nil {
		if te, ok := err.(tokenError); ok {
			return c.Status(te.status).JSON(fiber.Map{"error": te.msg})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	provider, err := h.registry.Get(channel)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := provider.HideComment(c.Context(), req.CommentID, token); err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}

	_, _ = h.store.Append(c.Context(), events.AppendInput{
		TenantID:      tenant.ID,
		EventType:     events.CommentHidden,
		Channel:       channel,
		AggregateType: events.AggregateComment,
		AggregateID:   req.CommentID,
		Payload: map[string]any{
			"comment_id": req.CommentID,
			"status":     models.CommentStatusHidden,
		},
	})

	return c.JSON(fiber.Map{"comment_id": req.CommentID, "hidden": true})
}

type tokenError struct {
	status int
	msg    string
}

func (e tokenError) Error() string { return e.msg }

func (h *Handler) resolveToken(c *fiber.Ctx, tenantID uuid.UUID, pageID, commentID string) (models.Page, string, error) {
	if pageID != "" {
		pageUUID, err := uuid.Parse(pageID)
		if err != nil {
			page, perr := h.pageRepo.GetByMetaPageID(c.Context(), pageID)
			if perr != nil {
				return models.Page{}, "", tokenError{status: fiber.StatusBadRequest, msg: "invalid page_id"}
			}
			if page.TenantID != tenantID {
				return models.Page{}, "", tokenError{status: fiber.StatusForbidden, msg: "page not found"}
			}
			pageUUID = page.ID
		}
		page, err := h.pageRepo.GetByID(c.Context(), tenantID, pageUUID)
		if err != nil {
			return models.Page{}, "", tokenError{status: fiber.StatusNotFound, msg: "page not found"}
		}
		token, err := h.encryptor.Decrypt(page.AccessToken)
		return page, token, err
	}

	comment, err := h.commentRepo.GetByExternalID(c.Context(), tenantID, commentID)
	if err != nil {
		return models.Page{}, "", tokenError{status: fiber.StatusBadRequest, msg: "page_id required when comment is unknown"}
	}
	page, err := h.pageRepo.GetByID(c.Context(), tenantID, comment.PageID)
	if err != nil {
		return models.Page{}, "", tokenError{status: fiber.StatusNotFound, msg: "page not found"}
	}
	token, err := h.encryptor.Decrypt(page.AccessToken)
	return page, token, err
}
