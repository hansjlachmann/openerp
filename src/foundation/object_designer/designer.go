package object_designer

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/hansjlachmann/openerp/src/foundation/data_manager"
	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// Database interface defines the methods we need from the main Database type
type Database interface {
	GetCurrentCompany() string
	CreateTable(tableName string) error
	ListTables() ([]string, error)
	DeleteTable(tableName string) error
	AddField(tableName, fieldName, fieldType string, isPrimaryKey bool) error
	ListFields(tableName string) ([]types.FieldInfo, error)
	InsertRecord(tableName string, record map[string]interface{}) (int64, error)
	GetRecord(tableName string, primaryKey map[string]interface{}) (map[string]interface{}, error)
	UpdateRecord(tableName string, primaryKey map[string]interface{}, updates map[string]interface{}) error
	DeleteRecord(tableName string, primaryKey map[string]interface{}) error
	ListRecords(tableName string) ([]map[string]interface{}, error)
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
		fmt.Println("4. Add Field to Table")
		fmt.Println("5. Manage Table Data")
		fmt.Println("6. Back to Main Menu")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Print("\nSelect option (1-6): ")

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
			addField(db, scanner)
		case "5":
			data_manager.Run(db, scanner)
		case "6":
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

func addField(db Database, scanner *bufio.Scanner) {
	// Get list of tables first
	tables, err := db.ListTables()
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
		return
	}

	if len(tables) == 0 {
		fmt.Println("✗ No tables available. Create a table first.")
		return
	}

	// Show available tables
	fmt.Println("\nAvailable tables:")
	for i, table := range tables {
		fmt.Printf("  %d. %s\n", i+1, table)
	}

	// Select table
	fmt.Print("\nEnter table name: ")
	if !scanner.Scan() {
		return
	}
	tableName := strings.TrimSpace(scanner.Text())

	if tableName == "" {
		fmt.Println("✗ Error: Table name cannot be empty")
		return
	}

	// Show existing fields
	fields, err := db.ListFields(tableName)
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
		return
	}

	if len(fields) > 0 {
		fmt.Printf("\nExisting fields in table '%s':\n", tableName)
		for i, field := range fields {
			fmt.Printf("  %d. %s (%s)\n", i+1, field.Name, field.Type)
		}
	} else {
		fmt.Printf("\nTable '%s' has no custom fields yet (only id and created_at)\n", tableName)
	}

	// Enter field name
	fmt.Print("\nEnter field name (e.g., name, email, price): ")
	if !scanner.Scan() {
		return
	}
	fieldName := strings.TrimSpace(scanner.Text())

	if fieldName == "" {
		fmt.Println("✗ Error: Field name cannot be empty")
		return
	}

	// Show field types
	fmt.Println("\nAvailable field types:")
	fmt.Println("  1. Text")
	fmt.Println("  2. Boolean")
	fmt.Println("  3. Date")
	fmt.Println("  4. Decimal")
	fmt.Println("  5. Integer")
	fmt.Print("\nSelect field type (1-5): ")

	if !scanner.Scan() {
		return
	}
	typeChoice := strings.TrimSpace(scanner.Text())

	fieldTypes := map[string]string{
		"1": "Text",
		"2": "Boolean",
		"3": "Date",
		"4": "Decimal",
		"5": "Integer",
	}

	fieldType, ok := fieldTypes[typeChoice]
	if !ok {
		fmt.Printf("✗ Error: Invalid field type selection: %s\n", typeChoice)
		return
	}

	// Ask if this is a primary key field
	fmt.Print("\nIs this a primary key field? (yes/no): ")
	if !scanner.Scan() {
		return
	}
	isPKInput := strings.ToLower(strings.TrimSpace(scanner.Text()))
	isPrimaryKey := isPKInput == "yes" || isPKInput == "y"

	// Add the field
	err = db.AddField(tableName, fieldName, fieldType, isPrimaryKey)
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
	} else {
		fullTableName := fmt.Sprintf("%s$%s", db.GetCurrentCompany(), tableName)
		pkStatus := ""
		if isPrimaryKey {
			pkStatus = " [PRIMARY KEY]"
		}
		fmt.Printf("✓ Field '%s' (%s)%s added to table '%s' successfully\n", fieldName, fieldType, pkStatus, fullTableName)
	}
}
