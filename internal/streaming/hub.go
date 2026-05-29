package streaming

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"github.com/pop/erp_meta/internal/events"
)

// Hub broadcasts events to connected WebSocket clients per tenant.
type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]map[chan []byte]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[uuid.UUID]map[chan []byte]struct{})}
}

func (h *Hub) Subscribe(tenantID uuid.UUID) (chan []byte, func()) {
	ch := make(chan []byte, 64)
	h.mu.Lock()
	if h.clients[tenantID] == nil {
		h.clients[tenantID] = make(map[chan []byte]struct{})
	}
	h.clients[tenantID][ch] = struct{}{}
	h.mu.Unlock()

	unsub := func() {
		h.mu.Lock()
		delete(h.clients[tenantID], ch)
		if len(h.clients[tenantID]) == 0 {
			delete(h.clients, tenantID)
		}
		h.mu.Unlock()
		close(ch)
	}
	return ch, unsub
}

func (h *Hub) Broadcast(tenantID uuid.UUID, envelope events.Envelope) {
	data, err := json.Marshal(envelope)
	if err != nil {
		return
	}
	h.broadcastBytes(tenantID, data)
}

func (h *Hub) broadcastBytes(tenantID uuid.UUID, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients[tenantID] {
		select {
		case ch <- data:
		default:
		}
	}
}

var _ events.Broadcaster = (*Hub)(nil)
