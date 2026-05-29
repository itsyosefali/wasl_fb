package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pop/erp_meta/internal/models"
)

var ErrNotFound = errors.New("not found")

type TenantRepo struct {
	pool *pgxpool.Pool
}

func NewTenantRepo(pool *pgxpool.Pool) *TenantRepo {
	return &TenantRepo{pool: pool}
}

func (r *TenantRepo) GetByAPIKey(ctx context.Context, apiKey string) (models.Tenant, error) {
	var t models.Tenant
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, api_key, created_at FROM tenants WHERE api_key = $1`,
		apiKey,
	).Scan(&t.ID, &t.Name, &t.APIKey, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Tenant{}, ErrNotFound
	}
	return t, err
}

func (r *TenantRepo) GetByID(ctx context.Context, id uuid.UUID) (models.Tenant, error) {
	var t models.Tenant
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, api_key, created_at FROM tenants WHERE id = $1`,
		id,
	).Scan(&t.ID, &t.Name, &t.APIKey, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Tenant{}, ErrNotFound
	}
	return t, err
}
