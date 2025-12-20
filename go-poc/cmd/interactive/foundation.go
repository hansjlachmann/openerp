package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/hansjlachmann/openerp-go/types"
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

	// Create FieldDefinition table for metadata
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS FieldDefinition (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			company TEXT NOT NULL,
			table_name TEXT NOT NULL,
			field_name TEXT NOT NULL,
			field_type TEXT NOT NULL,
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

	// Create FieldDefinition table if it doesn't exist (for backward compatibility)
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS FieldDefinition (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			company TEXT NOT NULL,
			table_name TEXT NOT NULL,
			field_name TEXT NOT NULL,
			field_type TEXT NOT NULL,
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

// CreateTable creates a new table for ALL companies
func (db *Database) CreateTable(tableName string) error {
	if db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return fmt.Errorf("no company context set - use EnterCompany() first")
	}

	if tableName == "" {
		return fmt.Errorf("table name cannot be empty")
	}

	// Validate table name (no special characters)
	if strings.ContainsAny(tableName, " $\"'`\\") {
		return fmt.Errorf("table name contains invalid characters")
	}

	// Get all companies to create table for each
	companies, err := db.ListCompanies()
	if err != nil {
		return fmt.Errorf("failed to get companies: %w", err)
	}

	// Create table for each company
	for _, company := range companies {
		fullTableName := fmt.Sprintf("%s$%s", company, tableName)

		// Create basic table with id and created_at
		_, err := db.conn.Exec(fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS "%s" (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`, fullTableName))
		if err != nil {
			return fmt.Errorf("failed to create table for company %s: %w", company, err)
		}
	}

	return nil
}

// ListTables returns all tables for the current company
func (db *Database) ListTables() ([]string, error) {
	if db.conn == nil {
		return nil, fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return nil, fmt.Errorf("no company context set - use EnterCompany() first")
	}

	// Find all tables for current company (Company$% pattern)
	rows, err := db.conn.Query(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name LIKE ?
		ORDER BY name
	`, db.currentCompany+"$%")
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var fullTableName string
		if err := rows.Scan(&fullTableName); err != nil {
			return nil, fmt.Errorf("failed to read table name: %w", err)
		}
		// Strip company prefix to show just the table name
		tableName := strings.TrimPrefix(fullTableName, db.currentCompany+"$")
		tables = append(tables, tableName)
	}

	return tables, nil
}

// DeleteTable deletes a table from ALL companies
func (db *Database) DeleteTable(tableName string) error {
	if db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return fmt.Errorf("no company context set - use EnterCompany() first")
	}

	if tableName == "" {
		return fmt.Errorf("table name cannot be empty")
	}

	// Check if table exists for current company
	fullTableName := fmt.Sprintf("%s$%s", db.currentCompany, tableName)
	var exists int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name = ?
	`, fullTableName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if exists == 0 {
		return fmt.Errorf("table '%s' does not exist", tableName)
	}

	// Get all companies to delete table from each
	companies, err := db.ListCompanies()
	if err != nil {
		return fmt.Errorf("failed to get companies: %w", err)
	}

	// Delete table from each company
	for _, company := range companies {
		companyTableName := fmt.Sprintf("%s$%s", company, tableName)

		// Delete the table
		_, err = db.conn.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS "%s"`, companyTableName))
		if err != nil {
			return fmt.Errorf("failed to delete table for company %s: %w", company, err)
		}
	}

	// Delete all field metadata for this table (all companies)
	_, err = db.conn.Exec(`DELETE FROM FieldDefinition WHERE table_name = ?`, tableName)
	if err != nil {
		return fmt.Errorf("failed to delete field metadata: %w", err)
	}

	return nil
}

// AddField adds a new field to a table
func (db *Database) AddField(tableName, fieldName, fieldType string) error {
	if db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return fmt.Errorf("no company context set - use EnterCompany() first")
	}

	if tableName == "" {
		return fmt.Errorf("table name cannot be empty")
	}

	if fieldName == "" {
		return fmt.Errorf("field name cannot be empty")
	}

	// Validate field name (no special characters)
	if strings.ContainsAny(fieldName, " $\"'`\\") {
		return fmt.Errorf("field name contains invalid characters")
	}

	// Validate field type
	validTypes := map[string]string{
		"Text":    "TEXT",
		"Boolean": "INTEGER", // SQLite uses 0/1 for boolean
		"Date":    "TEXT",    // SQLite stores dates as TEXT in ISO8601 format
		"Decimal": "REAL",
		"Integer": "INTEGER",
	}

	sqlType, ok := validTypes[fieldType]
	if !ok {
		return fmt.Errorf("invalid field type '%s'. Valid types: Text, Boolean, Date, Decimal, Integer", fieldType)
	}

	// Check if table exists for current company
	fullTableName := fmt.Sprintf("%s$%s", db.currentCompany, tableName)
	var exists int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name = ?
	`, fullTableName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if exists == 0 {
		return fmt.Errorf("table '%s' does not exist", tableName)
	}

	// Check if field already exists in metadata (check any company)
	var fieldExists int
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM FieldDefinition
		WHERE table_name = ? AND field_name = ?
		LIMIT 1
	`, tableName, fieldName).Scan(&fieldExists)
	if err != nil {
		return fmt.Errorf("failed to check field existence: %w", err)
	}

	if fieldExists > 0 {
		return fmt.Errorf("field '%s' already exists in table '%s'", fieldName, tableName)
	}

	// Get all companies to add field to each
	companies, err := db.ListCompanies()
	if err != nil {
		return fmt.Errorf("failed to get companies: %w", err)
	}

	// Add field to table in each company
	for _, company := range companies {
		companyTableName := fmt.Sprintf("%s$%s", company, tableName)

		// Add field to actual table using ALTER TABLE
		alterSQL := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" %s`, companyTableName, fieldName, sqlType)
		_, err = db.conn.Exec(alterSQL)
		if err != nil {
			return fmt.Errorf("failed to add field to table for company %s: %w", company, err)
		}

		// Store field metadata for each company
		_, err = db.conn.Exec(`
			INSERT INTO FieldDefinition (company, table_name, field_name, field_type)
			VALUES (?, ?, ?, ?)
		`, company, tableName, fieldName, fieldType)
		if err != nil {
			return fmt.Errorf("failed to store field metadata for company %s: %w", company, err)
		}
	}

	return nil
}

// ListFields returns all fields for a table
func (db *Database) ListFields(tableName string) ([]types.FieldInfo, error) {
	if db.conn == nil {
		return nil, fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return nil, fmt.Errorf("no company context set - use EnterCompany() first")
	}

	if tableName == "" {
		return nil, fmt.Errorf("table name cannot be empty")
	}

	// Get fields from metadata
	rows, err := db.conn.Query(`
		SELECT field_name, field_type
		FROM FieldDefinition
		WHERE company = ? AND table_name = ?
		ORDER BY id
	`, db.currentCompany, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to list fields: %w", err)
	}
	defer rows.Close()

	var fields []types.FieldInfo
	for rows.Next() {
		var field types.FieldInfo
		if err := rows.Scan(&field.Name, &field.Type); err != nil {
			return nil, fmt.Errorf("failed to read field: %w", err)
		}
		fields = append(fields, field)
	}

	return fields, nil
}

// InsertRecord inserts a new record into a table
func (db *Database) InsertRecord(tableName string, record map[string]interface{}) (int64, error) {
	if db.conn == nil {
		return 0, fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return 0, fmt.Errorf("no company context set - use EnterCompany() first")
	}

	if tableName == "" {
		return 0, fmt.Errorf("table name cannot be empty")
	}

	// Get full table name
	fullTableName := fmt.Sprintf("%s$%s", db.currentCompany, tableName)

	// Build INSERT query
	var columns []string
	var placeholders []string
	var values []interface{}

	for key, value := range record {
		columns = append(columns, fmt.Sprintf(`"%s"`, key))
		placeholders = append(placeholders, "?")
		values = append(values, value)
	}

	query := fmt.Sprintf(
		`INSERT INTO "%s" (%s) VALUES (%s)`,
		fullTableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	result, err := db.conn.Exec(query, values...)
	if err != nil {
		return 0, fmt.Errorf("failed to insert record: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get insert id: %w", err)
	}

	return id, nil
}

// GetRecord retrieves a single record by ID
func (db *Database) GetRecord(tableName string, id int64) (map[string]interface{}, error) {
	if db.conn == nil {
		return nil, fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return nil, fmt.Errorf("no company context set - use EnterCompany() first")
	}

	if tableName == "" {
		return nil, fmt.Errorf("table name cannot be empty")
	}

	// Get full table name
	fullTableName := fmt.Sprintf("%s$%s", db.currentCompany, tableName)

	// Get all column names using PRAGMA
	columnsQuery := fmt.Sprintf(`PRAGMA table_info("%s")`, fullTableName)
	rows, err := db.conn.Query(columnsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get table info: %w", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return nil, fmt.Errorf("failed to read column info: %w", err)
		}
		columns = append(columns, name)
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("table has no columns")
	}

	// Build SELECT query
	query := fmt.Sprintf(`SELECT * FROM "%s" WHERE id = ?`, fullTableName)
	row := db.conn.QueryRow(query, id)

	// Scan into map
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	if err := row.Scan(valuePtrs...); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("record with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to scan record: %w", err)
	}

	record := make(map[string]interface{})
	for i, col := range columns {
		record[col] = values[i]
	}

	return record, nil
}

// UpdateRecord updates a record by ID
func (db *Database) UpdateRecord(tableName string, id int64, updates map[string]interface{}) error {
	if db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return fmt.Errorf("no company context set - use EnterCompany() first")
	}

	if tableName == "" {
		return fmt.Errorf("table name cannot be empty")
	}

	if len(updates) == 0 {
		return fmt.Errorf("no updates provided")
	}

	// Get full table name
	fullTableName := fmt.Sprintf("%s$%s", db.currentCompany, tableName)

	// Build UPDATE query
	var setClauses []string
	var values []interface{}

	for key, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf(`"%s" = ?`, key))
		values = append(values, value)
	}
	values = append(values, id)

	query := fmt.Sprintf(
		`UPDATE "%s" SET %s WHERE id = ?`,
		fullTableName,
		strings.Join(setClauses, ", "),
	)

	result, err := db.conn.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("record with id %d not found", id)
	}

	return nil
}

// DeleteRecord deletes a record by ID
func (db *Database) DeleteRecord(tableName string, id int64) error {
	if db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return fmt.Errorf("no company context set - use EnterCompany() first")
	}

	if tableName == "" {
		return fmt.Errorf("table name cannot be empty")
	}

	// Get full table name
	fullTableName := fmt.Sprintf("%s$%s", db.currentCompany, tableName)

	query := fmt.Sprintf(`DELETE FROM "%s" WHERE id = ?`, fullTableName)
	result, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("record with id %d not found", id)
	}

	return nil
}

// ListRecords retrieves all records from a table
func (db *Database) ListRecords(tableName string) ([]map[string]interface{}, error) {
	if db.conn == nil {
		return nil, fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return nil, fmt.Errorf("no company context set - use EnterCompany() first")
	}

	if tableName == "" {
		return nil, fmt.Errorf("table name cannot be empty")
	}

	// Get full table name
	fullTableName := fmt.Sprintf("%s$%s", db.currentCompany, tableName)

	// Get all column names using PRAGMA
	columnsQuery := fmt.Sprintf(`PRAGMA table_info("%s")`, fullTableName)
	columnRows, err := db.conn.Query(columnsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get table info: %w", err)
	}
	defer columnRows.Close()

	var columns []string
	for columnRows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		if err := columnRows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return nil, fmt.Errorf("failed to read column info: %w", err)
		}
		columns = append(columns, name)
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("table has no columns")
	}

	// Query all records
	query := fmt.Sprintf(`SELECT * FROM "%s" ORDER BY id`, fullTableName)
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}
	defer rows.Close()

	var records []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}

		record := make(map[string]interface{})
		for i, col := range columns {
			record[col] = values[i]
		}
		records = append(records, record)
	}

	return records, nil
}
