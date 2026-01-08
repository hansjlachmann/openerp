package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CORS returns configured CORS middleware
func CORS() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5173,http://localhost:3000", // Vite dev server
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	})
}
