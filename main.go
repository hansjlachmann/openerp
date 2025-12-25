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
	fmt.Println("=== OpenERP - Business Central Style ERP ===\n")

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
			fmt.Printf("\n[Database: %s", db.GetDatabasePath())
			if currentCompany != "" {
				fmt.Printf(" | Company: %s", currentCompany)
			}
			fmt.Printf("]\n")
		} else {
			fmt.Printf("\n[No database open]\n")
		}

		// Show menu
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("MAIN MENU")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("1. Create new database")
		fmt.Println("2. Open existing database")
		fmt.Println("3. Close database")
		fmt.Println("4. Create company (auto-initializes all tables)")
		fmt.Println("5. Enter company")
		fmt.Println("6. Exit company")
		fmt.Println("7. Delete company")
		fmt.Println("8. List companies")
		fmt.Println("9. List registered objects")
		fmt.Println("10. Exit application")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Print("\nSelect option (1-10): ")

		if !scanner.Scan() {
			break
		}

		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			// Create new database
			if db != nil {
				fmt.Println("✗ Error: A database is already open. Close it first.")
				continue
			}

			fmt.Print("\nEnter database filename (e.g., mycompany.db): ")
			if !scanner.Scan() {
				continue
			}
			path := strings.TrimSpace(scanner.Text())

			if path == "" {
				fmt.Println("✗ Error: Database filename cannot be empty")
				continue
			}

			// Add .db extension if not present
			if !strings.HasSuffix(path, ".db") {
				path += ".db"
			}

			newDB, err := database.CreateDatabase(path)
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
			} else {
				db = newDB
				companyMgr = company.NewManager(db, registry)
				fmt.Printf("✓ Database created: %s\n", path)
				fmt.Printf("✓ Object registry: %d table(s) registered\n", registry.GetTableCount())
			}

		case "2":
			// Open existing database
			if db != nil {
				fmt.Println("✗ Error: A database is already open. Close it first.")
				continue
			}

			fmt.Print("\nEnter database filename (e.g., mycompany.db): ")
			if !scanner.Scan() {
				continue
			}
			path := strings.TrimSpace(scanner.Text())

			if path == "" {
				fmt.Println("✗ Error: Database filename cannot be empty")
				continue
			}

			openedDB, err := database.OpenDatabase(path)
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
			} else {
				db = openedDB
				companyMgr = company.NewManager(db, registry)
				fmt.Printf("✓ Database opened: %s\n", path)
				fmt.Printf("✓ Object registry: %d table(s) registered\n", registry.GetTableCount())
			}

		case "3":
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

		case "4":
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

		case "5":
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

		case "6":
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

		case "7":
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

		case "8":
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

		case "9":
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

		case "10":
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
