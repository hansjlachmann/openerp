package api

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/hansjlachmann/openerp/src/api/handlers"
	"github.com/hansjlachmann/openerp/src/api/middleware"
)

// Server represents the API server
type Server struct {
	app *fiber.App
	db  *sql.DB
}

// NewServer creates a new API server
func NewServer(db *sql.DB) *Server {
	app := fiber.New(fiber.Config{
		AppName:      "OpenERP API v1.0",
		ServerHeader: "OpenERP",
		ErrorHandler: customErrorHandler,
	})

	return &Server{
		app: app,
		db:  db,
	}
}

// Setup configures all routes and middleware
func (s *Server) Setup() {
	// Global middleware
	s.app.Use(recover.New()) // Panic recovery
	s.app.Use(middleware.CORS())
	s.app.Use(middleware.Logger())

	// API routes
	api := s.app.Group("/api")

	// Initialize handlers
	sessionHandler := handlers.NewSessionHandler()
	tablesHandler := handlers.NewTablesHandler(s.db)
	pagesHandler := handlers.NewPagesHandler()
	preferencesHandler := handlers.NewPreferencesHandler(s.db)
	authHandler := handlers.NewAuthHandler(s.db)

	// Auth routes
	api.Post("/auth/login", authHandler.Login)
	api.Post("/auth/logout", authHandler.Logout)
	api.Get("/auth/user", authHandler.GetCurrentUser)
	api.Post("/auth/init", authHandler.CreateInitialUser)
	api.Get("/auth/companies", authHandler.ListCompanies)

	// Session routes
	api.Get("/session", sessionHandler.GetSession)

	// Table routes
	tables := api.Group("/tables/:table")
	tables.Get("/ids", tablesHandler.GetRecordIDs)        // Lightweight IDs-only endpoint
	tables.Get("/list", tablesHandler.ListRecords)
	tables.Get("/card/:id", tablesHandler.GetRecord)
	tables.Post("/insert", tablesHandler.InsertRecord)
	tables.Put("/modify/:id", tablesHandler.ModifyRecord)
	tables.Delete("/delete/:id", tablesHandler.DeleteRecord)
	tables.Post("/validate", tablesHandler.ValidateField)

	// Page routes
	api.Get("/pages", pagesHandler.GetAllPages)
	api.Get("/pages/:id", pagesHandler.GetPage)
	api.Get("/menu", pagesHandler.GetMenu)

	// Preferences routes
	api.Get("/preferences/:page_id/:type", preferencesHandler.GetPreferences)
	api.Post("/preferences/:page_id/:type", preferencesHandler.SavePreference)
	api.Delete("/preferences/:page_id/:type/:name", preferencesHandler.DeletePreference)

	// Health check
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"service": "openerp-api",
		})
	})

	// 404 handler
	s.app.Use(func(c *fiber.Ctx) error {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Route not found",
		})
	})
}

// Start starts the API server
func (s *Server) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	log.Printf("🚀 API Server starting on http://localhost%s\n", addr)
	log.Printf("📡 Health check: http://localhost%s/health\n", addr)
	log.Printf("📚 API base: http://localhost%s/api\n", addr)
	return s.app.Listen(addr)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}

// customErrorHandler handles fiber errors
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"error":   err.Error(),
	})
}
