package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

func SetupJetStream(nc *nats.Conn) (nats.JetStreamContext, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      StreamName,
		Subjects:  []string{StreamSubject},
		Storage:   nats.FileStorage,
		Retention: nats.LimitsPolicy,
		MaxAge:    7 * 24 * time.Hour,
	})
	if err != nil && !errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
		info, infoErr := js.StreamInfo(StreamName)
		if infoErr != nil || info == nil {
			return nil, fmt.Errorf("add stream: %w", err)
		}
	}
	return js, nil
}

type Publisher struct {
	js nats.JetStreamContext
}

func NewPublisher(js nats.JetStreamContext) *Publisher {
	return &Publisher{js: js}
}

type Envelope struct {
	EventID   uuid.UUID       `json:"event_id"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	EventType string          `json:"event"`
	Channel   string          `json:"channel"`
	Payload   json.RawMessage `json:"payload"`
}

func (p *Publisher) Publish(tenantID uuid.UUID, eventType string, envelope Envelope) error {
	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	subject := Subject(tenantID.String(), eventType)
	_, err = p.js.Publish(subject, data)
	return err
}

type Subscriber struct {
	js nats.JetStreamContext
}

func NewSubscriber(js nats.JetStreamContext) *Subscriber {
	return &Subscriber{js: js}
}

func (s *Subscriber) Subscribe(handler func(Envelope) error) (*nats.Subscription, error) {
	return s.js.Subscribe(WildcardSubject(), func(msg *nats.Msg) {
		var env Envelope
		if err := json.Unmarshal(msg.Data, &env); err != nil {
			_ = msg.Ack()
			return
		}
		if err := handler(env); err != nil {
			_ = msg.Nak()
			return
		}
		_ = msg.Ack()
	}, nats.BindStream(StreamName), nats.Durable(ConsumerName), nats.ManualAck(), nats.DeliverNew())
}

// SubscribeAll is reserved for future fan-out consumers.
func (s *Subscriber) SubscribeAll(handler func(Envelope)) (*nats.Subscription, error) {
	return s.js.Subscribe(StreamSubject, func(msg *nats.Msg) {
		var env Envelope
		if err := json.Unmarshal(msg.Data, &env); err != nil {
			_ = msg.Ack()
			return
		}
		handler(env)
		_ = msg.Ack()
	}, nats.ManualAck())
}
