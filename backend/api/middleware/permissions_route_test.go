package middleware

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// List copy reads records, list paste inserts them: the permission check must know both
func TestMapRouteToOperationCopyPaste(t *testing.T) {
	app := fiber.New()
	app.All("/*", func(c *fiber.Ctx) error { return c.SendString(string(mapRouteToOperation(c))) })
	for _, tc := range []struct{ method, path, want string }{
		{"POST", "/api/tables/Customer/insert-many", "insert"},
		{"POST", "/api/tables/Customer/copy", "read"},
		{"POST", "/api/tables/Customer/insert", "insert"},
		{"DELETE", "/api/tables/Customer/delete-many", "delete"},
	} {
		resp, err := app.Test(httptest.NewRequest(tc.method, tc.path, nil))
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		if string(body) != tc.want {
			t.Errorf("%s %s = %q, want %q", tc.method, tc.path, body, tc.want)
		}
	}
}
