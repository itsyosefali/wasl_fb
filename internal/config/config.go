package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	APIHost string `envconfig:"API_HOST" default:"0.0.0.0"`
	APIPort int    `envconfig:"API_PORT" default:"8080"`

	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`
	RedisURL    string `envconfig:"REDIS_URL" required:"true"`
	NATSURL     string `envconfig:"NATS_URL" required:"true"`

	MetaAppSecret     string `envconfig:"META_APP_SECRET" required:"true"`
	MetaVerifyToken   string `envconfig:"META_VERIFY_TOKEN" required:"true"`
	MetaGraphVersion  string `envconfig:"META_GRAPH_VERSION" default:"v23.0"`
	EncryptionKey     string `envconfig:"ENCRYPTION_KEY" required:"true"`

	MetaAppID            string `envconfig:"META_APP_ID"`
	MetaOAuthRedirectURL string `envconfig:"META_OAUTH_REDIRECT_URL" default:"http://localhost:8080/auth/facebook/callback"`
	MetaOAuthScopes      string `envconfig:"META_OAUTH_SCOPES" default:"pages_show_list,pages_messaging,pages_manage_metadata,pages_read_engagement,pages_manage_engagement,instagram_basic,instagram_manage_messages,business_management"`
	OAuthSuccessRedirect string `envconfig:"OAUTH_SUCCESS_REDIRECT"`

	RateLimitRequests      int `envconfig:"RATE_LIMIT_REQUESTS" default:"100"`
	RateLimitWindowSeconds int `envconfig:"RATE_LIMIT_WINDOW_SECONDS" default:"60"`

	DeliveryMaxAttempts   int    `envconfig:"DELIVERY_MAX_ATTEMPTS" default:"5"`
	DeliveryBackoffSeconds string `envconfig:"DELIVERY_BACKOFF_SECONDS" default:"1,5,30,120,600"`
}

func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.APIHost, c.APIPort)
}

func (c Config) RateLimitWindow() time.Duration {
	return time.Duration(c.RateLimitWindowSeconds) * time.Second
}

func (c Config) DeliveryBackoffs() []time.Duration {
	parts := strings.Split(c.DeliveryBackoffSeconds, ",")
	backoffs := make([]time.Duration, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		secs, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		backoffs = append(backoffs, time.Duration(secs)*time.Second)
	}
	if len(backoffs) == 0 {
		return []time.Duration{1 * time.Second, 5 * time.Second, 30 * time.Second, 2 * time.Minute, 10 * time.Minute}
	}
	return backoffs
}

func Load() (Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
