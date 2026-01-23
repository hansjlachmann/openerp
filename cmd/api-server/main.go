package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/hansjlachmann/openerp/backend/api"
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/company"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/objects"
	"github.com/hansjlachmann/openerp/backend/foundation/session"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	fmt.Println("=== OpenERP API Server ===")

	// Initialize object registry
	registry := objects.NewObjectRegistry()

	// Register tables
	if err := registry.RegisterTable(tables.PaymentTermsTableID, &tables.PaymentTerms{}); err != nil {
		log.Printf("Warning: Failed to register PaymentTerms: %v\n", err)
	}
	if err := registry.RegisterTable(tables.CustomerTableID, &tables.Customer{}); err != nil {
		log.Printf("Warning: Failed to register Customer: %v\n", err)
	}
	if err := registry.RegisterTable(tables.CustomerLedgerEntryTableID, &tables.CustomerLedgerEntry{}); err != nil {
		log.Printf("Warning: Failed to register Customer Ledger Entry: %v\n", err)
	}
	if err := registry.RegisterTable(tables.UserTableID, &tables.User{}); err != nil {
		log.Printf("Warning: Failed to register User: %v\n", err)
	}
	if err := registry.RegisterTable(tables.UserPreferencesTableID, &tables.UserPreferences{}); err != nil {
		log.Printf("Warning: Failed to register UserPreferences: %v\n", err)
	}
	if err := registry.RegisterTable(gtables.LanguageTableID, &tables.Language{}); err != nil {
		log.Printf("Warning: Failed to register Language: %v\n", err)
	}
	if err := registry.RegisterTable(gtables.MenuTableID, &tables.Menu{}); err != nil {
		log.Printf("Warning: Failed to register Menu: %v\n", err)
	}

	var db *database.Database
	var companyName string
	var port string
	var err error

	// Check if we're in Docker/production mode (env vars set)
	dbHost := os.Getenv("DB_HOST")
	if dbHost != "" {
		// PostgreSQL mode (Docker/production)
		config := &database.PostgresConfig{
			Host:     dbHost,
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "openerp"),
			Password: getEnv("DB_PASSWORD", "openerp"),
			DBName:   getEnv("DB_NAME", "openerp"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		}

		fmt.Printf("Connecting to PostgreSQL at %s:%s...\n", config.Host, config.Port)

		db, err = database.CreateOrOpenPostgresDatabase(config)
		if err != nil {
			log.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}

		companyName = getEnv("COMPANY_NAME", "cronus")
		port = getEnv("PORT", "8080")

		fmt.Printf("✓ Connected to PostgreSQL database: %s\n", config.DBName)
	} else {
		// SQLite mode (local development)
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("Enter database path (or press Enter for 'test.db'): ")
		scanner.Scan()
		dbPath := strings.TrimSpace(scanner.Text())
		if dbPath == "" {
			dbPath = "test.db"
		}

		db, err = database.OpenDatabase(dbPath)
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}

		fmt.Printf("✓ Database opened: %s\n", dbPath)

		fmt.Print("Enter company name (or press Enter for 'cronus'): ")
		scanner.Scan()
		companyName = strings.TrimSpace(scanner.Text())
		if companyName == "" {
			companyName = "cronus"
		}

		port = "8080"
	}

	defer db.CloseDatabase()

	// Enter company (auto-create if it doesn't exist in Docker mode)
	companyMgr := company.NewManager(db, registry)
	if err := companyMgr.EnterCompany(companyName); err != nil {
		// In Docker mode, auto-create the company if it doesn't exist
		if dbHost != "" && strings.Contains(err.Error(), "does not exist") {
			fmt.Printf("Company '%s' not found, creating...\n", companyName)
			if createErr := companyMgr.CreateCompany(companyName); createErr != nil {
				log.Fatalf("Failed to create company: %v", createErr)
			}
			fmt.Printf("✓ Company created: %s\n", companyName)

			// Now enter the newly created company
			if enterErr := companyMgr.EnterCompany(companyName); enterErr != nil {
				log.Fatalf("Failed to enter company: %v", enterErr)
			}
		} else {
			log.Fatalf("Failed to enter company: %v", err)
		}
	}

	fmt.Printf("✓ Company entered: %s\n", companyName)

	// Create a default session for API access
	sess := session.NewSession(db, companyName, nil)
	sess.SetUser("api-user", "API User", "en-US", "admin")
	session.SetCurrent(sess)

	fmt.Printf("✓ Session created for API access\n\n")

	// Create and setup API server
	var server *api.Server
	if dbHost != "" {
		// PostgreSQL mode
		server = api.NewServerWithDBType(db.GetConnection(), database.DBTypePostgres)
	} else {
		// SQLite mode
		server = api.NewServer(db.GetConnection())
	}
	server.Setup()

	// Start server in a goroutine
	go func() {
		if err := server.StartOnPort(port); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	fmt.Println("\n✅ API Server is running!")
	fmt.Printf("📡 Listening on port %s\n", port)
	fmt.Println("📝 Press Ctrl+C to stop the server")
	fmt.Println("Available endpoints:")
	fmt.Println("  GET    /health")
	fmt.Println("  GET    /api/session")
	fmt.Println("  GET    /api/tables/Customer/list")
	fmt.Println("  GET    /api/tables/Customer/card/:id")
	fmt.Println("  POST   /api/tables/Customer/insert")
	fmt.Println("  PUT    /api/tables/Customer/modify/:id")
	fmt.Println("  DELETE /api/tables/Customer/delete/:id")
	fmt.Println("  POST   /api/tables/Customer/validate")
	fmt.Println("\n(Same endpoints available for Payment_terms and Customer_ledger_entry)")

	<-quit
	fmt.Println("\n🛑 Shutting down server...")

	if err := server.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	fmt.Println("✓ Server stopped")
}
