package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Logger returns a simple logging middleware
func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process request
		err := c.Next()

		// Log request
		duration := time.Since(start)
		status := c.Response().StatusCode()

		// Color code based on status
		statusColor := "\033[32m" // Green for 2xx
		if status >= 400 && status < 500 {
			statusColor = "\033[33m" // Yellow for 4xx
		} else if status >= 500 {
			statusColor = "\033[31m" // Red for 5xx
		}

		fmt.Printf("%s[%d]\033[0m %s %-7s %s (%v)\n",
			statusColor,
			status,
			c.Method(),
			c.Path(),
			c.IP(),
			duration,
		)

		return err
	}
}
