package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// RateLimit returns a generous, global rate limiter applied to every request.
// It is keyed by client IP (c.IP()); when running behind a reverse proxy (nginx),
// the app's ProxyHeader must be set so c.IP() reflects the real client address.
// State is kept in-memory per process — for multi-instance deployments this would
// need a shared store (e.g. Redis).
//
// Health checks are skipped so liveness/readiness probes are never throttled.
func RateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:          300,
		Expiration:   1 * time.Minute,
		LimitReached: rateLimitReached,
		Next: func(c *fiber.Ctx) bool {
			return c.Path() == "/health"
		},
	})
}

// LoginRateLimit returns a strict limiter for the login endpoint. Login runs a
// deliberately slow bcrypt password check, so an unbounded endpoint invites both
// brute-force/credential-stuffing and CPU exhaustion; capping attempts per IP
// blunts both.
func LoginRateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:          10,
		Expiration:   1 * time.Minute,
		LimitReached: rateLimitReached,
	})
}

// rateLimitReached responds with 429 in the app's standard {success, error} envelope.
func rateLimitReached(c *fiber.Ctx) error {
	return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
		"success": false,
		"error":   "Too many requests, please try again later",
	})
}
