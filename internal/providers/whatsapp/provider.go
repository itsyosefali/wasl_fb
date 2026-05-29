package whatsapp

import (
	"context"
	"fmt"

	"github.com/pop/erp_meta/internal/providers"
)

// Provider is a stub for future WhatsApp Cloud API integration.
type Provider struct {
	appSecret string
}

func NewProvider(appSecret string) *Provider {
	return &Provider{appSecret: appSecret}
}

func (p *Provider) Channel() providers.Channel { return providers.ChannelWhatsApp }

func (p *Provider) notImplemented(action string) error {
	return fmt.Errorf("whatsapp provider: %s not implemented yet", action)
}

func (p *Provider) SendMessage(ctx context.Context, req providers.SendMessageRequest) (string, error) {
	return "", p.notImplemented("send_message")
}

func (p *Provider) SendImage(ctx context.Context, req providers.SendMediaRequest) (string, error) {
	return "", p.notImplemented("send_image")
}

func (p *Provider) SendCarousel(ctx context.Context, req providers.SendCarouselRequest) (string, error) {
	return "", p.notImplemented("send_carousel")
}

func (p *Provider) SendTemplate(ctx context.Context, req providers.SendTemplateRequest) (string, error) {
	return "", p.notImplemented("send_template")
}

func (p *Provider) SendProduct(ctx context.Context, req providers.SendProductRequest) (string, error) {
	return "", p.notImplemented("send_product")
}

func (p *Provider) ReplyComment(ctx context.Context, req providers.ReplyCommentRequest) (string, error) {
	return "", p.notImplemented("reply_comment")
}

func (p *Provider) PrivateReplyComment(ctx context.Context, req providers.ReplyCommentRequest) (string, error) {
	return "", p.notImplemented("private_reply")
}

func (p *Provider) HideComment(ctx context.Context, commentID, accessToken string) error {
	return p.notImplemented("hide_comment")
}

func (p *Provider) VerifyWebhook(signatureHeader string, body []byte) bool {
	return false
}

func (p *Provider) VerifyChallenge(mode, token, challenge, expectedToken string) (string, bool) {
	return "", false
}

func (p *Provider) ParseWebhook(body []byte) ([]providers.NormalizedEvent, error) {
	return nil, p.notImplemented("parse_webhook")
}

var _ providers.Provider = (*Provider)(nil)
