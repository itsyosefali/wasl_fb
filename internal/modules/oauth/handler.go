package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/pop/erp_meta/internal/auth"
	"github.com/pop/erp_meta/internal/crypto"
	"github.com/pop/erp_meta/internal/db"
	"github.com/pop/erp_meta/internal/events"
	"github.com/pop/erp_meta/internal/models"
	"github.com/pop/erp_meta/internal/providers"
	metapkg "github.com/pop/erp_meta/internal/providers/meta"
)

const (
	statePrefix   = "oauth:state:"
	stateTTL      = 10 * time.Minute
	webhookFields = "messages,message_deliveries,message_echoes,message_reads,standby,messaging_handovers,feed"
	testScopes    = "public_profile"
)

type oauthState struct {
	TenantID uuid.UUID `json:"tenant_id"`
	Scopes   string    `json:"scopes"`
	TestMode bool      `json:"test_mode"`
}

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
	h.RegisterPublic(r)
}

func (h *Handler) RegisterPublic(r fiber.Router) {
	r.Get("/facebook", h.Start)
	r.Get("/facebook/callback", h.Callback)
}

func (h *Handler) RegisterAuthed(r fiber.Router) {
	r.Post("/facebook/pages", h.ListPagesForToken)
	r.Post("/facebook/register", h.RegisterPage)
}

type facebookPagesRequest struct {
	UserAccessToken string `json:"user_access_token"`
	OmniauthToken   string `json:"omniauth_token"`
}

type facebookRegisterRequest struct {
	UserAccessToken string `json:"user_access_token"`
	PageAccessToken string `json:"page_access_token"`
	PageID          string `json:"page_id"`
	Name            string `json:"name"`
	InboxName       string `json:"inbox_name"`
}

// ListPagesForToken mirrors Chatwoot POST /callbacks/facebook_pages.
func (h *Handler) ListPagesForToken(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	var req facebookPagesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	token := req.UserAccessToken
	if token == "" {
		token = req.OmniauthToken
	}
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_access_token required"})
	}

	longToken, _, _ := h.oauth.LongLivedToken(c.Context(), token)
	if longToken != "" {
		token = longToken
	}

	pages, err := h.oauth.ListPages(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}

	existing, _ := h.pageRepo.ListByTenant(c.Context(), tenant.ID)
	connected := map[string]bool{}
	for _, p := range existing {
		connected[p.MetaPageID] = true
	}

	details := make([]fiber.Map, 0, len(pages))
	for _, p := range pages {
		details = append(details, fiber.Map{
			"id":            p.ID,
			"name":          p.Name,
			"access_token":  p.AccessToken,
			"exists":        connected[p.ID],
		})
	}

	return c.JSON(fiber.Map{
		"user_access_token": token,
		"page_details":      details,
	})
}

// RegisterPage mirrors Chatwoot register_facebook_page.
func (h *Handler) RegisterPage(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	var req facebookRegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if req.PageID == "" || req.PageAccessToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "page_id and page_access_token required"})
	}
	name := req.Name
	if name == "" {
		name = req.InboxName
	}

	encrypted, err := h.encryptor.Encrypt(req.PageAccessToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "encrypt failed"})
	}

	page, err := h.pageRepo.Upsert(c.Context(), models.Page{
		TenantID:    tenant.ID,
		MetaPageID:  req.PageID,
		Name:        name,
		AccessToken: encrypted,
		Status:      models.PageStatusActive,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	_ = h.oauth.SubscribeApp(c.Context(), req.PageID, req.PageAccessToken, webhookFields)

	_, _ = h.store.Append(c.Context(), events.AppendInput{
		TenantID:      tenant.ID,
		EventType:     events.PageConnected,
		Channel:       providers.ChannelFacebook,
		AggregateType: events.AggregatePage,
		AggregateID:   page.MetaPageID,
		Payload: map[string]any{
			"page_id":     page.MetaPageID,
			"name":        page.Name,
			"internal_id": page.ID.String(),
			"via":         "fb_sdk",
		},
	})

	page.AccessToken = ""
	return c.Status(fiber.StatusCreated).JSON(page)
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

	scopes := h.scopes
	testMode := c.Query("test") == "1" || c.Query("test") == "true"
	if testMode {
		scopes = testScopes
	} else if q := strings.TrimSpace(c.Query("scopes")); q != "" {
		scopes = q
	}

	st := oauthState{TenantID: tenant.ID, Scopes: scopes, TestMode: testMode}
	raw, _ := json.Marshal(st)
	if err := h.redis.Set(c.Context(), statePrefix+state, string(raw), stateTTL).Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to persist state"})
	}

	return c.Redirect(h.oauth.AuthorizeURL(state, scopes), fiber.StatusFound)
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

	st, err := h.consumeState(c.Context(), state)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid or expired state"})
	}
	tenantID := st.TenantID

	shortToken, err := h.oauth.ExchangeCode(c.Context(), code)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": fmt.Sprintf("code exchange failed: %v", err)})
	}

	if st.TestMode {
		return c.JSON(fiber.Map{
			"status":  "oauth_ok",
			"message": "Facebook Login works. Add the Messenger use case in Meta dashboard, then connect again without ?test=1 to link Pages.",
			"hint":    fmt.Sprintf("https://developers.facebook.com/apps/%s/use_cases/", h.oauth.AppID()),
		})
	}

	userToken := shortToken
	if longToken, _, err := h.oauth.LongLivedToken(c.Context(), shortToken); err == nil && longToken != "" {
		userToken = longToken
	}

	metaPages, err := h.oauth.ListPages(c.Context(), userToken)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to list pages: %v", err),
			"hint":  "Ensure your Meta app has use case 'Engage with customers on Messenger from Meta' or 'Manage everything on your Page'.",
		})
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

func (h *Handler) consumeState(ctx context.Context, state string) (oauthState, error) {
	key := statePrefix + state
	val, err := h.redis.Get(ctx, key).Result()
	if err != nil {
		return oauthState{}, err
	}
	_ = h.redis.Del(ctx, key).Err()

	var st oauthState
	if err := json.Unmarshal([]byte(val), &st); err == nil && st.TenantID != uuid.Nil {
		return st, nil
	}
	// backward compat: value was plain tenant UUID string
	id, err := uuid.Parse(val)
	if err != nil {
		return oauthState{}, err
	}
	return oauthState{TenantID: id, Scopes: h.scopes}, nil
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
