package object_designer

import (
	"bufio"
	"fmt"
	"strings"
)

// Database interface defines the methods we need from the main Database type
type Database interface {
	GetCurrentCompany() string
	CreateTable(tableName string) error
	ListTables() ([]string, error)
	DeleteTable(tableName string) error
}

// Run starts the Object Designer interactive menu
func Run(db Database, scanner *bufio.Scanner) {
	for {
		// Show Object Designer header
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Printf("OBJECT DESIGNER - Company: %s\n", db.GetCurrentCompany())
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("1. Create Table")
		fmt.Println("2. List Tables")
		fmt.Println("3. Delete Table")
		fmt.Println("4. Back to Main Menu")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Print("\nSelect option (1-4): ")

		if !scanner.Scan() {
			return
		}

		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			createTable(db, scanner)
		case "2":
			listTables(db)
		case "3":
			deleteTable(db, scanner)
		case "4":
			// Back to Main Menu
			fmt.Println("\n✓ Returning to Main Menu")
			return
		default:
			fmt.Printf("✗ Invalid option: %s\n", choice)
		}
	}
}

func createTable(db Database, scanner *bufio.Scanner) {
	fmt.Print("\nEnter table name (e.g., customers): ")
	if !scanner.Scan() {
		return
	}
	tableName := strings.TrimSpace(scanner.Text())

	if tableName == "" {
		fmt.Println("✗ Error: Table name cannot be empty")
		return
	}

	err := db.CreateTable(tableName)
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
	} else {
		fullName := fmt.Sprintf("%s$%s", db.GetCurrentCompany(), tableName)
		fmt.Printf("✓ Table '%s' created successfully\n", fullName)
	}
}

func listTables(db Database) {
	tables, err := db.ListTables()
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
		return
	}

	if len(tables) == 0 {
		fmt.Println("\nNo tables found for this company")
	} else {
		fmt.Printf("\n✓ Found %d table(s) for company '%s':\n", len(tables), db.GetCurrentCompany())
		for i, table := range tables {
			fullName := fmt.Sprintf("%s$%s", db.GetCurrentCompany(), table)
			fmt.Printf("  %d. %s (full name: %s)\n", i+1, table, fullName)
		}
	}
}

func deleteTable(db Database, scanner *bufio.Scanner) {
	tables, err := db.ListTables()
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
		return
	}

	if len(tables) == 0 {
		fmt.Println("✗ No tables to delete")
		return
	}

	fmt.Println("\nAvailable tables:")
	for i, table := range tables {
		fmt.Printf("  %d. %s\n", i+1, table)
	}

	fmt.Print("\nEnter table name to delete: ")
	if !scanner.Scan() {
		return
	}
	tableName := strings.TrimSpace(scanner.Text())

	if tableName == "" {
		fmt.Println("✗ Error: Table name cannot be empty")
		return
	}

	// Confirm deletion
	fullName := fmt.Sprintf("%s$%s", db.GetCurrentCompany(), tableName)
	fmt.Printf("\n⚠️  WARNING: This will permanently delete table '%s'!\n", fullName)
	fmt.Print("Are you sure? (yes/no): ")
	if !scanner.Scan() {
		return
	}
	confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))

	if confirm != "yes" {
		fmt.Println("✓ Deletion cancelled")
		return
	}

	err = db.DeleteTable(tableName)
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
	} else {
		fmt.Printf("✓ Table '%s' deleted successfully\n", fullName)
	}
}
