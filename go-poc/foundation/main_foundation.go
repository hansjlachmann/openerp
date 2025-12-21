package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/hansjlachmann/openerp-go/object_designer"
)

func main() {
	fmt.Println("=== OpenERP Foundation Layer - Interactive Menu ===\n")

	var db *Database
	scanner := bufio.NewScanner(os.Stdin)

	for {
		// Show current status
		if db != nil {
			fmt.Printf("\n[Database: %s", db.path)
			if db.currentCompany != "" {
				fmt.Printf(" | Company: %s", db.currentCompany)
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
		fmt.Println("4. Create company")
		fmt.Println("5. Enter company")
		fmt.Println("6. Exit company")
		fmt.Println("7. Delete company")
		fmt.Println("8. List companies")
		fmt.Println("9. Object Designer")
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
			fmt.Print("\nEnter database path (e.g., openerp.db): ")
			if !scanner.Scan() {
				continue
			}
			path := strings.TrimSpace(scanner.Text())

			if path == "" {
				fmt.Println("✗ Error: Database path cannot be empty")
				continue
			}

			// Check if file already exists
			if _, err := os.Stat(path); err == nil {
				fmt.Printf("✗ Error: File '%s' already exists. Use option 2 to open it.\n", path)
				continue
			}

			newDB, err := CreateDatabase(path)
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
			} else {
				db = newDB
				fmt.Printf("✓ Database created successfully: %s\n", path)
			}

		case "2":
			// Open existing database
			if db != nil {
				fmt.Println("✗ Error: A database is already open. Close it first.")
				continue
			}

			fmt.Print("\nEnter database path (e.g., openerp.db): ")
			if !scanner.Scan() {
				continue
			}
			path := strings.TrimSpace(scanner.Text())

			if path == "" {
				fmt.Println("✗ Error: Database path cannot be empty")
				continue
			}

			openedDB, err := OpenDatabase(path)
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
			} else {
				db = openedDB
				fmt.Printf("✓ Database opened successfully: %s\n", path)
			}

		case "3":
			// Close database
			if db == nil {
				fmt.Println("✗ Error: No database is open")
				continue
			}

			err := db.CloseDatabase()
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
			} else {
				fmt.Println("✓ Database closed successfully")
				db = nil
			}

		case "4":
			// Create company
			if db == nil {
				fmt.Println("✗ Error: No database is open")
				continue
			}

			fmt.Print("\nEnter company name (e.g., ACME): ")
			if !scanner.Scan() {
				continue
			}
			name := strings.TrimSpace(scanner.Text())

			err := db.CreateCompany(name)
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
			} else {
				fmt.Printf("✓ Company '%s' created successfully\n", name)
			}

		case "5":
			// Enter company
			if db == nil {
				fmt.Println("✗ Error: No database is open")
				continue
			}

			if db.currentCompany != "" {
				fmt.Printf("✗ Error: Already in company '%s'. Exit first.\n", db.currentCompany)
				continue
			}

			// Show available companies
			companies, err := db.ListCompanies()
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

			err = db.EnterCompany(name)
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
			} else {
				fmt.Printf("✓ Entered company '%s'\n", name)
			}

		case "6":
			// Exit company
			if db == nil {
				fmt.Println("✗ Error: No database is open")
				continue
			}

			err := db.ExitCompany()
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
			} else {
				fmt.Println("✓ Exited company session")
			}

		case "7":
			// Delete company
			if db == nil {
				fmt.Println("✗ Error: No database is open")
				continue
			}

			// Show available companies
			companies, err := db.ListCompanies()
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

			err = db.DeleteCompany(name)
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
			} else {
				fmt.Printf("✓ Company '%s' and all its tables deleted successfully\n", name)
			}

		case "8":
			// List companies
			if db == nil {
				fmt.Println("✗ Error: No database is open")
				continue
			}

			companies, err := db.ListCompanies()
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
				continue
			}

			if len(companies) == 0 {
				fmt.Println("\nNo companies found in database")
			} else {
				fmt.Printf("\n✓ Found %d company/companies:\n", len(companies))
				for i, comp := range companies {
					if comp == db.currentCompany {
						fmt.Printf("  %d. %s (ACTIVE)\n", i+1, comp)
					} else {
						fmt.Printf("  %d. %s\n", i+1, comp)
					}
				}
			}

		case "9":
			// Object Designer
			if db == nil {
				fmt.Println("✗ Error: No database is open")
				continue
			}

			if db.currentCompany == "" {
				fmt.Println("✗ Error: You must enter a company first")
				continue
			}

			object_designer.Run(db, scanner)

		case "10":
			// Exit application
			if db != nil {
				fmt.Println("\nClosing database before exit...")
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
