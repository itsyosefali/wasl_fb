package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pop/erp_meta/internal/models"
)

type WebhookRepo struct {
	pool *pgxpool.Pool
}

func NewWebhookRepo(pool *pgxpool.Pool) *WebhookRepo {
	return &WebhookRepo{pool: pool}
}

func (r *WebhookRepo) Create(ctx context.Context, w models.Webhook) (models.Webhook, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO webhooks (tenant_id, url, secret, enabled)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, tenant_id, url, secret, enabled, created_at`,
		w.TenantID, w.URL, w.Secret, w.Enabled,
	).Scan(&w.ID, &w.TenantID, &w.URL, &w.Secret, &w.Enabled, &w.CreatedAt)
	return w, err
}

func (r *WebhookRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]models.Webhook, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, url, secret, enabled, created_at
		 FROM webhooks WHERE tenant_id = $1 ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []models.Webhook
	for rows.Next() {
		var w models.Webhook
		if err := rows.Scan(&w.ID, &w.TenantID, &w.URL, &w.Secret, &w.Enabled, &w.CreatedAt); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, w)
	}
	return webhooks, rows.Err()
}

func (r *WebhookRepo) ListEnabledByTenant(ctx context.Context, tenantID uuid.UUID) ([]models.Webhook, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, url, secret, enabled, created_at
		 FROM webhooks WHERE tenant_id = $1 AND enabled = TRUE`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []models.Webhook
	for rows.Next() {
		var w models.Webhook
		if err := rows.Scan(&w.ID, &w.TenantID, &w.URL, &w.Secret, &w.Enabled, &w.CreatedAt); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, w)
	}
	return webhooks, rows.Err()
}

func (r *WebhookRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM webhooks WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
