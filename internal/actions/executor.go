package actions

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pop/erp_meta/internal/crypto"
	"github.com/pop/erp_meta/internal/db"
	"github.com/pop/erp_meta/internal/events"
	"github.com/pop/erp_meta/internal/models"
	"github.com/pop/erp_meta/internal/providers"
)

const (
	ActionSendMessage  = "send_message"
	ActionSendImage    = "send_image"
	ActionSendCarousel = "send_carousel"
	ActionSendTemplate = "send_template"
	ActionSendProduct  = "send_product"
	ActionReplyComment = "reply_comment"
	ActionPrivateReply = "private_reply"
	ActionHideComment  = "hide_comment"
)

type Request struct {
	Action      string         `json:"action"`
	Channel     string         `json:"channel"`
	PageID      string         `json:"page_id"`
	RecipientID string         `json:"recipient_id,omitempty"`
	CommentID   string         `json:"comment_id,omitempty"`
	Text        string         `json:"text,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
}

type Result struct {
	Action     string `json:"action"`
	ExternalID string `json:"external_id,omitempty"`
	Success    bool   `json:"success"`
}

type Executor struct {
	registry  *providers.Registry
	pageRepo  *db.PageRepo
	store     *events.Store
	encryptor *crypto.Encryptor
}

func NewExecutor(registry *providers.Registry, pageRepo *db.PageRepo, store *events.Store, encryptor *crypto.Encryptor) *Executor {
	return &Executor{
		registry:  registry,
		pageRepo:  pageRepo,
		store:     store,
		encryptor: encryptor,
	}
}

func (e *Executor) Execute(ctx context.Context, tenantID uuid.UUID, req Request) (Result, error) {
	channel := providers.ResolveChannel(req.Channel)
	provider, err := e.registry.Get(channel)
	if err != nil {
		return Result{}, err
	}

	page, token, err := e.resolvePage(ctx, tenantID, req.PageID)
	if err != nil {
		return Result{}, err
	}

	var externalID string
	switch req.Action {
	case ActionSendMessage:
		externalID, err = provider.SendMessage(ctx, providers.SendMessageRequest{
			PageID: page.MetaPageID, RecipientID: req.RecipientID, Text: req.Text, AccessToken: token,
		})
	case ActionSendImage:
		externalID, err = provider.SendImage(ctx, providers.SendMediaRequest{
			PageID: page.MetaPageID, RecipientID: req.RecipientID, URL: stringVal(req.Data, "url"), AccessToken: token,
		})
	case ActionSendCarousel:
		externalID, err = provider.SendCarousel(ctx, providers.SendCarouselRequest{
			PageID: page.MetaPageID, RecipientID: req.RecipientID, Elements: parseCarouselElements(req.Data), AccessToken: token,
		})
	case ActionSendTemplate:
		externalID, err = provider.SendTemplate(ctx, providers.SendTemplateRequest{
			PageID: page.MetaPageID, RecipientID: req.RecipientID,
			TemplateName: stringVal(req.Data, "template_name"),
			Language:     stringVal(req.Data, "language"),
			Components:   componentsVal(req.Data),
			AccessToken:  token,
		})
	case ActionSendProduct:
		externalID, err = provider.SendProduct(ctx, providers.SendProductRequest{
			PageID: page.MetaPageID, RecipientID: req.RecipientID,
			ProductID: stringVal(req.Data, "product_id"),
			CatalogID: stringVal(req.Data, "catalog_id"),
			AccessToken: token,
		})
	case ActionReplyComment:
		externalID, err = provider.ReplyComment(ctx, providers.ReplyCommentRequest{
			CommentID: req.CommentID, Text: req.Text, AccessToken: token,
		})
	case ActionPrivateReply:
		externalID, err = provider.PrivateReplyComment(ctx, providers.ReplyCommentRequest{
			CommentID: req.CommentID, Text: req.Text, AccessToken: token,
		})
	case ActionHideComment:
		err = provider.HideComment(ctx, req.CommentID, token)
	default:
		return Result{}, fmt.Errorf("unknown action: %s", req.Action)
	}
	if err != nil {
		return Result{}, err
	}

	eventType := events.MessageSent
	aggregateType := events.AggregateMessage
	payload := map[string]any{
		"page_id":      page.ID.String(),
		"channel":      string(channel),
		"action":       req.Action,
		"external_id":  externalID,
		"recipient_id": req.RecipientID,
		"text":         req.Text,
	}
	if req.Action == ActionReplyComment || req.Action == ActionPrivateReply {
		eventType = events.CommentReplied
		aggregateType = events.AggregateComment
		payload["comment_id"] = req.CommentID
		payload["external_id"] = externalID
	}
	if req.Action == ActionHideComment {
		eventType = events.CommentHidden
		aggregateType = events.AggregateComment
		payload["comment_id"] = req.CommentID
	}

	_, _ = e.store.Append(ctx, events.AppendInput{
		TenantID:      tenantID,
		EventType:     eventType,
		Channel:       channel,
		AggregateType: aggregateType,
		AggregateID:   externalID,
		Payload:       payload,
	})

	return Result{Action: req.Action, ExternalID: externalID, Success: true}, nil
}

func (e *Executor) resolvePage(ctx context.Context, tenantID uuid.UUID, pageID string) (models.Page, string, error) {
	page, err := e.pageRepo.GetByMetaPageID(ctx, pageID)
	if err != nil {
		parsed, perr := uuid.Parse(pageID)
		if perr != nil {
			return models.Page{}, "", fmt.Errorf("page not found")
		}
		page, err = e.pageRepo.GetByID(ctx, tenantID, parsed)
		if err != nil {
			return models.Page{}, "", fmt.Errorf("page not found")
		}
	}
	if page.TenantID != tenantID {
		return models.Page{}, "", fmt.Errorf("page not found")
	}
	token, err := e.encryptor.Decrypt(page.AccessToken)
	if err != nil {
		return models.Page{}, "", err
	}
	return page, token, nil
}

func stringVal(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	v, _ := data[key].(string)
	return v
}

func componentsVal(data map[string]any) []map[string]any {
	if data == nil {
		return nil
	}
	raw, ok := data["components"].([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func parseCarouselElements(data map[string]any) []providers.CarouselElement {
	if data == nil {
		return nil
	}
	raw, ok := data["elements"].([]any)
	if !ok {
		return nil
	}
	out := make([]providers.CarouselElement, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, providers.CarouselElement{
			Title:    stringVal(m, "title"),
			Subtitle: stringVal(m, "subtitle"),
			ImageURL: stringVal(m, "image_url"),
			URL:      stringVal(m, "url"),
		})
	}
	return out
}
