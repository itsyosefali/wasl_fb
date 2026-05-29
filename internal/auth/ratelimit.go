package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client  *redis.Client
	limit   int
	window  time.Duration
}

func NewRateLimiter(client *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		client: client,
		limit:  limit,
		window: window,
	}
}

func (rl *RateLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			return c.Next()
		}

		key := fmt.Sprintf("ratelimit:%s", apiKey)
		ctx := context.Background()

		count, err := rl.client.Incr(ctx, key).Result()
		if err != nil {
			return c.Next()
		}
		if count == 1 {
			rl.client.Expire(ctx, key, rl.window)
		}
		if count > int64(rl.limit) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "rate limit exceeded",
			})
		}
		return c.Next()
	}
}
