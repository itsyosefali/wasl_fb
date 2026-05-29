package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pop/erp_meta/internal/models"
)

type MessageRepo struct {
	pool *pgxpool.Pool
}

func NewMessageRepo(pool *pgxpool.Pool) *MessageRepo {
	return &MessageRepo{pool: pool}
}

func (r *MessageRepo) Create(ctx context.Context, m models.Message) (models.Message, error) {
	return r.CreateWithEvent(ctx, m, uuid.Nil)
}

func (r *MessageRepo) CreateWithEvent(ctx context.Context, m models.Message, eventID uuid.UUID) (models.Message, error) {
	if m.ID == uuid.Nil {
		m.ID = eventID
	}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO messages (id, tenant_id, external_id, page_id, contact_id, direction, message, event_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8::uuid, '00000000-0000-0000-0000-000000000000'))
		 RETURNING id, tenant_id, external_id, page_id, contact_id, direction, message, created_at`,
		m.ID, m.TenantID, m.ExternalID, m.PageID, m.ContactID, m.Direction, m.Message, eventID,
	).Scan(&m.ID, &m.TenantID, &m.ExternalID, &m.PageID, &m.ContactID, &m.Direction, &m.Message, &m.CreatedAt)
	return m, err
}

func (r *MessageRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]models.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, external_id, page_id, contact_id, direction, message, created_at
		 FROM messages WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2`,
		tenantID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ID, &m.TenantID, &m.ExternalID, &m.PageID, &m.ContactID, &m.Direction, &m.Message, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (r *MessageRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (models.Message, error) {
	var m models.Message
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, external_id, page_id, contact_id, direction, message, created_at
		 FROM messages WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	).Scan(&m.ID, &m.TenantID, &m.ExternalID, &m.PageID, &m.ContactID, &m.Direction, &m.Message, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Message{}, ErrNotFound
	}
	return m, err
}
