package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/hansjlachmann/openerp/src/api"
	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/company"
	"github.com/hansjlachmann/openerp/src/foundation/database"
	"github.com/hansjlachmann/openerp/src/foundation/objects"
	"github.com/hansjlachmann/openerp/src/foundation/session"
)

func main() {
	fmt.Println("=== OpenERP API Server ===\n")

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

	// Prompt for database
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter database path (or press Enter for 'test.db'): ")
	scanner.Scan()
	dbPath := strings.TrimSpace(scanner.Text())
	if dbPath == "" {
		dbPath = "test.db"
	}

	// Open database
	db, err := database.OpenDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.CloseDatabase()

	fmt.Printf("✓ Database opened: %s\n", dbPath)

	// Prompt for company
	fmt.Print("Enter company name (or press Enter for 'cronus'): ")
	scanner.Scan()
	companyName := strings.TrimSpace(scanner.Text())
	if companyName == "" {
		companyName = "cronus"
	}

	// Enter company
	companyMgr := company.NewManager(db, registry)
	if err := companyMgr.EnterCompany(companyName); err != nil {
		log.Fatalf("Failed to enter company: %v", err)
	}

	fmt.Printf("✓ Company entered: %s\n", companyName)

	// Create a default session for API access
	sess := session.NewSession(db, companyName, nil)
	sess.SetUser("api-user", "API User", "en-US")
	session.SetCurrent(sess)

	fmt.Printf("✓ Session created for API access\n\n")

	// Create and setup API server
	server := api.NewServer(db.GetConnection())
	server.Setup()

	// Start server in a goroutine
	port := 8080
	go func() {
		if err := server.Start(port); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	fmt.Println("\n✅ API Server is running!")
	fmt.Println("📝 Press Ctrl+C to stop the server\n")
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
