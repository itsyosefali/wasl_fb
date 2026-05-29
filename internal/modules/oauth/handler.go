package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/pop/erp_meta/internal/crypto"
	"github.com/pop/erp_meta/internal/db"
	"github.com/pop/erp_meta/internal/events"
	"github.com/pop/erp_meta/internal/models"
	"github.com/pop/erp_meta/internal/providers"
	metapkg "github.com/pop/erp_meta/internal/providers/meta"
)

const (
	statePrefix      = "oauth:state:"
	stateTTL         = 10 * time.Minute
	webhookFields    = "messages,messaging_postbacks,messaging_optins,message_deliveries,message_reads,feed"
	instagramSubject = "instagram"
)

type Handler struct {
	oauth           *metapkg.OAuthClient
	tenantRepo      *db.TenantRepo
	pageRepo        *db.PageRepo
	store           *events.Store
	encryptor       *crypto.Encryptor
	redis           *redis.Client
	scopes          string
	successRedirect string
}

func NewHandler(
	oauth *metapkg.OAuthClient,
	tenantRepo *db.TenantRepo,
	pageRepo *db.PageRepo,
	store *events.Store,
	encryptor *crypto.Encryptor,
	redisClient *redis.Client,
	scopes string,
	successRedirect string,
) *Handler {
	return &Handler{
		oauth:           oauth,
		tenantRepo:      tenantRepo,
		pageRepo:        pageRepo,
		store:           store,
		encryptor:       encryptor,
		redis:           redisClient,
		scopes:          scopes,
		successRedirect: successRedirect,
	}
}

func (h *Handler) Register(r fiber.Router) {
	r.Get("/facebook", h.Start)
	r.Get("/facebook/callback", h.Callback)
}

// Start begins the Facebook Login flow.
// The browser hits GET /auth/facebook?api_key=... and is redirected to Meta.
func (h *Handler) Start(c *fiber.Ctx) error {
	if !h.oauth.Configured() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "facebook login is not configured; set META_APP_ID and META_APP_SECRET",
		})
	}

	apiKey := c.Query("api_key")
	if apiKey == "" {
		apiKey = c.Get("X-API-Key")
	}
	if apiKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "api_key query parameter is required",
		})
	}

	tenant, err := h.tenantRepo.GetByAPIKey(c.Context(), apiKey)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid api_key"})
	}

	state, err := randomState()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create state"})
	}

	if err := h.redis.Set(c.Context(), statePrefix+state, tenant.ID.String(), stateTTL).Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to persist state"})
	}

	return c.Redirect(h.oauth.AuthorizeURL(state, h.scopes), fiber.StatusFound)
}

// Callback handles Meta's redirect back, exchanges the code, and connects pages.
func (h *Handler) Callback(c *fiber.Ctx) error {
	if errParam := c.Query("error"); errParam != "" {
		reason := c.Query("error_description", errParam)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "facebook authorization was denied",
			"detail": reason,
		})
	}

	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing code or state"})
	}

	tenantID, err := h.consumeState(c.Context(), state)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid or expired state"})
	}

	shortToken, err := h.oauth.ExchangeCode(c.Context(), code)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": fmt.Sprintf("code exchange failed: %v", err)})
	}

	userToken := shortToken
	if longToken, _, err := h.oauth.LongLivedToken(c.Context(), shortToken); err == nil && longToken != "" {
		userToken = longToken
	}

	metaPages, err := h.oauth.ListPages(c.Context(), userToken)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": fmt.Sprintf("failed to list pages: %v", err)})
	}

	connected := make([]fiber.Map, 0, len(metaPages))
	for _, mp := range metaPages {
		encrypted, err := h.encryptor.Encrypt(mp.AccessToken)
		if err != nil {
			continue
		}

		page, err := h.pageRepo.Upsert(c.Context(), models.Page{
			TenantID:    tenantID,
			MetaPageID:  mp.ID,
			Name:        mp.Name,
			AccessToken: encrypted,
			Status:      models.PageStatusActive,
		})
		if err != nil {
			continue
		}

		// Subscribe the app to this page's webhook fields (best-effort).
		_ = h.oauth.SubscribeApp(c.Context(), mp.ID, mp.AccessToken, webhookFields)

		_, _ = h.store.Append(c.Context(), events.AppendInput{
			TenantID:      tenantID,
			EventType:     events.PageConnected,
			Channel:       providers.ChannelFacebook,
			AggregateType: events.AggregatePage,
			AggregateID:   page.MetaPageID,
			Payload: map[string]any{
				"page_id":     page.MetaPageID,
				"name":        page.Name,
				"internal_id": page.ID.String(),
				"via":         "oauth",
			},
		})

		connected = append(connected, fiber.Map{
			"id":           page.ID.String(),
			"meta_page_id": page.MetaPageID,
			"name":         page.Name,
		})
	}

	if h.successRedirect != "" {
		sep := "?"
		if hasQuery(h.successRedirect) {
			sep = "&"
		}
		return c.Redirect(fmt.Sprintf("%s%sconnected=%d", h.successRedirect, sep, len(connected)), fiber.StatusFound)
	}

	return c.JSON(fiber.Map{
		"connected_pages": len(connected),
		"pages":           connected,
	})
}

func (h *Handler) consumeState(ctx context.Context, state string) (uuid.UUID, error) {
	key := statePrefix + state
	val, err := h.redis.Get(ctx, key).Result()
	if err != nil {
		return uuid.Nil, err
	}
	_ = h.redis.Del(ctx, key).Err()
	return uuid.Parse(val)
}

func randomState() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hasQuery(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.RawQuery != ""
}
