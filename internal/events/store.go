package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/pop/erp_meta/internal/db"
	"github.com/pop/erp_meta/internal/models"
	"github.com/pop/erp_meta/internal/providers"
)

// AppendInput is the input for appending an event to the store.
type AppendInput struct {
	TenantID      uuid.UUID
	EventType     string
	Channel       providers.Channel
	AggregateType string
	AggregateID   string
	Payload       map[string]any
}

// Broadcaster pushes events to live subscribers (e.g. WebSocket clients).
type Broadcaster interface {
	Broadcast(tenantID uuid.UUID, envelope Envelope)
}

// Store is the event-sourced write model. Events are the source of truth.
type Store struct {
	eventRepo   *db.EventRepo
	projector   *Projector
	publisher   *Publisher
	broadcaster Broadcaster
}

func NewStore(eventRepo *db.EventRepo, projector *Projector, publisher *Publisher) *Store {
	return &Store{
		eventRepo: eventRepo,
		projector: projector,
		publisher: publisher,
	}
}

func (s *Store) SetBroadcaster(b Broadcaster) {
	s.broadcaster = b
}

func (s *Store) Append(ctx context.Context, input AppendInput) (models.Event, error) {
	raw, err := json.Marshal(input.Payload)
	if err != nil {
		return models.Event{}, err
	}

	ev, err := s.eventRepo.Create(ctx, models.Event{
		TenantID:      input.TenantID,
		EventType:     input.EventType,
		Channel:       string(input.Channel),
		AggregateType: input.AggregateType,
		AggregateID:   input.AggregateID,
		Payload:       raw,
		Status:        models.EventStatusPending,
	})
	if err != nil {
		return models.Event{}, err
	}

	if err := s.projector.Project(ctx, ev); err != nil {
		return ev, fmt.Errorf("project event: %w", err)
	}

	envelope := Envelope{
		EventID:   ev.ID,
		TenantID:  input.TenantID,
		EventType: input.EventType,
		Channel:   string(input.Channel),
		Payload:   raw,
	}
	if err := s.publisher.Publish(input.TenantID, input.EventType, envelope); err != nil {
		return ev, fmt.Errorf("publish event: %w", err)
	}

	if s.broadcaster != nil {
		s.broadcaster.Broadcast(input.TenantID, envelope)
	}

	return ev, nil
}
