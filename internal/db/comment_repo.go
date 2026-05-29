package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pop/erp_meta/internal/models"
)

type CommentRepo struct {
	pool *pgxpool.Pool
}

func NewCommentRepo(pool *pgxpool.Pool) *CommentRepo {
	return &CommentRepo{pool: pool}
}

func (r *CommentRepo) Create(ctx context.Context, c models.Comment) (models.Comment, error) {
	return r.CreateWithEvent(ctx, c, uuid.Nil)
}

func (r *CommentRepo) CreateWithEvent(ctx context.Context, c models.Comment, eventID uuid.UUID) (models.Comment, error) {
	if c.ID == uuid.Nil {
		c.ID = eventID
	}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO comments (id, tenant_id, external_id, page_id, contact_id, message, status, event_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8::uuid, '00000000-0000-0000-0000-000000000000'))
		 ON CONFLICT (tenant_id, external_id) DO UPDATE SET
		   message = EXCLUDED.message,
		   status = EXCLUDED.status,
		   event_id = COALESCE(EXCLUDED.event_id, comments.event_id)
		 RETURNING id, tenant_id, external_id, page_id, contact_id, message, status, created_at`,
		c.ID, c.TenantID, c.ExternalID, c.PageID, c.ContactID, c.Message, c.Status, eventID,
	).Scan(&c.ID, &c.TenantID, &c.ExternalID, &c.PageID, &c.ContactID, &c.Message, &c.Status, &c.CreatedAt)
	return c, err
}

func (r *CommentRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]models.Comment, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, external_id, page_id, contact_id, message, status, created_at
		 FROM comments WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2`,
		tenantID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var c models.Comment
		if err := rows.Scan(&c.ID, &c.TenantID, &c.ExternalID, &c.PageID, &c.ContactID, &c.Message, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (r *CommentRepo) GetByExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (models.Comment, error) {
	var c models.Comment
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, external_id, page_id, contact_id, message, status, created_at
		 FROM comments WHERE tenant_id = $1 AND external_id = $2`,
		tenantID, externalID,
	).Scan(&c.ID, &c.TenantID, &c.ExternalID, &c.PageID, &c.ContactID, &c.Message, &c.Status, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Comment{}, ErrNotFound
	}
	return c, err
}

func (r *CommentRepo) UpdateStatus(ctx context.Context, tenantID uuid.UUID, externalID, status string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE comments SET status = $3 WHERE tenant_id = $1 AND external_id = $2`,
		tenantID, externalID, status,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
