package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// Database represents the core database connection
type Database struct {
	conn *sql.DB
}

// NewDatabase creates a new database connection
func NewDatabase(dbPath string) (*Database, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable foreign keys
	_, err = conn.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return nil, err
	}

	return &Database{conn: conn}, nil
}

// Company represents a company in the multi-tenant system
type Company struct {
	Name string
}

// CreateCompany creates a new company
func (db *Database) CreateCompany(name string) error {
	_, err := db.conn.Exec("INSERT INTO Company (Name) VALUES (?)", name)
	return err
}

// GetFullTableName returns the company-prefixed table name
func GetFullTableName(tableName, companyName string) string {
	if companyName != "" {
		return fmt.Sprintf("%s$%s", companyName, tableName)
	}
	return tableName
}

// Record represents a database record (NAV-style)
type Record map[string]interface{}

// CRUDManager handles CRUD operations
type CRUDManager struct {
	db *Database
	// TODO: Add Python trigger engine here
}

// NewCRUDManager creates a new CRUD manager
func NewCRUDManager(db *Database) *CRUDManager {
	return &CRUDManager{db: db}
}

// Insert inserts a new record
func (crud *CRUDManager) Insert(tableName string, record Record) (int64, error) {
	// TODO: Execute Python ON_INSERT trigger here
	// For now, just do the database insert

	// Build INSERT statement dynamically
	var columns []string
	var placeholders []string
	var values []interface{}

	for key, value := range record {
		columns = append(columns, key)
		placeholders = append(placeholders, "?")
		values = append(values, value)
	}

	query := fmt.Sprintf(
		`INSERT INTO "%s" (%s) VALUES (%s)`,
		tableName,
		joinStrings(columns, ", "),
		joinStrings(placeholders, ", "),
	)

	result, err := crud.db.conn.Exec(query, values...)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// Get retrieves a record by ID (NAV-style)
func (crud *CRUDManager) Get(tableName string, id int64) (Record, error) {
	query := fmt.Sprintf(`SELECT * FROM "%s" WHERE id = ?`, tableName)

	rows, err := crud.db.conn.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("record not found")
	}

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// Create a slice of interface{} to hold the values
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Scan the row
	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	// Build the record map
	record := make(Record)
	for i, col := range columns {
		record[col] = values[i]
	}

	return record, nil
}

// Update updates a record (NAV-style)
func (crud *CRUDManager) Update(tableName string, id int64, updates Record) error {
	// TODO: Execute Python ON_UPDATE trigger here

	var setClauses []string
	var values []interface{}

	for key, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", key))
		values = append(values, value)
	}
	values = append(values, id)

	query := fmt.Sprintf(
		`UPDATE "%s" SET %s WHERE id = ?`,
		tableName,
		joinStrings(setClauses, ", "),
	)

	_, err := crud.db.conn.Exec(query, values...)
	return err
}

// Delete deletes a record (NAV-style)
func (crud *CRUDManager) Delete(tableName string, id int64) error {
	// TODO: Execute Python ON_DELETE trigger here

	query := fmt.Sprintf(`DELETE FROM "%s" WHERE id = ?`, tableName)
	_, err := crud.db.conn.Exec(query, id)
	return err
}

// FindSet returns all records (NAV-style)
func (crud *CRUDManager) FindSet(tableName string) ([]Record, error) {
	query := fmt.Sprintf(`SELECT * FROM "%s"`, tableName)

	rows, err := crud.db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		record := make(Record)
		for i, col := range columns {
			record[col] = values[i]
		}
		records = append(records, record)
	}

	return records, nil
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// Example usage (NAV-style)
func main() {
	fmt.Println("=== OpenERP Go - Proof of Concept ===\n")

	// Create database
	db, err := NewDatabase(":memory:")
	if err != nil {
		log.Fatal(err)
	}

	// Create Company table
	_, err = db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS Company (
			Name TEXT PRIMARY KEY
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Create company
	err = db.CreateCompany("ACME")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Created company: ACME")

	// Create customers table for ACME
	tableName := GetFullTableName("customers", "ACME")
	_, err = db.conn.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS "%s" (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT,
			phone TEXT,
			balance REAL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`, tableName))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Created table: %s\n\n", tableName)

	// CRUD operations
	crud := NewCRUDManager(db)

	// INSERT (NAV-style)
	fmt.Println("1. INSERT - Creating customer")
	fmt.Println(strings.Repeat("-", 50))
	customer := Record{
		"name":    "John Doe",
		"email":   "john@example.com",
		"phone":   "+1-555-0100",
		"balance": 1000.0,
	}

	id, err := crud.Insert(tableName, customer)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Customer created with ID: %d\n", id)

	// GET (NAV-style: customer.GET(1))
	fmt.Println("\n2. GET - Retrieving customer")
	fmt.Println(strings.Repeat("-", 50))
	customer, err = crud.Get(tableName, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Found customer: %v\n", customer["name"])
	prettyPrint(customer)

	// MODIFY (NAV-style: customer.MODIFY)
	fmt.Println("\n3. MODIFY - Updating customer")
	fmt.Println(strings.Repeat("-", 50))
	updates := Record{
		"balance": 2500.0,
		"email":   "john.doe@example.com",
	}
	err = crud.Update(tableName, id, updates)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Customer updated")

	// Verify update
	customer, _ = crud.Get(tableName, id)
	prettyPrint(customer)

	// FINDSET (NAV-style: customer.FINDSET)
	fmt.Println("\n4. FINDSET - Getting all customers")
	fmt.Println(strings.Repeat("-", 50))

	// Insert more customers
	crud.Insert(tableName, Record{
		"name": "Jane Smith", "email": "jane@example.com", "balance": 3000.0,
	})
	crud.Insert(tableName, Record{
		"name": "Bob Wilson", "email": "bob@example.com", "balance": 1500.0,
	})

	customers, err := crud.FindSet(tableName)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Found %d customers:\n", len(customers))
	for _, c := range customers {
		fmt.Printf("  - %v: %v ($%.2f)\n", c["name"], c["email"], c["balance"])
	}

	// DELETE (NAV-style: customer.DELETE)
	fmt.Println("\n5. DELETE - Removing customer")
	fmt.Println(strings.Repeat("-", 50))
	err = crud.Delete(tableName, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Customer deleted")

	customers, _ = crud.FindSet(tableName)
	fmt.Printf("  Remaining customers: %d\n", len(customers))

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Proof of Concept Complete!")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add Python trigger execution")
	fmt.Println("  2. Add metadata management")
	fmt.Println("  3. Add HTTP API layer")
	fmt.Println("  4. Performance benchmarking")
}

func prettyPrint(record Record) {
	data, _ := json.MarshalIndent(record, "  ", "  ")
	fmt.Printf("  %s\n", data)
}
