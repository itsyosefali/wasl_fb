package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pop/erp_meta/internal/models"
)

type PageRepo struct {
	pool *pgxpool.Pool
}

func NewPageRepo(pool *pgxpool.Pool) *PageRepo {
	return &PageRepo{pool: pool}
}

func (r *PageRepo) Create(ctx context.Context, p models.Page) (models.Page, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO pages (tenant_id, meta_page_id, name, access_token, status)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, tenant_id, meta_page_id, name, access_token, status, created_at`,
		p.TenantID, p.MetaPageID, p.Name, p.AccessToken, p.Status,
	).Scan(&p.ID, &p.TenantID, &p.MetaPageID, &p.Name, &p.AccessToken, &p.Status, &p.CreatedAt)
	return p, err
}

// Upsert inserts a page or, if the (tenant_id, meta_page_id) pair already
// exists, refreshes its name, access token and status. Used by the OAuth flow.
func (r *PageRepo) Upsert(ctx context.Context, p models.Page) (models.Page, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO pages (tenant_id, meta_page_id, name, access_token, status)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (tenant_id, meta_page_id) DO UPDATE SET
		   name = EXCLUDED.name,
		   access_token = EXCLUDED.access_token,
		   status = EXCLUDED.status
		 RETURNING id, tenant_id, meta_page_id, name, access_token, status, created_at`,
		p.TenantID, p.MetaPageID, p.Name, p.AccessToken, p.Status,
	).Scan(&p.ID, &p.TenantID, &p.MetaPageID, &p.Name, &p.AccessToken, &p.Status, &p.CreatedAt)
	return p, err
}

func (r *PageRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]models.Page, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, meta_page_id, name, access_token, status, created_at
		 FROM pages WHERE tenant_id = $1 ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []models.Page
	for rows.Next() {
		var p models.Page
		if err := rows.Scan(&p.ID, &p.TenantID, &p.MetaPageID, &p.Name, &p.AccessToken, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

func (r *PageRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (models.Page, error) {
	var p models.Page
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, meta_page_id, name, access_token, status, created_at
		 FROM pages WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	).Scan(&p.ID, &p.TenantID, &p.MetaPageID, &p.Name, &p.AccessToken, &p.Status, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Page{}, ErrNotFound
	}
	return p, err
}

func (r *PageRepo) GetByMetaPageID(ctx context.Context, metaPageID string) (models.Page, error) {
	var p models.Page
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, meta_page_id, name, access_token, status, created_at
		 FROM pages WHERE meta_page_id = $1 AND status = 'active' LIMIT 1`,
		metaPageID,
	).Scan(&p.ID, &p.TenantID, &p.MetaPageID, &p.Name, &p.AccessToken, &p.Status, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Page{}, ErrNotFound
	}
	return p, err
}

func (r *PageRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM pages WHERE tenant_id = $1 AND id = $2`,
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
