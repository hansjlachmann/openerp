package middleware

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CORS returns configured CORS middleware
func CORS() fiber.Handler {
	// Allow configuration via environment variable
	// Default allows all origins when behind reverse proxy (nginx)
	allowOrigins := os.Getenv("CORS_ORIGINS")
	if allowOrigins == "" {
		// In production behind nginx, allow all origins
		// nginx handles the actual domain/origin validation
		allowOrigins = "*"
	}

	return cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: allowOrigins != "*", // Can't use credentials with wildcard
		MaxAge:           86400,               // 24 hours
	})
}
