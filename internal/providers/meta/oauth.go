package meta

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// OAuthClient handles the Facebook Login (OAuth) flow for connecting Pages.
type OAuthClient struct {
	appID        string
	appSecret    string
	graphVersion string
	redirectURL  string
	httpClient   *http.Client
}

func NewOAuthClient(appID, appSecret, graphVersion, redirectURL string) *OAuthClient {
	return &OAuthClient{
		appID:        appID,
		appSecret:    appSecret,
		graphVersion: graphVersion,
		redirectURL:  redirectURL,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *OAuthClient) Configured() bool {
	return c.appID != "" && c.appSecret != ""
}

// AuthorizeURL builds the Facebook OAuth dialog URL the user is redirected to.
func (c *OAuthClient) AuthorizeURL(state, scopes string) string {
	q := url.Values{}
	q.Set("client_id", c.appID)
	q.Set("redirect_uri", c.redirectURL)
	q.Set("state", state)
	q.Set("scope", scopes)
	q.Set("response_type", "code")
	return fmt.Sprintf("https://www.facebook.com/%s/dialog/oauth?%s", c.graphVersion, q.Encode())
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// ExchangeCode swaps an OAuth authorization code for a short-lived user access token.
func (c *OAuthClient) ExchangeCode(ctx context.Context, code string) (string, error) {
	q := url.Values{}
	q.Set("client_id", c.appID)
	q.Set("client_secret", c.appSecret)
	q.Set("redirect_uri", c.redirectURL)
	q.Set("code", code)

	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/oauth/access_token?%s", c.graphVersion, q.Encode())

	var resp tokenResponse
	if err := c.get(ctx, endpoint, &resp); err != nil {
		return "", err
	}
	if resp.AccessToken == "" {
		return "", fmt.Errorf("empty access token from code exchange")
	}
	return resp.AccessToken, nil
}

// LongLivedToken exchanges a short-lived user token for a long-lived one (~60 days).
func (c *OAuthClient) LongLivedToken(ctx context.Context, shortToken string) (string, int64, error) {
	q := url.Values{}
	q.Set("grant_type", "fb_exchange_token")
	q.Set("client_id", c.appID)
	q.Set("client_secret", c.appSecret)
	q.Set("fb_exchange_token", shortToken)

	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/oauth/access_token?%s", c.graphVersion, q.Encode())

	var resp tokenResponse
	if err := c.get(ctx, endpoint, &resp); err != nil {
		return "", 0, err
	}
	if resp.AccessToken == "" {
		return "", 0, fmt.Errorf("empty long-lived token")
	}
	return resp.AccessToken, resp.ExpiresIn, nil
}

// OAuthPage is a Page returned from /me/accounts during the OAuth flow.
type OAuthPage struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AccessToken string `json:"access_token"`
}

type accountsResponse struct {
	Data   []OAuthPage `json:"data"`
	Paging struct {
		Next string `json:"next"`
	} `json:"paging"`
}

// ListPages fetches all Pages the user granted access to, with per-Page tokens.
func (c *OAuthClient) ListPages(ctx context.Context, userToken string) ([]OAuthPage, error) {
	q := url.Values{}
	q.Set("access_token", userToken)
	q.Set("fields", "id,name,access_token")
	q.Set("limit", "100")

	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/me/accounts?%s", c.graphVersion, q.Encode())

	var pages []OAuthPage
	for endpoint != "" {
		var resp accountsResponse
		if err := c.get(ctx, endpoint, &resp); err != nil {
			return nil, err
		}
		pages = append(pages, resp.Data...)
		endpoint = resp.Paging.Next
	}
	return pages, nil
}

// SubscribeApp subscribes the app to a Page's webhook fields so events flow in.
func (c *OAuthClient) SubscribeApp(ctx context.Context, pageID, pageToken, fields string) error {
	q := url.Values{}
	q.Set("access_token", pageToken)
	q.Set("subscribed_fields", fields)

	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s/subscribed_apps?%s", c.graphVersion, pageID, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("subscribe app failed: status %d", resp.StatusCode)
	}
	return nil
}

func (c *OAuthClient) get(ctx context.Context, endpoint string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	return doGraphGet(c.httpClient, req, result)
}
