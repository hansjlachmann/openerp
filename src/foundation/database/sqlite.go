package database

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// Database represents a SQLite database connection with session state
type Database struct {
	conn           *sql.DB
	path           string
	currentCompany string // Per-connection state for thread safety
}

// CreateDatabase creates a new SQLite database file
func CreateDatabase(path string) (*Database, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create Company table
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS "Company" (
			name TEXT PRIMARY KEY,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create Company table: %w", err)
	}

	db := &Database{
		conn:           conn,
		path:           path,
		currentCompany: "",
	}

	return db, nil
}

// OpenDatabase opens an existing SQLite database file
func OpenDatabase(path string) (*Database, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Verify Company table exists
	var tableName string
	err = conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='Company'").Scan(&tableName)
	if err == sql.ErrNoRows {
		conn.Close()
		return nil, fmt.Errorf("not a valid OpenERP database: Company table not found")
	}
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to verify database: %w", err)
	}

	db := &Database{
		conn:           conn,
		path:           path,
		currentCompany: "",
	}

	return db, nil
}

// CloseDatabase closes the database connection
func (db *Database) CloseDatabase() error {
	if db.conn == nil {
		return fmt.Errorf("database already closed")
	}

	// Exit company session if active
	if db.currentCompany != "" {
		db.currentCompany = ""
	}

	err := db.conn.Close()
	if err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	db.conn = nil
	return nil
}

// GetConnection returns the underlying database connection
func (db *Database) GetConnection() *sql.DB {
	return db.conn
}

// GetCurrentCompany returns the current company context
func (db *Database) GetCurrentCompany() string {
	return db.currentCompany
}

// SetCurrentCompany sets the current company context (internal use)
func (db *Database) SetCurrentCompany(company string) {
	db.currentCompany = company
}

// GetDatabasePath returns the database file path
func (db *Database) GetDatabasePath() string {
	return db.path
}

// GetFullTableName returns the company-prefixed table name
// Format: CompanyName$TableName (Business Central style)
func (db *Database) GetFullTableName(tableName string) (string, error) {
	if db.currentCompany == "" {
		return "", fmt.Errorf("no company context set - use EnterCompany() first")
	}
	return fmt.Sprintf("%s$%s", db.currentCompany, tableName), nil
}

// TableExists checks if a table exists in the database
func (db *Database) TableExists(tableName string) (bool, error) {
	if db.conn == nil {
		return false, fmt.Errorf("database not open")
	}

	var name string
	err := db.conn.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name=?
	`, tableName).Scan(&name)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// CreateTable creates a new table with the given schema
func (db *Database) CreateTable(tableName, schema string) error {
	if db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return fmt.Errorf("no company context set - use EnterCompany() first")
	}

	fullTableName, err := db.GetFullTableName(tableName)
	if err != nil {
		return err
	}

	// Check if table already exists
	exists, err := db.TableExists(fullTableName)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("table %s already exists", fullTableName)
	}

	// Create table
	createSQL := fmt.Sprintf(`CREATE TABLE "%s" (%s)`, fullTableName, schema)
	_, err = db.conn.Exec(createSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

// DropTable drops a table from the database
func (db *Database) DropTable(tableName string) error {
	if db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return fmt.Errorf("no company context set - use EnterCompany() first")
	}

	fullTableName, err := db.GetFullTableName(tableName)
	if err != nil {
		return err
	}

	_, err = db.conn.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS "%s"`, fullTableName))
	return err
}

// ValidateCompanyName checks if a company name is valid
func ValidateCompanyName(name string) error {
	if name == "" {
		return fmt.Errorf("company name cannot be empty")
	}

	// Validate company name (no special characters that could break table names)
	if strings.ContainsAny(name, " $\"'`\\;-") {
		return fmt.Errorf("company name contains invalid characters (spaces, $, quotes, backslashes, semicolons, or hyphens not allowed)")
	}

	return nil
}
