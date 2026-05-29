package stream

import (
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/pop/erp_meta/internal/auth"
	"github.com/pop/erp_meta/internal/models"
	"github.com/pop/erp_meta/internal/streaming"
)

const tenantLocalKey = "tenant"

type Handler struct {
	hub *streaming.Hub
}

func NewHandler(hub *streaming.Hub) *Handler {
	return &Handler{hub: hub}
}

func (h *Handler) Register(r fiber.Router) {
	r.Use("/stream", h.wsUpgrade)
	r.Get("/stream", websocket.New(h.handle))
}

func (h *Handler) wsUpgrade(c *fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	c.Locals(tenantLocalKey, tenant)
	return c.Next()
}

func (h *Handler) handle(conn *websocket.Conn) {
	tenant, ok := conn.Locals(tenantLocalKey).(models.Tenant)
	if !ok {
		_ = conn.Close()
		return
	}

	ch, unsub := h.hub.Subscribe(tenant.ID)
	defer unsub()
	defer conn.Close()

	_ = conn.WriteJSON(fiber.Map{
		"type":      "connected",
		"tenant_id": tenant.ID.String(),
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}
}
