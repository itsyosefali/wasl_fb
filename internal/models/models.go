package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	APIKey    string    `json:"api_key,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Page struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	MetaPageID  string    `json:"meta_page_id"`
	Name        string    `json:"name"`
	AccessToken string    `json:"-"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type Contact struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	Platform   string    `json:"platform"`
	ExternalID string    `json:"external_id"`
	Name       string    `json:"name"`
	Avatar     string    `json:"avatar,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Comment struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	ExternalID string    `json:"external_id"`
	PageID     uuid.UUID `json:"page_id"`
	ContactID  uuid.UUID `json:"contact_id"`
	Message    string    `json:"message"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type Message struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	ExternalID string    `json:"external_id,omitempty"`
	PageID     uuid.UUID `json:"page_id"`
	ContactID  uuid.UUID `json:"contact_id"`
	Direction  string    `json:"direction"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}

type Webhook struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	URL       string    `json:"url"`
	Secret    string    `json:"secret,omitempty"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type Event struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	EventType     string          `json:"event_type"`
	Channel       string          `json:"channel"`
	AggregateType string          `json:"aggregate_type,omitempty"`
	AggregateID   string          `json:"aggregate_id,omitempty"`
	Payload       json.RawMessage `json:"payload"`
	Status        string          `json:"status"`
	Attempts      int             `json:"attempts"`
	CreatedAt     time.Time       `json:"created_at"`
}

const (
	PageStatusActive   = "active"
	PageStatusInactive = "inactive"

	DirectionIn  = "in"
	DirectionOut = "out"

	CommentStatusVisible = "visible"
	CommentStatusHidden  = "hidden"

	EventStatusPending   = "pending"
	EventStatusDelivered = "delivered"
	EventStatusFailed    = "failed"

	PlatformFacebook  = "facebook"
	PlatformInstagram = "instagram"
	PlatformWhatsApp  = "whatsapp"
	PlatformTelegram  = "telegram"
)
