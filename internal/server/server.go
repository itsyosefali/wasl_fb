package server

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"

	"github.com/pop/erp_meta/internal/actions"
	"github.com/pop/erp_meta/internal/auth"
	"github.com/pop/erp_meta/internal/config"
	"github.com/pop/erp_meta/internal/crypto"
	"github.com/pop/erp_meta/internal/db"
	"github.com/pop/erp_meta/internal/events"
	"github.com/pop/erp_meta/internal/ingest"
	actionmod "github.com/pop/erp_meta/internal/modules/actions"
	"github.com/pop/erp_meta/internal/modules/comments"
	"github.com/pop/erp_meta/internal/modules/contacts"
	eventmod "github.com/pop/erp_meta/internal/modules/events"
	"github.com/pop/erp_meta/internal/modules/messages"
	"github.com/pop/erp_meta/internal/modules/oauth"
	"github.com/pop/erp_meta/internal/modules/pages"
	"github.com/pop/erp_meta/internal/modules/stream"
	"github.com/pop/erp_meta/internal/modules/webhook"
	"github.com/pop/erp_meta/internal/modules/webhooks"
	"github.com/pop/erp_meta/internal/providers"
	"github.com/pop/erp_meta/internal/providers/instagram"
	metapkg "github.com/pop/erp_meta/internal/providers/meta"
	"github.com/pop/erp_meta/internal/providers/telegram"
	"github.com/pop/erp_meta/internal/providers/whatsapp"
	"github.com/pop/erp_meta/internal/streaming"
)

type App struct {
	fiber *fiber.App
	pool  interface{ Close() }
	nc    *nats.Conn
	redis *redis.Client
}

func New(cfg config.Config) (*App, error) {
	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("redis url: %w", err)
	}
	redisClient := redis.NewClient(opt)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		pool.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	nc, err := nats.Connect(cfg.NATSURL,
		nats.Name("meta-gateway-api"),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		pool.Close()
		redisClient.Close()
		return nil, fmt.Errorf("nats: %w", err)
	}

	js, err := events.SetupJetStream(nc)
	if err != nil {
		pool.Close()
		redisClient.Close()
		nc.Close()
		return nil, fmt.Errorf("jetstream: %w", err)
	}

	encryptor, err := crypto.NewEncryptor(cfg.EncryptionKey)
	if err != nil {
		pool.Close()
		redisClient.Close()
		nc.Close()
		return nil, fmt.Errorf("encryptor: %w", err)
	}

	registry := providers.NewRegistry(
		metapkg.NewFacebookProvider(cfg.MetaGraphVersion, cfg.MetaAppSecret),
		instagram.NewProvider(cfg.MetaGraphVersion, cfg.MetaAppSecret),
		whatsapp.NewProvider(cfg.MetaAppSecret),
		telegram.NewProvider(),
	)

	tenantRepo := db.NewTenantRepo(pool)
	pageRepo := db.NewPageRepo(pool)
	contactRepo := db.NewContactRepo(pool)
	messageRepo := db.NewMessageRepo(pool)
	commentRepo := db.NewCommentRepo(pool)
	webhookRepo := db.NewWebhookRepo(pool)
	eventRepo := db.NewEventRepo(pool)

	projector := events.NewProjector(contactRepo, messageRepo, commentRepo)
	publisher := events.NewPublisher(js)
	hub := streaming.NewHub()

	store := events.NewStore(eventRepo, projector, publisher)
	store.SetBroadcaster(hub)

	ingestService := ingest.NewService(pageRepo, contactRepo, store, registry)
	actionExecutor := actions.NewExecutor(registry, pageRepo, store, encryptor)

	app := fiber.New(fiber.Config{
		AppName:      "Meta Gateway API",
		BodyLimit:    4 * 1024 * 1024,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	})

	app.Use(recover.New())
	app.Use(logger.New())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	webhook.NewHandler(cfg.MetaVerifyToken, ingestService, registry, redisClient).
		Register(app.Group("/webhooks"))

	oauthClient := metapkg.NewOAuthClient(cfg.MetaAppID, cfg.MetaAppSecret, cfg.MetaGraphVersion, cfg.MetaOAuthRedirectURL)
	oauth.NewHandler(oauthClient, tenantRepo, pageRepo, store, encryptor, redisClient, cfg.MetaOAuthScopes, cfg.OAuthSuccessRedirect).
		Register(app.Group("/auth"))

	authMiddleware := auth.Middleware(tenantRepo)
	rateLimiter := auth.NewRateLimiter(redisClient, cfg.RateLimitRequests, cfg.RateLimitWindow())

	api := app.Group("/", rateLimiter.Middleware(), authMiddleware)

	pages.NewHandler(pageRepo, store, encryptor).Register(api.Group("/pages"))
	messages.NewHandler(messageRepo, pageRepo, contactRepo, store, registry, encryptor).Register(api.Group("/messages"))
	comments.NewHandler(commentRepo, pageRepo, store, registry, encryptor).Register(api.Group("/comments"))
	contacts.NewHandler(contactRepo).Register(api.Group("/contacts"))
	webhooks.NewHandler(webhookRepo).Register(api.Group("/webhooks"))
	eventmod.NewHandler(eventRepo).Register(api.Group("/events"))
	stream.NewHandler(hub).Register(api.Group("/events"))
	actionmod.NewHandler(actionExecutor).Register(api.Group("/actions"))

	return &App{
		fiber: app,
		pool:  pool,
		nc:    nc,
		redis: redisClient,
	}, nil
}

func (a *App) Listen(addr string) error {
	log.Printf("starting API on %s", addr)
	return a.fiber.Listen(addr)
}

func (a *App) Shutdown() error {
	if a.nc != nil {
		a.nc.Close()
	}
	if a.redis != nil {
		_ = a.redis.Close()
	}
	if a.pool != nil {
		a.pool.Close()
	}
	return a.fiber.Shutdown()
}
