package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/company"
	"github.com/hansjlachmann/openerp/src/foundation/database"
	"github.com/hansjlachmann/openerp/src/foundation/objects"
)

// Note: tables import is still needed for registering PaymentTerms

func main() {
	fmt.Println("=== OpenERP - PostgreSQL Edition ===\n")

	var db *database.Database
	var companyMgr *company.Manager
	var registry *objects.ObjectRegistry
	scanner := bufio.NewScanner(os.Stdin)

	// Initialize object registry
	registry = objects.NewObjectRegistry()

	// Register PaymentTerms table (Table ID: 3)
	if err := registry.RegisterTable(tables.PaymentTermsTableID, &tables.PaymentTerms{}); err != nil {
		fmt.Printf("Warning: Failed to register PaymentTerms: %v\n", err)
	}

	for {
		// Show current status
		if db != nil && companyMgr != nil {
			currentCompany := companyMgr.GetCurrentCompany()
			fmt.Printf("\n[Database: Connected")
			if currentCompany != "" {
				fmt.Printf(" | Company: %s", currentCompany)
			}
			fmt.Printf("]\n")
		} else {
			fmt.Printf("\n[No database connection]\n")
		}

		// Show menu
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("MAIN MENU")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("1. Connect to PostgreSQL database")
		fmt.Println("2. Close database connection")
		fmt.Println("3. Create company (auto-initializes all tables)")
		fmt.Println("4. Enter company")
		fmt.Println("5. Exit company")
		fmt.Println("6. Delete company")
		fmt.Println("7. List companies")
		fmt.Println("8. List registered objects")
		fmt.Println("9. Exit application")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Print("\nSelect option (1-9): ")

		if !scanner.Scan() {
			break
		}

		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			// Connect to PostgreSQL
			if db != nil {
				fmt.Println("✗ Error: Already connected to a database. Close it first.")
				continue
			}

			fmt.Print("\nEnter PostgreSQL host (default: localhost): ")
			scanner.Scan()
			host := strings.TrimSpace(scanner.Text())
			if host == "" {
				host = "localhost"
			}

			fmt.Print("Enter PostgreSQL port (default: 5432): ")
			scanner.Scan()
			port := strings.TrimSpace(scanner.Text())
			if port == "" {
				port = "5432"
			}

			fmt.Print("Enter PostgreSQL user (default: postgres): ")
			scanner.Scan()
			user := strings.TrimSpace(scanner.Text())
			if user == "" {
				user = "postgres"
			}

			fmt.Print("Enter PostgreSQL password: ")
			scanner.Scan()
			password := strings.TrimSpace(scanner.Text())

			fmt.Print("Enter database name (default: openerp): ")
			scanner.Scan()
			dbname := strings.TrimSpace(scanner.Text())
			if dbname == "" {
				dbname = "openerp"
			}

			newDB, err := database.CreateDatabase(host, port, user, password, dbname)
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
			} else {
				db = newDB
				companyMgr = company.NewManager(db, registry)
				fmt.Printf("✓ Connected to PostgreSQL database '%s'\n", dbname)
				fmt.Printf("✓ Object registry: %d table(s) registered\n", registry.GetTableCount())
			}

		case "2":
			// Close database
			if db == nil {
				fmt.Println("✗ Error: No database connection")
				continue
			}

			err := db.CloseDatabase()
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
			} else {
				fmt.Println("✓ Database connection closed")
				db = nil
				companyMgr = nil
			}

		case "3":
			// Create company
			if companyMgr == nil {
				fmt.Println("✗ Error: No database connection")
				continue
			}

			fmt.Print("\nEnter company name (e.g., CRONUS): ")
			if !scanner.Scan() {
				continue
			}
			name := strings.TrimSpace(scanner.Text())

			err := companyMgr.CreateCompany(name)
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
			} else {
				fmt.Printf("✓ Company '%s' created successfully\n", name)
			}

		case "4":
			// Enter company
			if companyMgr == nil {
				fmt.Println("✗ Error: No database connection")
				continue
			}

			if companyMgr.GetCurrentCompany() != "" {
				fmt.Printf("✗ Error: Already in company '%s'. Exit first.\n", companyMgr.GetCurrentCompany())
				continue
			}

			// Show available companies
			companies, err := companyMgr.ListCompanies()
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
				continue
			}

			if len(companies) == 0 {
				fmt.Println("✗ No companies available. Create one first.")
				continue
			}

			fmt.Println("\nAvailable companies:")
			for i, comp := range companies {
				fmt.Printf("  %d. %s\n", i+1, comp)
			}

			fmt.Print("\nEnter company name: ")
			if !scanner.Scan() {
				continue
			}
			name := strings.TrimSpace(scanner.Text())

			err = companyMgr.EnterCompany(name)
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
			} else {
				fmt.Printf("✓ Entered company '%s'\n", name)
			}

		case "5":
			// Exit company
			if companyMgr == nil {
				fmt.Println("✗ Error: No database connection")
				continue
			}

			err := companyMgr.ExitCompany()
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
			} else {
				fmt.Println("✓ Exited company session")
			}

		case "6":
			// Delete company
			if companyMgr == nil {
				fmt.Println("✗ Error: No database connection")
				continue
			}

			// Show available companies
			companies, err := companyMgr.ListCompanies()
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
				continue
			}

			if len(companies) == 0 {
				fmt.Println("✗ No companies to delete")
				continue
			}

			fmt.Println("\nAvailable companies:")
			for i, comp := range companies {
				fmt.Printf("  %d. %s\n", i+1, comp)
			}

			fmt.Print("\nEnter company name to delete: ")
			if !scanner.Scan() {
				continue
			}
			name := strings.TrimSpace(scanner.Text())

			// Confirm deletion
			fmt.Printf("\n⚠️  WARNING: This will delete company '%s' and ALL its tables!\n", name)
			fmt.Print("Are you sure? (yes/no): ")
			if !scanner.Scan() {
				continue
			}
			confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))

			if confirm != "yes" {
				fmt.Println("✓ Deletion cancelled")
				continue
			}

			err = companyMgr.DeleteCompany(name)
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
			} else {
				fmt.Printf("✓ Company '%s' and all its tables deleted successfully\n", name)
			}

		case "7":
			// List companies
			if companyMgr == nil {
				fmt.Println("✗ Error: No database connection")
				continue
			}

			companies, err := companyMgr.ListCompanies()
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
				continue
			}

			if len(companies) == 0 {
				fmt.Println("\nNo companies found in database")
			} else {
				fmt.Printf("\n✓ Found %d company/companies:\n", len(companies))
				for i, comp := range companies {
					if comp == companyMgr.GetCurrentCompany() {
						fmt.Printf("  %d. %s (ACTIVE)\n", i+1, comp)
					} else {
						fmt.Printf("  %d. %s\n", i+1, comp)
					}
				}
			}

		case "8":
			// List registered objects
			fmt.Println("\n=== Registered Objects ===")

			tableIDs := registry.ListTables()
			if len(tableIDs) > 0 {
				fmt.Println("\nTables:")
				for _, id := range tableIDs {
					fmt.Printf("  - Table %d (%s)\n", id, objects.GetObjectRange(id))
				}
			}

			pageIDs := registry.ListPages()
			if len(pageIDs) > 0 {
				fmt.Println("\nPages:")
				for _, id := range pageIDs {
					fmt.Printf("  - Page %d (%s)\n", id, objects.GetObjectRange(id))
				}
			}

			codeunitIDs := registry.ListCodeunits()
			if len(codeunitIDs) > 0 {
				fmt.Println("\nCodeunits:")
				for _, id := range codeunitIDs {
					fmt.Printf("  - Codeunit %d (%s)\n", id, objects.GetObjectRange(id))
				}
			}

			if len(tableIDs) == 0 && len(pageIDs) == 0 && len(codeunitIDs) == 0 {
				fmt.Println("No objects registered yet")
			}

		case "9":
			// Exit application
			if db != nil {
				fmt.Println("\nClosing database connection...")
				db.CloseDatabase()
			}
			fmt.Println("\n✓ Goodbye!")
			return

		default:
			fmt.Printf("✗ Invalid option: %s\n", choice)
		}
	}

	if scanner.Err() != nil {
		fmt.Printf("\n✗ Error reading input: %v\n", scanner.Err())
	}
}
