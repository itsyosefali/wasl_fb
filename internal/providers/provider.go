package providers

import "context"

// Channel identifies a communication provider.
type Channel string

const (
	ChannelFacebook  Channel = "facebook"
	ChannelInstagram Channel = "instagram"
	ChannelWhatsApp  Channel = "whatsapp"
	ChannelTelegram  Channel = "telegram"
)

// NormalizedEvent is a provider-agnostic inbound event.
type NormalizedEvent struct {
	EventType string
	Channel   Channel
	PageID    string // external page/account id
	Payload   map[string]any
}

type SendMessageRequest struct {
	PageID      string
	RecipientID string
	Text        string
	AccessToken string
}

type SendMediaRequest struct {
	PageID      string
	RecipientID string
	URL         string
	AccessToken string
}

type SendCarouselRequest struct {
	PageID      string
	RecipientID string
	Elements    []CarouselElement
	AccessToken string
}

type CarouselElement struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	URL      string `json:"url,omitempty"`
}

type SendTemplateRequest struct {
	PageID       string
	RecipientID  string
	TemplateName string
	Language     string
	Components   []map[string]any
	AccessToken  string
}

type SendProductRequest struct {
	PageID      string
	RecipientID string
	ProductID   string
	CatalogID   string
	AccessToken string
}

type ReplyCommentRequest struct {
	CommentID   string
	Text        string
	AccessToken string
}

// Provider is the channel-agnostic interface. Application code never imports Meta SDK details.
type Provider interface {
	Channel() Channel

	SendMessage(ctx context.Context, req SendMessageRequest) (externalID string, err error)
	SendImage(ctx context.Context, req SendMediaRequest) (externalID string, err error)
	SendCarousel(ctx context.Context, req SendCarouselRequest) (externalID string, err error)
	SendTemplate(ctx context.Context, req SendTemplateRequest) (externalID string, err error)
	SendProduct(ctx context.Context, req SendProductRequest) (externalID string, err error)

	ReplyComment(ctx context.Context, req ReplyCommentRequest) (externalID string, err error)
	PrivateReplyComment(ctx context.Context, req ReplyCommentRequest) (externalID string, err error)
	HideComment(ctx context.Context, commentID, accessToken string) error

	VerifyWebhook(signatureHeader string, body []byte) bool
	VerifyChallenge(mode, token, challenge, expectedToken string) (response string, ok bool)
	ParseWebhook(body []byte) ([]NormalizedEvent, error)
}
