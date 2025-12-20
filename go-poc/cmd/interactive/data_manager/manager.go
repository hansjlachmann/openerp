package data_manager

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/hansjlachmann/openerp-go/types"
)

// Database interface defines the methods we need from the main Database type
type Database interface {
	GetCurrentCompany() string
	ListTables() ([]string, error)
	ListFields(tableName string) ([]types.FieldInfo, error)
	InsertRecord(tableName string, record map[string]interface{}) (int64, error)
	GetRecord(tableName string, id int64) (map[string]interface{}, error)
	UpdateRecord(tableName string, id int64, updates map[string]interface{}) error
	DeleteRecord(tableName string, id int64) error
	ListRecords(tableName string) ([]map[string]interface{}, error)
}

// Run starts the Data Manager interactive menu
func Run(db Database, scanner *bufio.Scanner) {
	// Select table first
	tables, err := db.ListTables()
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
		return
	}

	if len(tables) == 0 {
		fmt.Println("✗ No tables available. Create a table first in Object Designer.")
		return
	}

	fmt.Println("\nAvailable tables:")
	for i, table := range tables {
		fmt.Printf("  %d. %s\n", i+1, table)
	}

	fmt.Print("\nEnter table name: ")
	if !scanner.Scan() {
		return
	}
	tableName := strings.TrimSpace(scanner.Text())

	if tableName == "" {
		fmt.Println("✗ Error: Table name cannot be empty")
		return
	}

	// Enter table data management menu
	manageTableData(db, scanner, tableName)
}

func manageTableData(db Database, scanner *bufio.Scanner, tableName string) {
	for {
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Printf("DATA MANAGER - %s$%s\n", db.GetCurrentCompany(), tableName)
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("1. View all records")
		fmt.Println("2. Add new record")
		fmt.Println("3. View single record")
		fmt.Println("4. Update record")
		fmt.Println("5. Delete record")
		fmt.Println("6. Back to Object Designer")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Print("\nSelect option (1-6): ")

		if !scanner.Scan() {
			return
		}

		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			viewAllRecords(db, tableName)
		case "2":
			addRecord(db, scanner, tableName)
		case "3":
			viewRecord(db, scanner, tableName)
		case "4":
			updateRecord(db, scanner, tableName)
		case "5":
			deleteRecord(db, scanner, tableName)
		case "6":
			return
		default:
			fmt.Printf("✗ Invalid option: %s\n", choice)
		}
	}
}

func viewAllRecords(db Database, tableName string) {
	records, err := db.ListRecords(tableName)
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
		return
	}

	if len(records) == 0 {
		fmt.Println("\nNo records found in this table")
		return
	}

	fmt.Printf("\n✓ Found %d record(s):\n\n", len(records))

	// Get fields to show in order
	fields, err := db.ListFields(tableName)
	if err != nil {
		fmt.Printf("✗ Error getting fields: %v\n", err)
		return
	}

	// Print header
	fmt.Printf("%-6s %-20s ", "ID", "Created At")
	for _, field := range fields {
		fmt.Printf("%-20s ", field.Name)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))

	// Print records
	for _, record := range records {
		fmt.Printf("%-6v %-20v ", record["id"], formatValue(record["created_at"]))
		for _, field := range fields {
			fmt.Printf("%-20v ", formatValue(record[field.Name]))
		}
		fmt.Println()
	}
}

func addRecord(db Database, scanner *bufio.Scanner, tableName string) {
	fields, err := db.ListFields(tableName)
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
		return
	}

	if len(fields) == 0 {
		fmt.Println("\n✗ No fields defined for this table. Add fields first in Object Designer.")
		return
	}

	fmt.Println("\nEnter values for new record:")
	record := make(map[string]interface{})

	for _, field := range fields {
		fmt.Printf("%s (%s): ", field.Name, field.Type)
		if !scanner.Scan() {
			return
		}
		value := strings.TrimSpace(scanner.Text())

		// Convert value based on field type
		converted, err := convertValue(value, field.Type)
		if err != nil {
			fmt.Printf("✗ Invalid value for %s: %v\n", field.Type, err)
			return
		}
		record[field.Name] = converted
	}

	id, err := db.InsertRecord(tableName, record)
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
		return
	}

	fmt.Printf("✓ Record created successfully with ID: %d\n", id)
}

func viewRecord(db Database, scanner *bufio.Scanner, tableName string) {
	fmt.Print("\nEnter record ID: ")
	if !scanner.Scan() {
		return
	}
	idStr := strings.TrimSpace(scanner.Text())

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		fmt.Println("✗ Invalid ID")
		return
	}

	record, err := db.GetRecord(tableName, id)
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
		return
	}

	fmt.Printf("\n✓ Record ID %d:\n", id)
	fmt.Println(strings.Repeat("-", 40))
	for key, value := range record {
		fmt.Printf("%-20s: %v\n", key, formatValue(value))
	}
}

func updateRecord(db Database, scanner *bufio.Scanner, tableName string) {
	fmt.Print("\nEnter record ID to update: ")
	if !scanner.Scan() {
		return
	}
	idStr := strings.TrimSpace(scanner.Text())

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		fmt.Println("✗ Invalid ID")
		return
	}

	// Show current record
	record, err := db.GetRecord(tableName, id)
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
		return
	}

	fmt.Printf("\nCurrent values for record %d:\n", id)
	fields, err := db.ListFields(tableName)
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
		return
	}

	for _, field := range fields {
		fmt.Printf("%-20s: %v\n", field.Name, formatValue(record[field.Name]))
	}

	fmt.Println("\nEnter new values (press Enter to skip a field):")
	updates := make(map[string]interface{})

	for _, field := range fields {
		fmt.Printf("%s (%s) [current: %v]: ", field.Name, field.Type, formatValue(record[field.Name]))
		if !scanner.Scan() {
			return
		}
		value := strings.TrimSpace(scanner.Text())

		if value == "" {
			continue // Skip this field
		}

		// Convert value based on field type
		converted, err := convertValue(value, field.Type)
		if err != nil {
			fmt.Printf("✗ Invalid value for %s: %v\n", field.Type, err)
			return
		}
		updates[field.Name] = converted
	}

	if len(updates) == 0 {
		fmt.Println("✓ No changes made")
		return
	}

	err = db.UpdateRecord(tableName, id, updates)
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
		return
	}

	fmt.Printf("✓ Record %d updated successfully\n", id)
}

func deleteRecord(db Database, scanner *bufio.Scanner, tableName string) {
	fmt.Print("\nEnter record ID to delete: ")
	if !scanner.Scan() {
		return
	}
	idStr := strings.TrimSpace(scanner.Text())

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		fmt.Println("✗ Invalid ID")
		return
	}

	// Show record first
	record, err := db.GetRecord(tableName, id)
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
		return
	}

	fmt.Printf("\nRecord to delete (ID %d):\n", id)
	for key, value := range record {
		fmt.Printf("%-20s: %v\n", key, formatValue(value))
	}

	fmt.Print("\n⚠️  Are you sure you want to delete this record? (yes/no): ")
	if !scanner.Scan() {
		return
	}
	confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))

	if confirm != "yes" {
		fmt.Println("✓ Deletion cancelled")
		return
	}

	err = db.DeleteRecord(tableName, id)
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
		return
	}

	fmt.Printf("✓ Record %d deleted successfully\n", id)
}

func convertValue(value string, fieldType string) (interface{}, error) {
	if value == "" {
		return nil, nil
	}

	switch fieldType {
	case "Text":
		return value, nil
	case "Boolean":
		lower := strings.ToLower(value)
		if lower == "true" || lower == "1" || lower == "yes" {
			return 1, nil
		} else if lower == "false" || lower == "0" || lower == "no" {
			return 0, nil
		}
		return nil, fmt.Errorf("invalid boolean value (use true/false, 1/0, yes/no)")
	case "Integer":
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}
		return i, nil
	case "Decimal":
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}
		return f, nil
	case "Date":
		// Accept date in any format for now
		return value, nil
	default:
		return value, nil
	}
}

func formatValue(value interface{}) string {
	if value == nil {
		return "<null>"
	}

	// Convert byte slices to strings (SQLite sometimes returns these)
	if b, ok := value.([]byte); ok {
		return string(b)
	}

	return fmt.Sprintf("%v", value)
}
