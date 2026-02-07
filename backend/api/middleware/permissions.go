package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	apitypes "github.com/hansjlachmann/openerp/backend/api/types"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/session"
)

// PermissionCheck returns middleware that enforces table-level permissions.
// Maps HTTP method + path to a permission operation and checks against the session.
func PermissionCheck() fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess := session.GetCurrent()
		if sess == nil {
			// No session — let downstream handlers deal with it
			return c.Next()
		}

		// SUPER users bypass all permission checks
		if sess.IsSuper {
			return c.Next()
		}

		tableName := c.Params("table")
		if tableName == "" {
			return c.Next()
		}

		op := mapRouteToOperation(c)
		if op == "" {
			// Route doesn't map to a permission operation (e.g. unknown path)
			return c.Next()
		}

		if sess.HasPermission(tableName, op) {
			return c.Next()
		}

		language := sess.GetLanguage()
		opName := string(op)
		appErr := apperrors.PermissionDenied(opName, tableName)
		return c.Status(403).JSON(apitypes.NewErrorResponse(appErr.Message(language)))
	}
}

// mapRouteToOperation maps the HTTP method and path to a permission operation.
func mapRouteToOperation(c *fiber.Ctx) session.PermissionOperation {
	method := c.Method()
	path := c.Path()

	switch method {
	case "GET":
		return session.PermRead
	case "POST":
		if strings.HasSuffix(path, "/insert") {
			return session.PermInsert
		}
		if strings.HasSuffix(path, "/validate") {
			return session.PermRead
		}
		return ""
	case "PUT":
		return session.PermModify
	case "DELETE":
		return session.PermDelete
	default:
		return ""
	}
}
