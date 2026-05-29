package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GraphClient is the low-level Meta Graph API HTTP client.
type GraphClient struct {
	graphVersion string
	httpClient   *http.Client
}

func NewGraphClient(graphVersion string) *GraphClient {
	return &GraphClient{
		graphVersion: graphVersion,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *GraphClient) graphURL(path string) string {
	return fmt.Sprintf("https://graph.facebook.com/%s/%s", c.graphVersion, path)
}

func (c *GraphClient) DoJSON(ctx context.Context, method, url string, body any, accessToken string, result any) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		q := req.URL.Query()
		q.Set("access_token", accessToken)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("graph api error %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return err
		}
	}
	return nil
}

// doGraphGet executes a prepared GET request and decodes the JSON response,
// surfacing Graph API error bodies on non-2xx status codes.
func doGraphGet(client *http.Client, req *http.Request, result any) error {
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("graph api error %d: %s", resp.StatusCode, string(body))
	}
	if result != nil && len(body) > 0 {
		if err := json.Unmarshal(body, result); err != nil {
			return err
		}
	}
	return nil
}

type graphIDResponse struct {
	ID        string `json:"id"`
	MessageID string `json:"message_id"`
}

func (c *GraphClient) SendRawMessage(ctx context.Context, pageID string, body map[string]any, accessToken string) (string, error) {
	var resp graphIDResponse
	if err := c.DoJSON(ctx, http.MethodPost, c.graphURL(pageID+"/messages"), body, accessToken, &resp); err != nil {
		return "", err
	}
	if resp.MessageID != "" {
		return resp.MessageID, nil
	}
	return resp.ID, nil
}

func (c *GraphClient) ReplyComment(ctx context.Context, commentID, text, accessToken string) (string, error) {
	body := map[string]string{"message": text}
	var resp graphIDResponse
	if err := c.DoJSON(ctx, http.MethodPost, c.graphURL(commentID+"/comments"), body, accessToken, &resp); err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (c *GraphClient) PrivateReplyComment(ctx context.Context, commentID, text, accessToken string) (string, error) {
	body := map[string]string{"message": text}
	var resp graphIDResponse
	if err := c.DoJSON(ctx, http.MethodPost, c.graphURL(commentID+"/private_replies"), body, accessToken, &resp); err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (c *GraphClient) HideComment(ctx context.Context, commentID, accessToken string) error {
	url := c.graphURL(commentID) + "?is_hidden=true"
	return c.DoJSON(ctx, http.MethodPost, url, nil, accessToken, nil)
}
