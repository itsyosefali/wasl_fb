package events

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/pop/erp_meta/internal/db"
	"github.com/pop/erp_meta/internal/models"
)

// Projector materializes read models from events.
type Projector struct {
	contactRepo *db.ContactRepo
	messageRepo *db.MessageRepo
	commentRepo *db.CommentRepo
}

func NewProjector(contactRepo *db.ContactRepo, messageRepo *db.MessageRepo, commentRepo *db.CommentRepo) *Projector {
	return &Projector{
		contactRepo: contactRepo,
		messageRepo: messageRepo,
		commentRepo: commentRepo,
	}
}

func (p *Projector) Project(ctx context.Context, ev models.Event) error {
	var payload map[string]any
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		return err
	}

	switch ev.EventType {
	case MessageReceived, MessageSent, InstagramMessageRecv:
		return p.projectMessage(ctx, ev, payload)
	case CommentCreated, CommentUpdated, CommentDeleted, CommentReplied, InstagramCommentCreated:
		return p.projectComment(ctx, ev, payload)
	case CommentHidden:
		return p.projectCommentHidden(ctx, ev, payload)
	}
	return nil
}

func (p *Projector) projectMessage(ctx context.Context, ev models.Event, payload map[string]any) error {
	pageID, err := parseUUID(payload["page_id"])
	if err != nil {
		return nil
	}
	contactID, err := parseUUID(payload["contact_id"])
	if err != nil {
		return nil
	}

	direction := models.DirectionIn
	if ev.EventType == MessageSent {
		direction = models.DirectionOut
	}

	text, _ := payload["text"].(string)
	if text == "" {
		if msg, ok := payload["message"].(map[string]any); ok {
			text, _ = msg["text"].(string)
		}
	}
	externalID, _ := payload["external_id"].(string)
	if externalID == "" {
		if msg, ok := payload["message"].(map[string]any); ok {
			externalID, _ = msg["mid"].(string)
		}
	}

	_, err = p.messageRepo.CreateWithEvent(ctx, models.Message{
		ID:         ev.ID,
		TenantID:   ev.TenantID,
		ExternalID: externalID,
		PageID:     pageID,
		ContactID:  contactID,
		Direction:  direction,
		Message:    text,
	}, ev.ID)
	return err
}

func (p *Projector) projectComment(ctx context.Context, ev models.Event, payload map[string]any) error {
	pageID, err := parseUUID(payload["page_id"])
	if err != nil {
		return nil
	}
	contactID, err := parseUUID(payload["contact_id"])
	if err != nil {
		return nil
	}

	externalID, _ := payload["external_id"].(string)
	if externalID == "" {
		externalID, _ = payload["comment_id"].(string)
	}
	text, _ := payload["text"].(string)
	if text == "" {
		text, _ = payload["message"].(string)
	}

	status := models.CommentStatusVisible
	if ev.EventType == CommentDeleted {
		status = "deleted"
	}

	_, err = p.commentRepo.CreateWithEvent(ctx, models.Comment{
		ID:         ev.ID,
		TenantID:   ev.TenantID,
		ExternalID: externalID,
		PageID:     pageID,
		ContactID:  contactID,
		Message:    text,
		Status:     status,
	}, ev.ID)
	return err
}

func (p *Projector) projectCommentHidden(ctx context.Context, ev models.Event, payload map[string]any) error {
	externalID, _ := payload["comment_id"].(string)
	if externalID == "" {
		externalID, _ = payload["external_id"].(string)
	}
	if externalID == "" {
		return nil
	}
	return p.commentRepo.UpdateStatus(ctx, ev.TenantID, externalID, models.CommentStatusHidden)
}

func parseUUID(v any) (uuid.UUID, error) {
	switch t := v.(type) {
	case string:
		return uuid.Parse(t)
	case uuid.UUID:
		return t, nil
	default:
		return uuid.Nil, errors.New("invalid uuid")
	}
}
