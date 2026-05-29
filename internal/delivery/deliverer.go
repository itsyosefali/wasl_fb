package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pop/erp_meta/internal/db"
	"github.com/pop/erp_meta/internal/events"
	metasig "github.com/pop/erp_meta/internal/providers/meta"
	"github.com/pop/erp_meta/internal/models"
)

type Deliverer struct {
	webhookRepo *db.WebhookRepo
	eventRepo   *db.EventRepo
	httpClient  *http.Client
	maxAttempts int
	backoffs    []time.Duration
}

func NewDeliverer(webhookRepo *db.WebhookRepo, eventRepo *db.EventRepo, maxAttempts int, backoffs []time.Duration) *Deliverer {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	if len(backoffs) == 0 {
		backoffs = []time.Duration{1 * time.Second, 5 * time.Second, 30 * time.Second, 2 * time.Minute, 10 * time.Minute}
	}
	return &Deliverer{
		webhookRepo: webhookRepo,
		eventRepo:   eventRepo,
		httpClient:  &http.Client{Timeout: 15 * time.Second},
		maxAttempts: maxAttempts,
		backoffs:    backoffs,
	}
}

func (d *Deliverer) Handle(ctx context.Context, env events.Envelope) error {
	webhooks, err := d.webhookRepo.ListEnabledByTenant(ctx, env.TenantID)
	if err != nil {
		return err
	}
	if len(webhooks) == 0 {
		return d.eventRepo.UpdateStatus(ctx, env.EventID, models.EventStatusDelivered, 0, "no webhooks configured")
	}

	body, err := buildDeliveryBody(env)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 1; attempt <= d.maxAttempts; attempt++ {
		allOK := true
		for _, wh := range webhooks {
			if err := d.postWebhook(ctx, wh, body); err != nil {
				lastErr = err
				allOK = false
			}
		}
		if allOK {
			return d.eventRepo.UpdateStatus(ctx, env.EventID, models.EventStatusDelivered, attempt, "")
		}
		if attempt < d.maxAttempts {
			idx := attempt - 1
			if idx >= len(d.backoffs) {
				idx = len(d.backoffs) - 1
			}
			time.Sleep(d.backoffs[idx])
		}
		_ = d.eventRepo.UpdateStatus(ctx, env.EventID, models.EventStatusPending, attempt, lastErr.Error())
	}

	status := models.EventStatusFailed
	if lastErr != nil {
		return d.eventRepo.UpdateStatus(ctx, env.EventID, status, d.maxAttempts, lastErr.Error())
	}
	return d.eventRepo.UpdateStatus(ctx, env.EventID, status, d.maxAttempts, "delivery failed")
}

func buildDeliveryBody(env events.Envelope) ([]byte, error) {
	payload := map[string]any{
		"event_id":  env.EventID.String(),
		"event":     env.EventType,
		"tenant_id": env.TenantID.String(),
	}
	var inner map[string]any
	if err := json.Unmarshal(env.Payload, &inner); err == nil {
		for k, v := range inner {
			payload[k] = v
		}
	}
	return json.Marshal(payload)
}

func (d *Deliverer) postWebhook(ctx context.Context, wh models.Webhook, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gateway-Signature", metasig.SignPayload(wh.Secret, body))
	req.Header.Set("X-Gateway-Event-ID", uuid.New().String())

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook %s returned %d", wh.URL, resp.StatusCode)
	}
	return nil
}
