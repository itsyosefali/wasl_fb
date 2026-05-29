package db

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pop/erp_meta/internal/models"
)

type EventRepo struct {
	pool *pgxpool.Pool
}

func NewEventRepo(pool *pgxpool.Pool) *EventRepo {
	return &EventRepo{pool: pool}
}

func (r *EventRepo) Create(ctx context.Context, e models.Event) (models.Event, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO events (tenant_id, event_type, channel, aggregate_type, aggregate_id, payload, status, attempts)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, tenant_id, event_type, channel, aggregate_type, aggregate_id, payload, status, attempts, created_at`,
		e.TenantID, e.EventType, e.Channel, e.AggregateType, e.AggregateID, e.Payload, e.Status, e.Attempts,
	).Scan(&e.ID, &e.TenantID, &e.EventType, &e.Channel, &e.AggregateType, &e.AggregateID, &e.Payload, &e.Status, &e.Attempts, &e.CreatedAt)
	return e, err
}

func (r *EventRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]models.Event, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, event_type, channel, aggregate_type, aggregate_id, payload, status, attempts, created_at
		 FROM events WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2`,
		tenantID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var e models.Event
		if err := rows.Scan(&e.ID, &e.TenantID, &e.EventType, &e.Channel, &e.AggregateType, &e.AggregateID, &e.Payload, &e.Status, &e.Attempts, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *EventRepo) GetByID(ctx context.Context, id uuid.UUID) (models.Event, error) {
	var e models.Event
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, event_type, channel, aggregate_type, aggregate_id, payload, status, attempts, created_at
		 FROM events WHERE id = $1`,
		id,
	).Scan(&e.ID, &e.TenantID, &e.EventType, &e.Channel, &e.AggregateType, &e.AggregateID, &e.Payload, &e.Status, &e.Attempts, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Event{}, ErrNotFound
	}
	return e, err
}

func (r *EventRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string, attempts int, lastError string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE events SET status = $2, attempts = $3, last_error = $4, updated_at = NOW() WHERE id = $1`,
		id, status, attempts, lastError,
	)
	return err
}

func (r *EventRepo) MarshalPayload(v any) (json.RawMessage, error) {
	return json.Marshal(v)
}
