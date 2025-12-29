package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/hansjlachmann/openerp/src/business-logic/codeunits"
	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/company"
	"github.com/hansjlachmann/openerp/src/foundation/config"
	"github.com/hansjlachmann/openerp/src/foundation/database"
	"github.com/hansjlachmann/openerp/src/foundation/objects"
	"github.com/hansjlachmann/openerp/src/foundation/session"
	"github.com/hansjlachmann/openerp/src/foundation/user"
)

// Note: tables import is needed for registering tables (PaymentTerms, Customer, etc.)

func main() {
	fmt.Println("=== OpenERP - Business Central Style ERP ===\n")

	var db *database.Database
	var companyMgr *company.Manager
	var registry *objects.ObjectRegistry
	var sess *session.Session
	scanner := bufio.NewScanner(os.Stdin)

	// Initialize object registry
	registry = objects.NewObjectRegistry()

	// Register PaymentTerms table (Table ID: 3)
	if err := registry.RegisterTable(tables.PaymentTermsTableID, &tables.PaymentTerms{}); err != nil {
		fmt.Printf("Warning: Failed to register PaymentTerms: %v\n", err)
	}

	// Register Customer table (Table ID: 18)
	if err := registry.RegisterTable(tables.CustomerTableID, &tables.Customer{}); err != nil {
		fmt.Printf("Warning: Failed to register Customer: %v\n", err)
	}

	// Register Customer Ledger Entry table (Table ID: 21)
	if err := registry.RegisterTable(tables.CustomerLedgerEntryTableID, &tables.CustomerLedgerEntry{}); err != nil {
		fmt.Printf("Warning: Failed to register Customer Ledger Entry: %v\n", err)
	}

	// Try to auto-connect to last used database and company
	if lastConn, err := config.LoadLastConnection(); err == nil {
		fmt.Printf("Auto-connecting to last session...\n")
		fmt.Printf("  Database: %s\n", lastConn.DatabasePath)
		fmt.Printf("  Company: %s\n", lastConn.Company)

		// Open database
		if openedDB, err := database.OpenDatabase(lastConn.DatabasePath); err == nil {
			db = openedDB
			companyMgr = company.NewManager(db, registry)

			// Require login before entering company
			authenticatedUser, err := login(db, scanner)
			if err != nil {
				fmt.Printf("✗ Auto-connect cancelled: Login failed\n")
			} else {
				// Enter company after successful login
				if err := companyMgr.EnterCompany(lastConn.Company); err == nil {
					sess = session.NewSession(db, lastConn.Company, scanner)
					sess.SetUser(authenticatedUser.Username, authenticatedUser.FullName, authenticatedUser.Language)
					session.SetCurrent(sess) // Set global session
					fmt.Printf("✓ Auto-connected successfully!\n")
				} else {
					fmt.Printf("✗ Failed to enter company: %v\n", err)
				}
			}
		} else {
			fmt.Printf("✗ Failed to open database: %v\n", err)
		}
	}

	for {
		// Show current status
		if db != nil && companyMgr != nil {
			currentCompany := companyMgr.GetCurrentCompany()
			fmt.Printf("\n[Database: %s", db.GetDatabasePath())
			if currentCompany != "" {
				fmt.Printf(" | Company: %s", currentCompany)
			}
			// Show logged-in user if in session
			if sess != nil && sess.GetUserID() != "" {
				fmt.Printf(" | User: %s (%s)", sess.GetUserName(), sess.GetUserID())
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
		fmt.Println("10. Codeunits")
		fmt.Println("11. Users")
		fmt.Println("12. Exit application")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Print("\nSelect option (1-12): ")

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
				sess = nil
				session.ClearCurrent() // Clear global session

				// Clear last connection
				config.ClearLastConnection()
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

			// Require login before entering company
			authenticatedUser, err := login(db, scanner)
			if err != nil {
				fmt.Printf("✗ Cannot enter company: Login failed\n")
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
				// Create session with user information
				sess = session.NewSession(db, name, scanner)
				sess.SetUser(authenticatedUser.Username, authenticatedUser.FullName, authenticatedUser.Language)
				session.SetCurrent(sess) // Set global session
				fmt.Printf("✓ Entered company '%s'\n", name)

				// Save last connection
				if err := config.SaveLastConnection(db.GetDatabasePath(), name); err != nil {
					fmt.Printf("Warning: Failed to save connection info: %v\n", err)
				}
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
				// Clear session
				sess = nil
				session.ClearCurrent() // Clear global session
				fmt.Println("✓ Exited company session")

				// Clear last connection
				config.ClearLastConnection()
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
			// Codeunits
			if sess == nil {
				fmt.Println("✗ Error: No company selected. Please enter a company first (option 5)")
				continue
			}

			// Show Codeunit submenu
			fmt.Println("\n" + strings.Repeat("=", 60))
			fmt.Println("CODEUNITS MENU")
			fmt.Println(strings.Repeat("=", 60))
			fmt.Printf("Current Company: %s\n", sess.GetCompany())
			fmt.Println(strings.Repeat("-", 60))
			fmt.Println("1. Codeunit 50000 - Payment Terms Management")
			fmt.Println("2. Codeunit 50001 - Insert 10 Payment Terms Records")
			fmt.Println("3. Codeunit 50002 - Customer Management")
			fmt.Println("4. Codeunit 50003 - Customer Search Demo (SetRange/FindFirst/FindLast)")
			fmt.Println("5. Codeunit 50004 - Customer Phase 3 Demo (IsEmpty/ModifyAll/DeleteAll)")
			fmt.Println("6. Codeunit 50005 - Helper Functions Demo (IncStr/CopyStr)")
			fmt.Println("7. Codeunit 50006 - OnValidate Triggers Demo (Field Validation)")
			fmt.Println("8. Codeunit 50007 - Bidirectional Navigation Demo (FindSet/FindSetBuffered)")
			fmt.Println("9. Codeunit 50008 - New Data Types Demo (Decimal/Date/DateTime/BLOB)")
			fmt.Println("10. Codeunit 50009 - Customer Ledger Entry Demo (Insert Test Data)")
			fmt.Println("11. Codeunit 50010 - FlowField Demo (Sum/Count Calculations)")
			fmt.Println("12. Codeunit 50011 - Create Large Dataset (100K entries for CUST-001)")
			fmt.Println("13. Codeunit 50012 - Calculate FlowFields for CUST-001")
			fmt.Println("14. Codeunit 50013 - Transaction Demo (Commit/Rollback)")
			fmt.Println("15. Codeunit 50014 - Translation Demo (Multilanguage Support)")
			fmt.Println("0. Back to main menu")
			fmt.Println(strings.Repeat("=", 60))
			fmt.Print("\nSelect codeunit: ")

			if !scanner.Scan() {
				continue
			}

			codeunitChoice := strings.TrimSpace(scanner.Text())

			switch codeunitChoice {
			case "1":
				// Codeunit 50000: Payment Terms Management
				codeunits.RunPaymentTermsMgt()

			case "2":
				// Codeunit 50001: Insert 10 Payment Terms Records
				codeunits.RunPaymentTermsInsert10()

			case "3":
				// Codeunit 50002: Customer Management
				codeunits.RunCustomerMgt()

			case "4":
				// Codeunit 50003: Customer Search Demo
				codeunits.RunCustomerSearchDemo()

			case "5":
				// Codeunit 50004: Customer Phase 3 Demo
				codeunits.RunCustomerPhase3Demo()

			case "6":
				// Codeunit 50005: Helper Functions Demo
				codeunits.RunHelpersDemo()

			case "7":
				// Codeunit 50006: OnValidate Triggers Demo
				codeunits.RunOnValidateDemo()

			case "8":
				// Codeunit 50007: Bidirectional Navigation Demo
				codeunits.RunBidirectionalNavDemo()

			case "9":
				// Codeunit 50008: New Data Types Demo
				codeunits.NewTypesDemo(sess.GetConnection(), sess.GetCompany())

			case "10":
				// Codeunit 50009: Customer Ledger Entry Demo
				codeunits.CustLedgerEntryDemo(sess.GetConnection(), sess.GetCompany())

			case "11":
				// Codeunit 50010: FlowField Demo
				codeunits.FlowFieldDemo(sess.GetConnection(), sess.GetCompany())

			case "12":
				// Codeunit 50011: Create Large Dataset (100K entries for CUST-001)
				codeunits.CreateLargeCustomerDataset(sess.GetConnection(), sess.GetCompany())

			case "13":
				// Codeunit 50012: Calculate FlowFields for CUST-001
				codeunits.CalcFieldsLargeCustomer(sess.GetConnection(), sess.GetCompany())

			case "14":
				// Codeunit 50013: Transaction Demo (Commit/Rollback)
				codeunits.TransactionDemo(sess.GetConnection(), sess.GetCompany())

			case "15":
				// Codeunit 50014: Translation Demo (Multilanguage Support)
				codeunits.TranslationDemo(sess.GetConnection(), sess.GetCompany())

			case "0":
				fmt.Println("✓ Returning to main menu")

			default:
				fmt.Printf("✗ Invalid codeunit option: %s\n", codeunitChoice)
			}

		case "11":
			// Users
			if db == nil {
				fmt.Println("✗ Error: No database connection")
				continue
			}

			// Show Users submenu
			fmt.Println("\n" + strings.Repeat("=", 60))
			fmt.Println("USERS MENU")
			fmt.Println(strings.Repeat("=", 60))
			fmt.Println("1. Create User")
			fmt.Println("2. Modify User")
			fmt.Println("3. Delete User")
			fmt.Println("4. List Users")
			fmt.Println("0. Back to main menu")
			fmt.Println(strings.Repeat("=", 60))
			fmt.Print("\nSelect option: ")

			if !scanner.Scan() {
				continue
			}

			userChoice := strings.TrimSpace(scanner.Text())

			switch userChoice {
			case "1":
				// Create User
				createUser(db, scanner)

			case "2":
				// Modify User
				modifyUser(db, scanner)

			case "3":
				// Delete User
				deleteUser(db, scanner)

			case "4":
				// List Users
				listUsers(db, scanner)

			case "0":
				fmt.Println("✓ Returning to main menu")

			default:
				fmt.Printf("✗ Invalid option: %s\n", userChoice)
			}

		case "12":
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

// ========================================
// User Management Functions
// ========================================

// createUser creates a new user
func createUser(db *database.Database, scanner *bufio.Scanner) {
	userMgr := user.NewManager(db)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("CREATE USER")
	fmt.Println(strings.Repeat("=", 60))

	// Get username
	fmt.Print("\nEnter username: ")
	if !scanner.Scan() {
		return
	}
	username := strings.TrimSpace(scanner.Text())

	if username == "" {
		fmt.Println("✗ Error: Username cannot be empty")
		fmt.Print("\nPress Enter to continue...")
		scanner.Scan()
		return
	}

	// Get password
	fmt.Print("Enter password: ")
	if !scanner.Scan() {
		return
	}
	password := strings.TrimSpace(scanner.Text())

	if password == "" {
		fmt.Println("✗ Error: Password cannot be empty")
		fmt.Print("\nPress Enter to continue...")
		scanner.Scan()
		return
	}

	// Get full name
	fmt.Print("Enter full name: ")
	if !scanner.Scan() {
		return
	}
	fullName := strings.TrimSpace(scanner.Text())

	// Get language
	fmt.Print("Enter language (default: en-US): ")
	if !scanner.Scan() {
		return
	}
	language := strings.TrimSpace(scanner.Text())
	if language == "" {
		language = "en-US"
	}

	// Create user (using plain password as hash for now - should use proper hashing in production)
	err := userMgr.CreateUser(username, password, fullName, language)
	if err != nil {
		fmt.Printf("\n✗ Error: %v\n", err)
	} else {
		fmt.Printf("\n✓ User '%s' created successfully\n", username)
	}

	fmt.Print("\nPress Enter to continue...")
	scanner.Scan()
}

// modifyUser modifies an existing user
func modifyUser(db *database.Database, scanner *bufio.Scanner) {
	userMgr := user.NewManager(db)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("MODIFY USER")
	fmt.Println(strings.Repeat("=", 60))

	// Show available users
	users, err := userMgr.ListUsers()
	if err != nil {
		fmt.Printf("\n✗ Error: %v\n", err)
		fmt.Print("\nPress Enter to continue...")
		scanner.Scan()
		return
	}

	if len(users) == 0 {
		fmt.Println("\n✗ No users found. Create one first.")
		fmt.Print("\nPress Enter to continue...")
		scanner.Scan()
		return
	}

	fmt.Println("\nAvailable users:")
	for i, u := range users {
		activeStr := "Active"
		if !u.Active {
			activeStr = "Inactive"
		}
		fmt.Printf("  %d. %s (%s) - %s [%s]\n", i+1, u.Username, u.FullName, u.Language, activeStr)
	}

	// Get username
	fmt.Print("\nEnter username to modify: ")
	if !scanner.Scan() {
		return
	}
	username := strings.TrimSpace(scanner.Text())

	// Get existing user
	existingUser, err := userMgr.GetUser(username)
	if err != nil {
		fmt.Printf("\n✗ Error: %v\n", err)
		fmt.Print("\nPress Enter to continue...")
		scanner.Scan()
		return
	}

	// Get new full name
	fmt.Printf("Enter new full name (current: %s, press Enter to keep): ", existingUser.FullName)
	if !scanner.Scan() {
		return
	}
	fullName := strings.TrimSpace(scanner.Text())
	if fullName == "" {
		fullName = existingUser.FullName
	}

	// Get new language
	fmt.Printf("Enter new language (current: %s, press Enter to keep): ", existingUser.Language)
	if !scanner.Scan() {
		return
	}
	language := strings.TrimSpace(scanner.Text())
	if language == "" {
		language = existingUser.Language
	}

	// Get active status
	fmt.Printf("Active? (y/n, current: %v, press Enter to keep): ", existingUser.Active)
	if !scanner.Scan() {
		return
	}
	activeInput := strings.ToLower(strings.TrimSpace(scanner.Text()))
	active := existingUser.Active
	if activeInput == "y" || activeInput == "yes" {
		active = true
	} else if activeInput == "n" || activeInput == "no" {
		active = false
	}

	// Update user
	err = userMgr.UpdateUser(username, fullName, language, active)
	if err != nil {
		fmt.Printf("\n✗ Error: %v\n", err)
	} else {
		fmt.Printf("\n✓ User '%s' updated successfully\n", username)
	}

	fmt.Print("\nPress Enter to continue...")
	scanner.Scan()
}

// deleteUser deletes a user
func deleteUser(db *database.Database, scanner *bufio.Scanner) {
	userMgr := user.NewManager(db)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("DELETE USER")
	fmt.Println(strings.Repeat("=", 60))

	// Show available users
	users, err := userMgr.ListUsers()
	if err != nil {
		fmt.Printf("\n✗ Error: %v\n", err)
		fmt.Print("\nPress Enter to continue...")
		scanner.Scan()
		return
	}

	if len(users) == 0 {
		fmt.Println("\n✗ No users to delete")
		fmt.Print("\nPress Enter to continue...")
		scanner.Scan()
		return
	}

	fmt.Println("\nAvailable users:")
	for i, u := range users {
		fmt.Printf("  %d. %s (%s)\n", i+1, u.Username, u.FullName)
	}

	// Get username
	fmt.Print("\nEnter username to delete: ")
	if !scanner.Scan() {
		return
	}
	username := strings.TrimSpace(scanner.Text())

	// Confirm deletion
	fmt.Printf("\n⚠️  WARNING: This will delete user '%s'!\n", username)
	fmt.Print("Are you sure? (yes/no): ")
	if !scanner.Scan() {
		return
	}
	confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))

	if confirm != "yes" {
		fmt.Println("✓ Deletion cancelled")
		fmt.Print("\nPress Enter to continue...")
		scanner.Scan()
		return
	}

	// Delete user
	err = userMgr.DeleteUser(username)
	if err != nil {
		fmt.Printf("\n✗ Error: %v\n", err)
	} else {
		fmt.Printf("\n✓ User '%s' deleted successfully\n", username)
	}

	fmt.Print("\nPress Enter to continue...")
	scanner.Scan()
}

// listUsers lists all users
func listUsers(db *database.Database, scanner *bufio.Scanner) {
	userMgr := user.NewManager(db)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("USER LIST")
	fmt.Println(strings.Repeat("=", 60))

	users, err := userMgr.ListUsers()
	if err != nil {
		fmt.Printf("\n✗ Error: %v\n", err)
		fmt.Print("\nPress Enter to continue...")
		scanner.Scan()
		return
	}

	if len(users) == 0 {
		fmt.Println("\nNo users found in database")
	} else {
		fmt.Printf("\n✓ Found %d user(s):\n\n", len(users))
		fmt.Printf("%-20s %-30s %-10s %-10s\n", "Username", "Full Name", "Language", "Status")
		fmt.Println(strings.Repeat("-", 70))
		for _, u := range users {
			activeStr := "Active"
			if !u.Active {
				activeStr = "Inactive"
			}
			fmt.Printf("%-20s %-30s %-10s %-10s\n", u.Username, u.FullName, u.Language, activeStr)
		}
	}

	fmt.Print("\nPress Enter to continue...")
	scanner.Scan()
}

// ========================================
// Login Function
// ========================================

// login prompts for username and password and validates credentials
func login(db *database.Database, scanner *bufio.Scanner) (*user.User, error) {
	userMgr := user.NewManager(db)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("LOGIN")
	fmt.Println(strings.Repeat("=", 60))

	maxAttempts := 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Get username
		fmt.Print("\nUsername: ")
		if !scanner.Scan() {
			return nil, fmt.Errorf("failed to read username")
		}
		username := strings.TrimSpace(scanner.Text())

		// Get password
		fmt.Print("Password: ")
		if !scanner.Scan() {
			return nil, fmt.Errorf("failed to read password")
		}
		password := strings.TrimSpace(scanner.Text())

		// Validate credentials
		authenticatedUser, err := userMgr.ValidateCredentials(username, password)
		if err == nil {
			fmt.Printf("\n✓ Welcome, %s!\n", authenticatedUser.FullName)
			return authenticatedUser, nil
		}

		// Login failed
		fmt.Printf("\n✗ Login failed: %v\n", err)
		if attempt < maxAttempts {
			fmt.Printf("Attempts remaining: %d\n", maxAttempts-attempt)
		}
	}

	return nil, fmt.Errorf("maximum login attempts exceeded")
}
