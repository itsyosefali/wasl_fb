package meta

import (
	"context"
	"encoding/json"

	"github.com/pop/erp_meta/internal/events"
	"github.com/pop/erp_meta/internal/providers"
)

// FacebookProvider implements Provider for Facebook Pages / Messenger.
type FacebookProvider struct {
	graph     *GraphClient
	appSecret string
}

func NewFacebookProvider(graphVersion, appSecret string) *FacebookProvider {
	return &FacebookProvider{
		graph:     NewGraphClient(graphVersion),
		appSecret: appSecret,
	}
}

func (p *FacebookProvider) Channel() providers.Channel {
	return providers.ChannelFacebook
}

func (p *FacebookProvider) SendMessage(ctx context.Context, req providers.SendMessageRequest) (string, error) {
	body := map[string]any{
		"recipient": map[string]string{"id": req.RecipientID},
		"message":   map[string]string{"text": req.Text},
	}
	return p.graph.SendRawMessage(ctx, req.PageID, body, req.AccessToken)
}

func (p *FacebookProvider) SendImage(ctx context.Context, req providers.SendMediaRequest) (string, error) {
	body := map[string]any{
		"recipient": map[string]string{"id": req.RecipientID},
		"message": map[string]any{
			"attachment": map[string]any{
				"type": "image",
				"payload": map[string]string{
					"url":         req.URL,
					"is_reusable": "true",
				},
			},
		},
	}
	return p.graph.SendRawMessage(ctx, req.PageID, body, req.AccessToken)
}

func (p *FacebookProvider) SendCarousel(ctx context.Context, req providers.SendCarouselRequest) (string, error) {
	elements := make([]map[string]any, 0, len(req.Elements))
	for _, el := range req.Elements {
		element := map[string]any{
			"title": el.Title,
		}
		if el.Subtitle != "" {
			element["subtitle"] = el.Subtitle
		}
		if el.ImageURL != "" {
			element["image_url"] = el.ImageURL
		}
		if el.URL != "" {
			element["default_action"] = map[string]any{
				"type": "web_url",
				"url":  el.URL,
			}
			element["buttons"] = []map[string]any{{
				"type":  "web_url",
				"url":   el.URL,
				"title": "View",
			}}
		}
		elements = append(elements, element)
	}
	body := map[string]any{
		"recipient": map[string]string{"id": req.RecipientID},
		"message": map[string]any{
			"attachment": map[string]any{
				"type":    "template",
				"payload": map[string]any{"template_type": "generic", "elements": elements},
			},
		},
	}
	return p.graph.SendRawMessage(ctx, req.PageID, body, req.AccessToken)
}

func (p *FacebookProvider) SendTemplate(ctx context.Context, req providers.SendTemplateRequest) (string, error) {
	lang := req.Language
	if lang == "" {
		lang = "en_US"
	}
	body := map[string]any{
		"recipient": map[string]string{"id": req.RecipientID},
		"message": map[string]any{
			"attachment": map[string]any{
				"type": "template",
				"payload": map[string]any{
					"template_type": "button",
					"text":          req.TemplateName,
					"buttons":       req.Components,
				},
			},
		},
	}
	_ = lang
	return p.graph.SendRawMessage(ctx, req.PageID, body, req.AccessToken)
}

func (p *FacebookProvider) SendProduct(ctx context.Context, req providers.SendProductRequest) (string, error) {
	body := map[string]any{
		"recipient": map[string]string{"id": req.RecipientID},
		"message": map[string]any{
			"attachment": map[string]any{
				"type": "template",
				"payload": map[string]any{
					"template_type": "product",
					"elements": []map[string]any{{
						"id": req.ProductID,
					}},
				},
			},
		},
	}
	if req.CatalogID != "" {
		body["message"].(map[string]any)["attachment"].(map[string]any)["payload"].(map[string]any)["product_retailer_id"] = req.ProductID
	}
	return p.graph.SendRawMessage(ctx, req.PageID, body, req.AccessToken)
}

func (p *FacebookProvider) ReplyComment(ctx context.Context, req providers.ReplyCommentRequest) (string, error) {
	return p.graph.ReplyComment(ctx, req.CommentID, req.Text, req.AccessToken)
}

func (p *FacebookProvider) PrivateReplyComment(ctx context.Context, req providers.ReplyCommentRequest) (string, error) {
	return p.graph.PrivateReplyComment(ctx, req.CommentID, req.Text, req.AccessToken)
}

func (p *FacebookProvider) HideComment(ctx context.Context, commentID, accessToken string) error {
	return p.graph.HideComment(ctx, commentID, accessToken)
}

func (p *FacebookProvider) VerifyWebhook(signatureHeader string, body []byte) bool {
	return VerifyWebhookSignature(p.appSecret, signatureHeader, body)
}

func (p *FacebookProvider) VerifyChallenge(mode, token, challenge, expectedToken string) (string, bool) {
	if mode == "subscribe" && token == expectedToken {
		return challenge, true
	}
	return "", false
}

func (p *FacebookProvider) ParseWebhook(body []byte) ([]providers.NormalizedEvent, error) {
	return parseMetaWebhook(body, providers.ChannelFacebook)
}

func parseMetaWebhook(body []byte, defaultChannel providers.Channel) ([]providers.NormalizedEvent, error) {
	var wp struct {
		Object string `json:"object"`
		Entry  []struct {
			ID        string `json:"id"`
			Messaging []struct {
				Sender    struct{ ID string } `json:"sender"`
				Recipient struct{ ID string } `json:"recipient"`
				Timestamp int64               `json:"timestamp"`
				Message   *struct {
					MID  string `json:"mid"`
					Text string `json:"text"`
				} `json:"message"`
			} `json:"messaging"`
			Changes []struct {
				Field string          `json:"field"`
				Value json.RawMessage `json:"value"`
			} `json:"changes"`
		} `json:"entry"`
	}
	if err := json.Unmarshal(body, &wp); err != nil {
		return nil, err
	}

	channel := defaultChannel
	if wp.Object == "instagram" {
		channel = providers.ChannelInstagram
	}

	var out []providers.NormalizedEvent
	for _, entry := range wp.Entry {
		pageID := entry.ID

		for _, msg := range entry.Messaging {
			if msg.Message == nil {
				continue
			}
			eventType := events.MessageReceived
			if channel == providers.ChannelInstagram {
				eventType = events.InstagramMessageRecv
			}
			out = append(out, providers.NormalizedEvent{
				EventType: eventType,
				Channel:   channel,
				PageID:    pageID,
				Payload: map[string]any{
					"sender":    map[string]string{"id": msg.Sender.ID},
					"message":   map[string]string{"text": msg.Message.Text, "mid": msg.Message.MID},
					"timestamp": msg.Timestamp,
				},
			})
		}

		for _, change := range entry.Changes {
			if change.Field != "feed" && change.Field != "comments" {
				continue
			}
			var feed struct {
				Item      string `json:"item"`
				Verb      string `json:"verb"`
				CommentID string `json:"comment_id"`
				Message   string `json:"message"`
				From      struct {
					ID       string `json:"id"`
					Name     string `json:"name"`
					Username string `json:"username"`
				} `json:"from"`
				ID   string `json:"id"`
				Text string `json:"text"`
			}
			if err := json.Unmarshal(change.Value, &feed); err != nil {
				continue
			}

			commentID := feed.CommentID
			text := feed.Message
			senderID := feed.From.ID
			senderName := feed.From.Name
			if commentID == "" {
				commentID = feed.ID
			}
			if text == "" {
				text = feed.Text
			}
			if senderName == "" {
				senderName = feed.From.Username
			}
			if commentID == "" {
				continue
			}

			eventType := events.CommentCreated
			if channel == providers.ChannelInstagram {
				eventType = events.InstagramCommentCreated
			}
			switch feed.Verb {
			case "edited":
				eventType = events.CommentUpdated
			case "remove", "removed":
				eventType = events.CommentDeleted
			}

			out = append(out, providers.NormalizedEvent{
				EventType: eventType,
				Channel:   channel,
				PageID:    pageID,
				Payload: map[string]any{
					"comment_id": commentID,
					"user_id":    senderID,
					"user_name":  senderName,
					"message":    text,
				},
			})
		}
	}
	return out, nil
}

// Ensure interface compliance.
var _ providers.Provider = (*FacebookProvider)(nil)
