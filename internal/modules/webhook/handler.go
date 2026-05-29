package webhook

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/pop/erp_meta/internal/ingest"
	"github.com/pop/erp_meta/internal/providers"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	verifyToken   string
	ingestService *ingest.Service
	registry      *providers.Registry
	redis         *redis.Client
}

func NewHandler(verifyToken string, ingestService *ingest.Service, registry *providers.Registry, redisClient *redis.Client) *Handler {
	return &Handler{
		verifyToken:   verifyToken,
		ingestService: ingestService,
		registry:      registry,
		redis:         redisClient,
	}
}

func (h *Handler) Register(app fiber.Router) {
	app.Get("/meta", h.VerifyMeta)
	app.Post("/meta", h.ReceiveMeta)
}

func (h *Handler) VerifyMeta(c *fiber.Ctx) error {
	provider, _ := h.registry.Get(providers.ChannelFacebook)
	if provider == nil {
		return c.SendStatus(fiber.StatusForbidden)
	}
	challenge, ok := provider.VerifyChallenge(
		c.Query("hub.mode"),
		c.Query("hub.verify_token"),
		c.Query("hub.challenge"),
		h.verifyToken,
	)
	if ok {
		return c.SendString(challenge)
	}
	return c.SendStatus(fiber.StatusForbidden)
}

func (h *Handler) ReceiveMeta(c *fiber.Ctx) error {
	body := c.Body()
	provider, _ := h.registry.Get(providers.ChannelFacebook)
	if provider == nil || !provider.VerifyWebhook(c.Get("X-Hub-Signature-256"), body) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	return h.receive(c, body, providers.ChannelFacebook)
}

func (h *Handler) receive(c *fiber.Ctx, body []byte, channel providers.Channel) error {
	deliveryID := c.Get("X-Hub-Delivery-ID")
	if deliveryID != "" {
		ctx := context.Background()
		key := fmt.Sprintf("webhook:delivery:%s", deliveryID)
		ok, err := h.redis.SetNX(ctx, key, "1", 24*time.Hour).Result()
		if err == nil && !ok {
			return c.SendStatus(fiber.StatusOK)
		}
	}

	processed, err := h.ingestService.ProcessPayload(c.Context(), body, channel)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"processed": processed})
}
