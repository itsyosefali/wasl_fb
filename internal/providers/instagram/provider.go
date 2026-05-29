package instagram

import (
	"context"

	"github.com/pop/erp_meta/internal/providers"
	metapkg "github.com/pop/erp_meta/internal/providers/meta"
)

// Provider implements Provider for Instagram Business (Graph API via Meta).
type Provider struct {
	facebook *metapkg.FacebookProvider
}

func NewProvider(graphVersion, appSecret string) *Provider {
	return &Provider{facebook: metapkg.NewFacebookProvider(graphVersion, appSecret)}
}

func (p *Provider) Channel() providers.Channel { return providers.ChannelInstagram }

func (p *Provider) SendMessage(ctx context.Context, req providers.SendMessageRequest) (string, error) {
	return p.facebook.SendMessage(ctx, req)
}

func (p *Provider) SendImage(ctx context.Context, req providers.SendMediaRequest) (string, error) {
	return p.facebook.SendImage(ctx, req)
}

func (p *Provider) SendCarousel(ctx context.Context, req providers.SendCarouselRequest) (string, error) {
	return p.facebook.SendCarousel(ctx, req)
}

func (p *Provider) SendTemplate(ctx context.Context, req providers.SendTemplateRequest) (string, error) {
	return p.facebook.SendTemplate(ctx, req)
}

func (p *Provider) SendProduct(ctx context.Context, req providers.SendProductRequest) (string, error) {
	return p.facebook.SendProduct(ctx, req)
}

func (p *Provider) ReplyComment(ctx context.Context, req providers.ReplyCommentRequest) (string, error) {
	return p.facebook.ReplyComment(ctx, req)
}

func (p *Provider) PrivateReplyComment(ctx context.Context, req providers.ReplyCommentRequest) (string, error) {
	return p.facebook.PrivateReplyComment(ctx, req)
}

func (p *Provider) HideComment(ctx context.Context, commentID, accessToken string) error {
	return p.facebook.HideComment(ctx, commentID, accessToken)
}

func (p *Provider) VerifyWebhook(signatureHeader string, body []byte) bool {
	return p.facebook.VerifyWebhook(signatureHeader, body)
}

func (p *Provider) VerifyChallenge(mode, token, challenge, expectedToken string) (string, bool) {
	return p.facebook.VerifyChallenge(mode, token, challenge, expectedToken)
}

func (p *Provider) ParseWebhook(body []byte) ([]providers.NormalizedEvent, error) {
	events, err := p.facebook.ParseWebhook(body)
	if err != nil {
		return nil, err
	}
	for i := range events {
		events[i].Channel = providers.ChannelInstagram
	}
	return events, nil
}

var _ providers.Provider = (*Provider)(nil)
