package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pop/erp_meta/internal/models"
)

type ContactRepo struct {
	pool *pgxpool.Pool
}

func NewContactRepo(pool *pgxpool.Pool) *ContactRepo {
	return &ContactRepo{pool: pool}
}

func (r *ContactRepo) Upsert(ctx context.Context, c models.Contact) (models.Contact, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO contacts (tenant_id, platform, external_id, name, avatar)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (tenant_id, platform, external_id)
		 DO UPDATE SET
		   name = CASE WHEN EXCLUDED.name != '' THEN EXCLUDED.name ELSE contacts.name END,
		   avatar = CASE WHEN EXCLUDED.avatar != '' THEN EXCLUDED.avatar ELSE contacts.avatar END
		 RETURNING id, tenant_id, platform, external_id, name, avatar, created_at`,
		c.TenantID, c.Platform, c.ExternalID, c.Name, c.Avatar,
	).Scan(&c.ID, &c.TenantID, &c.Platform, &c.ExternalID, &c.Name, &c.Avatar, &c.CreatedAt)
	return c, err
}

func (r *ContactRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]models.Contact, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, platform, external_id, name, avatar, created_at
		 FROM contacts WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2`,
		tenantID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []models.Contact
	for rows.Next() {
		var c models.Contact
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Platform, &c.ExternalID, &c.Name, &c.Avatar, &c.CreatedAt); err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

func (r *ContactRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (models.Contact, error) {
	var c models.Contact
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, platform, external_id, name, avatar, created_at
		 FROM contacts WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	).Scan(&c.ID, &c.TenantID, &c.Platform, &c.ExternalID, &c.Name, &c.Avatar, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Contact{}, ErrNotFound
	}
	return c, err
}
