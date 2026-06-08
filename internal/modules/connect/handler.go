package connect

import (
	"github.com/gofiber/fiber/v2"
)

// Handler serves a simple local connect / setup page (no auth required).
type Handler struct {
	apiBase   string
	metaAppID string
}

func NewHandler(apiBase, metaAppID string) *Handler {
	if apiBase == "" {
		apiBase = "http://localhost:8080"
	}
	return &Handler{apiBase: apiBase, metaAppID: metaAppID}
}

func (h *Handler) Register(r fiber.Router) {
	r.Get("/connect", h.Page)
}

func (h *Handler) Page(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(pageHTML(h.apiBase, h.metaAppID))
}

func pageHTML(apiBase, metaAppID string) string {
	useCasesURL := "https://developers.facebook.com/apps/"
	if metaAppID != "" {
		useCasesURL = "https://developers.facebook.com/apps/" + metaAppID + "/use_cases/"
	}
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
  <title>Meta Gateway — Connect</title>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 720px; margin: 2rem auto; padding: 0 1rem; line-height: 1.5; }
    h1 { font-size: 1.5rem; }
    .box { background: #f4f6f8; border-radius: 8px; padding: 1rem 1.25rem; margin: 1rem 0; }
    .warn { background: #fff3cd; border: 1px solid #ffc107; }
    .ok { background: #d1e7dd; border: 1px solid #198754; }
    code, pre { background: #eee; padding: 2px 6px; border-radius: 4px; font-size: 0.9em; }
    pre { padding: 12px; overflow-x: auto; }
    a.btn { display: inline-block; margin: 0.5rem 0.5rem 0.5rem 0; padding: 0.6rem 1rem;
            background: #1877f2; color: #fff; text-decoration: none; border-radius: 6px; font-weight: 600; }
    a.btn.secondary { background: #6c757d; }
    ol li { margin: 0.4rem 0; }
  </style>
</head>
<body>
  <h1>Meta Gateway — Connect a Page</h1>

  <div class="box warn">
    <strong>Seeing “Invalid Scopes: pages_show_list, pages_messaging”?</strong>
    <p>Your Meta app has no <em>use case</em> that enables Page permissions. Fix this in the Meta dashboard first (steps below). This is not a bug in the gateway.</p>
  </div>

  <h2>Step 1 — Enable permissions on your Meta app</h2>
  <ol>
    <li>Open <a href="` + useCasesURL + `" target="_blank">App → Use cases</a></li>
    <li>Click <strong>Add use cases</strong></li>
    <li>Select <strong>Engage with customers on Messenger from Meta</strong>
      <br><small>(or <strong>Manage everything on your Page</strong> for comments + posts)</small></li>
    <li>Save / customize the use case</li>
    <li>Under <strong>Facebook Login → Settings</strong>, add redirect URI:<br>
      <code>` + apiBase + `/auth/facebook/callback</code></li>
  </ol>

  <h2>Step 2 — Webhooks (Messenger events)</h2>
  <div class="box warn">
    <p><strong>Do not</strong> use the OAuth callback URL here. Webhooks are a separate endpoint.</p>
    <p>Meta cannot reach <code>localhost</code> from the internet. Use <a href="https://ngrok.com/" target="_blank">ngrok</a> (or similar) and put your public URL below.</p>
  </div>
  <table style="width:100%; border-collapse:collapse; margin:0.5rem 0;">
    <tr><td style="padding:0.35rem 0"><strong>Callback URL</strong></td><td><code>https://YOUR-TUNNEL.ngrok.io/webhooks/meta</code></td></tr>
    <tr><td style="padding:0.35rem 0"><strong>Verify token</strong></td><td><code>test</code></td></tr>
  </table>
  <p>Local test (same verify token):</p>
  <pre>curl "` + apiBase + `/webhooks/meta?hub.mode=subscribe&amp;hub.verify_token=test&amp;hub.challenge=ok123"
# should return: ok123</pre>
  <p>Subscribe to: <code>messages</code>, <code>message_deliveries</code>, <code>message_echoes</code>, <code>message_reads</code>, <code>feed</code></p>

  <h2>Step 3 — Connect with Facebook Login</h2>
  <p>API key (demo tenant): <code>demo-api-key-change-in-production</code></p>
  <a class="btn" href="` + apiBase + `/auth/facebook?api_key=demo-api-key-change-in-production">Connect with Facebook</a>
  <a class="btn secondary" href="` + apiBase + `/auth/facebook?api_key=demo-api-key-change-in-production&amp;test=1">Test OAuth only (public_profile)</a>

  <h2>Step 4 — Or connect manually (Graph API Explorer)</h2>
  <div class="box">
    <ol>
      <li><a href="https://developers.facebook.com/tools/explorer/" target="_blank">Graph API Explorer</a> → select your app</li>
      <li>Add permission <code>pages_show_list</code> → Generate token → query <code>me/accounts</code></li>
      <li>Copy <strong>Page ID</strong> and <strong>access_token</strong> from the response</li>
      <li>POST to the gateway:</li>
    </ol>
    <pre>curl -X POST ` + apiBase + `/pages/connect \
  -H "X-API-Key: demo-api-key-change-in-production" \
  -H "Content-Type: application/json" \
  -d '{"meta_page_id":"PAGE_ID","name":"My Page","access_token":"PAGE_TOKEN"}'</pre>
  </div>

  <h2>Verify</h2>
  <pre>curl -H "X-API-Key: demo-api-key-change-in-production" ` + apiBase + `/pages
curl ` + apiBase + `/health</pre>

  <p><a href="` + apiBase + `/health">Health</a> · API base: <code>` + apiBase + `</code></p>
</body>
</html>`
}
