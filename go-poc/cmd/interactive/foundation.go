package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// Database represents a database connection with session state
type Database struct {
	conn           *sql.DB
	path           string
	currentCompany string // Per-connection state for thread safety
}

// CreateDatabase creates a new persistent database file
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
		CREATE TABLE IF NOT EXISTS Company (
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

// OpenDatabase opens an existing persistent database
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

// CreateCompany creates a new company in the database
func (db *Database) CreateCompany(name string) error {
	if db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if name == "" {
		return fmt.Errorf("company name cannot be empty")
	}

	// Validate company name (no special characters that could break table names)
	if strings.ContainsAny(name, " $\"'`\\") {
		return fmt.Errorf("company name contains invalid characters")
	}

	_, err := db.conn.Exec("INSERT INTO Company (name) VALUES (?)", name)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("company '%s' already exists", name)
		}
		return fmt.Errorf("failed to create company: %w", err)
	}

	return nil
}

// EnterCompany sets the current company context (session-based)
func (db *Database) EnterCompany(name string) error {
	if db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if name == "" {
		return fmt.Errorf("company name cannot be empty")
	}

	// Check if already in a company
	if db.currentCompany != "" {
		return fmt.Errorf("already in company '%s' - exit first before entering another", db.currentCompany)
	}

	// Verify company exists
	var companyName string
	err := db.conn.QueryRow("SELECT name FROM Company WHERE name = ?", name).Scan(&companyName)
	if err == sql.ErrNoRows {
		return fmt.Errorf("company '%s' does not exist", name)
	}
	if err != nil {
		return fmt.Errorf("failed to verify company: %w", err)
	}

	// Set current company (per-connection state)
	db.currentCompany = name
	return nil
}

// ExitCompany clears the current company context
func (db *Database) ExitCompany() error {
	if db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return fmt.Errorf("no company session active")
	}

	db.currentCompany = ""
	return nil
}

// DeleteCompany deletes a company and all its Company$Tables
func (db *Database) DeleteCompany(name string) error {
	if db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if name == "" {
		return fmt.Errorf("company name cannot be empty")
	}

	// Verify company exists
	var companyName string
	err := db.conn.QueryRow("SELECT name FROM Company WHERE name = ?", name).Scan(&companyName)
	if err == sql.ErrNoRows {
		return fmt.Errorf("company '%s' does not exist", name)
	}
	if err != nil {
		return fmt.Errorf("failed to verify company: %w", err)
	}

	// If this is the current company, exit it first
	if db.currentCompany == name {
		db.currentCompany = ""
	}

	// Find all tables belonging to this company (Company$TableName pattern)
	rows, err := db.conn.Query(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name LIKE ?
	`, name+"$%")
	if err != nil {
		return fmt.Errorf("failed to find company tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to read table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	// Delete all company tables
	for _, tableName := range tables {
		_, err := db.conn.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS "%s"`, tableName))
		if err != nil {
			return fmt.Errorf("failed to drop table %s: %w", tableName, err)
		}
	}

	// Delete company record
	_, err = db.conn.Exec("DELETE FROM Company WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("failed to delete company: %w", err)
	}

	return nil
}

// ListCompanies returns all companies in the database
func (db *Database) ListCompanies() ([]string, error) {
	if db.conn == nil {
		return nil, fmt.Errorf("database not open")
	}

	rows, err := db.conn.Query("SELECT name FROM Company ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("failed to list companies: %w", err)
	}
	defer rows.Close()

	var companies []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to read company name: %w", err)
		}
		companies = append(companies, name)
	}

	return companies, nil
}

// GetCurrentCompany returns the current company context
func (db *Database) GetCurrentCompany() string {
	return db.currentCompany
}

// GetFullTableName returns the company-prefixed table name
func (db *Database) GetFullTableName(tableName string) (string, error) {
	if db.currentCompany == "" {
		return "", fmt.Errorf("no company context set - use EnterCompany() first")
	}
	return fmt.Sprintf("%s$%s", db.currentCompany, tableName), nil
}
