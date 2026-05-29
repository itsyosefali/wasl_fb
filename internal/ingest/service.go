package ingest

import (
	"context"

	"github.com/google/uuid"
	"github.com/pop/erp_meta/internal/db"
	"github.com/pop/erp_meta/internal/events"
	"github.com/pop/erp_meta/internal/models"
	"github.com/pop/erp_meta/internal/providers"
)

type Service struct {
	pageRepo    *db.PageRepo
	contactRepo *db.ContactRepo
	store       *events.Store
	registry    *providers.Registry
}

func NewService(
	pageRepo *db.PageRepo,
	contactRepo *db.ContactRepo,
	store *events.Store,
	registry *providers.Registry,
) *Service {
	return &Service{
		pageRepo:    pageRepo,
		contactRepo: contactRepo,
		store:       store,
		registry:    registry,
	}
}

func (s *Service) ProcessPayload(ctx context.Context, body []byte, channel providers.Channel) (int, error) {
	provider, err := s.registry.Get(channel)
	if err != nil {
		// fallback to facebook/meta parser for legacy webhook endpoint
		provider, err = s.registry.Get(providers.ChannelFacebook)
		if err != nil {
			return 0, err
		}
	}

	normalized, err := provider.ParseWebhook(body)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, ne := range normalized {
		page, err := s.pageRepo.GetByMetaPageID(ctx, ne.PageID)
		if err != nil {
			continue
		}

		senderID, _ := ne.Payload["user_id"].(string)
		if senderID == "" {
			if sender, ok := ne.Payload["sender"].(map[string]string); ok {
				senderID = sender["id"]
			}
		}
		senderName, _ := ne.Payload["user_name"].(string)

		contact, err := s.contactRepo.Upsert(ctx, models.Contact{
			TenantID:   page.TenantID,
			Platform:   string(ne.Channel),
			ExternalID: senderID,
			Name:       senderName,
		})
		if err != nil {
			continue
		}

		payload := make(map[string]any, len(ne.Payload)+4)
		for k, v := range ne.Payload {
			payload[k] = v
		}
		payload["page_id"] = page.ID.String()
		payload["contact_id"] = contact.ID.String()
		payload["channel"] = string(ne.Channel)

		aggregateType := events.AggregateMessage
		aggregateID := ""
		if isCommentEvent(ne.EventType) {
			aggregateType = events.AggregateComment
			aggregateID, _ = payload["comment_id"].(string)
		} else if mid, ok := payload["message"].(map[string]string); ok {
			aggregateID = mid["mid"]
		}

		_, err = s.store.Append(ctx, events.AppendInput{
			TenantID:      page.TenantID,
			EventType:     ne.EventType,
			Channel:       ne.Channel,
			AggregateType: aggregateType,
			AggregateID:   aggregateID,
			Payload:       payload,
		})
		if err != nil {
			continue
		}
		processed++
		_ = uuid.Nil
	}
	return processed, nil
}

func isCommentEvent(eventType string) bool {
	switch eventType {
	case events.CommentCreated, events.CommentUpdated, events.CommentDeleted, events.CommentReplied, events.InstagramCommentCreated:
		return true
	default:
		return false
	}
}
