# Meta Gateway API

Developer-first **customer communication infrastructure** for Facebook, Instagram, WhatsApp, and Telegram. Exposes channel interactions through a clean REST API, webhooks, WebSocket streaming, and event-driven architecture — without business logic (no CRM, ERP, or AI baked in).

Inspired by [WuzAPI](https://github.com/asternic/wuzapi) and [Evolution API](https://github.com/EvolutionAPI/evolution-api).

## Architecture

```
Channel Webhooks → Provider Layer → Event Store → JetStream
                                        ↓
                              Projector (materialized views)
                                        ↓
                    Worker (webhook delivery) + WebSocket clients
```

| Layer | Purpose |
|-------|---------|
| **Providers** | Channel-agnostic interface (`facebook`, `instagram`, `whatsapp`, `telegram`) |
| **Event Store** | `events` table is the source of truth |
| **Projector** | Materializes `messages` / `comments` read models from events |
| **JetStream** | Durable event bus with replay |
| **WebSocket** | `GET /events/stream` for live event subscription |
| **Actions** | `POST /actions/execute` for ERP-driven custom actions |

## Quick Start

```bash
cp .env.example .env
docker compose up --build -d
curl http://localhost:8080/health
./scripts/verify.sh
```

Demo tenant API key: `demo-api-key-change-in-production`

## Provider Layer

All channel logic lives in `internal/providers/`. Application code never imports Meta SDK details directly.

```
internal/providers/
├── provider.go      # Provider interface
├── registry.go      # Channel registry
├── meta/            # Facebook / Messenger (Graph API)
├── instagram/       # Instagram Business
├── whatsapp/        # Stub (future)
└── telegram/        # Stub (future)
```

```go
type Provider interface {
    Channel() Channel
    SendMessage(...)
    SendImage(...)
    SendCarousel(...)
    SendTemplate(...)
    SendProduct(...)
    ReplyComment(...)
    HideComment(...)
    VerifyWebhook(...)
    ParseWebhook(...)
}
```

## Event-Sourced Model

Events are appended first; read models are projected:

| Event | Materialized View |
|-------|-------------------|
| `message.received` | `messages` (direction: in) |
| `message.sent` | `messages` (direction: out) |
| `comment.created` | `comments` |
| `comment.replied` | `comments` |
| `comment.hidden` | `comments` (status: hidden) |

SQL views `messages_view` and `comments_view` are available for analytics queries directly from the event log.

## API Reference

All tenant endpoints require `X-API-Key`.

### Events (source of truth)

```http
GET /events              # List events
GET /events/stream       # WebSocket live stream
```

WebSocket example:

```javascript
const ws = new WebSocket('ws://localhost:8080/events/stream', {
  headers: { 'X-API-Key': 'demo-api-key-change-in-production' }
});
ws.onmessage = (e) => console.log(JSON.parse(e.data));
```

### Custom Actions

ERP systems can drive the gateway with structured actions instead of low-level API calls:

```http
POST /actions/execute
```

```json
{
  "action": "send_product",
  "channel": "facebook",
  "page_id": "123456789",
  "recipient_id": "user_psid",
  "data": {
    "product_id": "123",
    "catalog_id": "cat_1"
  }
}
```

Supported actions:

| Action | Description |
|--------|-------------|
| `send_message` | Text message |
| `send_image` | Image attachment (`data.url`) |
| `send_carousel` | Generic template carousel (`data.elements`) |
| `send_template` | Template message (`data.template_name`) |
| `send_product` | Product template (`data.product_id`) |
| `reply_comment` | Public comment reply |
| `private_reply` | Private comment reply |
| `hide_comment` | Hide comment |

### Connect a Page with Facebook Login (OAuth)

Instead of manually pasting a Page access token, tenants can connect Pages
through the Facebook Login flow:

```http
GET /auth/facebook?api_key=<TENANT_API_KEY>   # redirects to Meta OAuth dialog
GET /auth/facebook/callback                    # Meta redirects here
```

Flow:

1. Point the browser at `/auth/facebook?api_key=...`
2. The gateway stores a short-lived `state` in Redis and redirects to Meta
3. User approves Page permissions (`pages_messaging`, `pages_manage_metadata`, ...)
4. Meta redirects to `/auth/facebook/callback?code=...&state=...`
5. The gateway exchanges the code for a long-lived user token, lists the user's
   Pages, encrypts and stores each Page token, subscribes the app to each Page's
   webhooks, and emits a `page.connected` event per Page

Setup (in `.env`):

```bash
META_APP_ID=...
META_APP_SECRET=...
META_OAUTH_REDIRECT_URL=https://your-domain/auth/facebook/callback
# Optional: redirect the browser to your dashboard after success
OAUTH_SUCCESS_REDIRECT=https://your-dashboard/connected
```

The redirect URI must be whitelisted under "Valid OAuth Redirect URIs" in the
Meta App's Facebook Login settings. Without `META_APP_ID`/`META_APP_SECRET` the
endpoint returns `503`; manual `POST /pages/connect` still works.

### Pages, Messages, Comments, Contacts, Webhooks

```http
GET/POST/DELETE /pages
POST /messages/send
GET  /messages
POST /comments/{reply,private-reply,hide}
GET  /comments
GET  /contacts
GET/POST/DELETE /webhooks
```

### Inbound Webhooks (from Meta)

```http
GET/POST /webhooks/meta
```

## Integration Flow

```
Customer comments "Price?" on Facebook
  → Meta webhook → Provider parses → event: comment.created
  → JetStream → Worker delivers to ERP webhook
  → ERP responds: { "action": "reply_comment", "text": "120 LYD" }
  → POST /actions/execute → Provider → Meta Graph API
```

## Configuration

| Variable | Description |
|----------|-------------|
| `NATS_URL` | NATS server (JetStream enabled) |
| `META_APP_SECRET` | Webhook HMAC verification |
| `META_GRAPH_VERSION` | Graph API version (default `v23.0`) |
| `ENCRYPTION_KEY` | 64-char hex AES-256 key for token encryption |

## Project Structure

```
cmd/api/              HTTP + WebSocket server
cmd/worker/           JetStream consumer → webhook delivery
internal/
  providers/          Channel abstraction (meta, instagram, whatsapp, telegram)
  events/             Event store, projector, JetStream pub/sub
  actions/            Custom action executor
  streaming/          WebSocket hub
  ingest/             Inbound webhook normalization
  modules/            REST handlers
migrations/           Postgres schema + event-sourced views
```

## License

MIT
