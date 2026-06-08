package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Conversation struct {
	ContactID     uuid.UUID `json:"contact_id"`
	PageID        uuid.UUID `json:"page_id"`
	ContactName   string    `json:"contact_name"`
	ExternalID    string    `json:"external_id"`
	Platform      string    `json:"platform"`
	PageName      string    `json:"page_name"`
	MetaPageID    string    `json:"meta_page_id"`
	LastMessage   string    `json:"last_message"`
	LastDirection string    `json:"last_direction"`
	LastMessageAt time.Time `json:"last_message_at"`
}

type ConversationRepo struct {
	pool *pgxpool.Pool
}

func NewConversationRepo(pool *pgxpool.Pool) *ConversationRepo {
	return &ConversationRepo{pool: pool}
}

func (r *ConversationRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, pageID *uuid.UUID, limit int) ([]Conversation, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT
			m.contact_id,
			m.page_id,
			c.name,
			c.external_id,
			c.platform,
			p.name,
			p.meta_page_id,
			(array_agg(m.message ORDER BY m.created_at DESC))[1] AS last_message,
			(array_agg(m.direction ORDER BY m.created_at DESC))[1] AS last_direction,
			max(m.created_at) AS last_message_at
		FROM messages m
		JOIN contacts c ON c.id = m.contact_id
		JOIN pages p ON p.id = m.page_id
		WHERE m.tenant_id = $1`
	args := []any{tenantID}
	if pageID != nil {
		query += ` AND m.page_id = $2`
		args = append(args, *pageID)
	}
	query += `
		GROUP BY m.contact_id, m.page_id, c.name, c.external_id, c.platform, p.name, p.meta_page_id
		ORDER BY last_message_at DESC
		LIMIT $` + fmt.Sprintf("%d", len(args)+1)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var conv Conversation
		if err := rows.Scan(
			&conv.ContactID, &conv.PageID, &conv.ContactName, &conv.ExternalID, &conv.Platform,
			&conv.PageName, &conv.MetaPageID, &conv.LastMessage, &conv.LastDirection, &conv.LastMessageAt,
		); err != nil {
			return nil, err
		}
		conversations = append(conversations, conv)
	}
	return conversations, rows.Err()
}

func (r *ConversationRepo) ListMessages(ctx context.Context, tenantID, contactID, pageID uuid.UUID, limit int) ([]MessageRow, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, external_id, page_id, contact_id, direction, message, created_at
		FROM messages
		WHERE tenant_id = $1 AND contact_id = $2 AND page_id = $3
		ORDER BY created_at ASC
		LIMIT $4`, tenantID, contactID, pageID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []MessageRow
	for rows.Next() {
		var m MessageRow
		if err := rows.Scan(&m.ID, &m.TenantID, &m.ExternalID, &m.PageID, &m.ContactID, &m.Direction, &m.Message, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

type MessageRow struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	ExternalID string
	PageID     uuid.UUID
	ContactID  uuid.UUID
	Direction  string
	Message    string
	CreatedAt  time.Time
}
