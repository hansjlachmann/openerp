package database

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

// Database represents a PostgreSQL database connection with session state
type Database struct {
	conn           *sql.DB
	connString     string
	currentCompany string // Per-connection state for thread safety
}

// CreateDatabase creates a new PostgreSQL database connection
func CreateDatabase(host, port, user, password, dbname string) (*Database, error) {
	connString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	conn, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create Company table
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS "Company" (
			name VARCHAR(50) PRIMARY KEY,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create Company table: %w", err)
	}

	// Create FieldDefinition table for metadata
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS "FieldDefinition" (
			id SERIAL PRIMARY KEY,
			company VARCHAR(50) NOT NULL,
			table_name VARCHAR(100) NOT NULL,
			field_name VARCHAR(100) NOT NULL,
			field_type VARCHAR(50) NOT NULL,
			is_primary_key BOOLEAN DEFAULT FALSE,
			field_order INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(company, table_name, field_name)
		)
	`)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create FieldDefinition table: %w", err)
	}

	db := &Database{
		conn:           conn,
		connString:     connString,
		currentCompany: "",
	}

	return db, nil
}

// OpenDatabase opens an existing PostgreSQL database connection
func OpenDatabase(host, port, user, password, dbname string) (*Database, error) {
	connString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	conn, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Verify Company table exists
	var tableName string
	err = conn.QueryRow(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name = 'Company'
	`).Scan(&tableName)

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
		connString:     connString,
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

	var exists bool
	err := db.conn.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)
	`, tableName).Scan(&exists)

	return exists, err
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
