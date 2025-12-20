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
			is_primary_key INTEGER DEFAULT 0,
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
			is_primary_key INTEGER DEFAULT 0,
			field_order INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(company, table_name, field_name)
		)
	`)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create FieldDefinition table: %w", err)
	}

	// Add new columns if they don't exist (for existing databases)
	conn.Exec(`ALTER TABLE FieldDefinition ADD COLUMN is_primary_key INTEGER DEFAULT 0`)
	conn.Exec(`ALTER TABLE FieldDefinition ADD COLUMN field_order INTEGER DEFAULT 0`)

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

// CreateTable registers a new table (NAV-style: table created when first field added)
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

	// Check if table already has fields defined (table exists)
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM FieldDefinition WHERE table_name = ?
	`, tableName).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("table '%s' already exists", tableName)
	}

	// NAV-style: Table is just registered here, actual SQL table created when first field added
	// No action needed - table will be created by AddField
	return nil
}

// ListTables returns all tables that have field definitions
func (db *Database) ListTables() ([]string, error) {
	if db.conn == nil {
		return nil, fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return nil, fmt.Errorf("no company context set - use EnterCompany() first")
	}

	// Get distinct table names from FieldDefinition
	rows, err := db.conn.Query(`
		SELECT DISTINCT table_name
		FROM FieldDefinition
		WHERE company = ?
		ORDER BY table_name
	`, db.currentCompany)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to read table name: %w", err)
		}
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

// AddField adds a new field to a table (NAV-style: creates table when first non-PK field added)
func (db *Database) AddField(tableName, fieldName, fieldType string, isPrimaryKey bool) error {
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

	// Check if field already exists in metadata (check any company)
	var fieldExists int
	err := db.conn.QueryRow(`
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

	// Get current field count to determine field order
	var fieldCount int
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM FieldDefinition
		WHERE table_name = ?
		LIMIT 1
	`, tableName).Scan(&fieldCount)
	if err != nil {
		return fmt.Errorf("failed to count fields: %w", err)
	}

	fieldOrder := fieldCount + 1

	// Check if SQL table exists for current company
	fullTableName := fmt.Sprintf("%s$%s", db.currentCompany, tableName)
	var tableExists int
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name = ?
	`, fullTableName).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	// Get all companies to sync changes
	companies, err := db.ListCompanies()
	if err != nil {
		return fmt.Errorf("failed to get companies: %w", err)
	}

	// NAV-Style Logic:
	// 1. If SQL table doesn't exist yet:
	//    - If this is a PK field: Store metadata only, don't create table yet
	//    - If this is NOT a PK field: Create table with all PK fields, then add this field
	// 2. If SQL table exists:
	//    - Can't add more PK fields (SQLite limitation)
	//    - Use ALTER TABLE to add regular field

	if tableExists == 0 {
		// Table doesn't exist yet
		if isPrimaryKey {
			// Just store metadata for PK field, don't create table yet
			// Table will be created when first non-PK field is added or when data is accessed
			for _, company := range companies {
				_, err = db.conn.Exec(`
					INSERT INTO FieldDefinition (company, table_name, field_name, field_type, is_primary_key, field_order)
					VALUES (?, ?, ?, ?, 1, ?)
				`, company, tableName, fieldName, fieldType, fieldOrder)
				if err != nil {
					return fmt.Errorf("failed to store field metadata for company %s: %w", company, err)
				}
			}
			return nil
		} else {
			// First non-PK field: Create table with all PK fields from metadata
			// Get all PK fields defined so far
			rows, err := db.conn.Query(`
				SELECT field_name, field_type
				FROM FieldDefinition
				WHERE table_name = ? AND is_primary_key = 1
				ORDER BY field_order
			`, tableName)
			if err != nil {
				return fmt.Errorf("failed to get PK fields: %w", err)
			}

			var pkFields []struct {
				name     string
				sqlType  string
			}

			for rows.Next() {
				var name, fType string
				if err := rows.Scan(&name, &fType); err != nil {
					rows.Close()
					return fmt.Errorf("failed to read PK field: %w", err)
				}
				pkFields = append(pkFields, struct {
					name    string
					sqlType string
				}{name, validTypes[fType]})
			}
			rows.Close()

			if len(pkFields) == 0 {
				return fmt.Errorf("table must have at least one primary key field before adding non-primary key fields")
			}

			// Create table for each company with PK fields
			for _, company := range companies {
				companyTableName := fmt.Sprintf("%s$%s", company, tableName)

				// Build CREATE TABLE statement with user-defined primary keys (NAV-style)
				var pkFieldDefs []string
				var pkNames []string
				for _, pk := range pkFields {
					pkFieldDefs = append(pkFieldDefs, fmt.Sprintf(`"%s" %s NOT NULL`, pk.name, pk.sqlType))
					pkNames = append(pkNames, fmt.Sprintf(`"%s"`, pk.name))
				}

				createSQL := fmt.Sprintf(`
					CREATE TABLE "%s" (
						%s,
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (%s)
					)
				`, companyTableName, strings.Join(pkFieldDefs, ", "), strings.Join(pkNames, ", "))

				_, err = db.conn.Exec(createSQL)
				if err != nil {
					return fmt.Errorf("failed to create table for company %s: %w", company, err)
				}

				// Now add this non-PK field
				alterSQL := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" %s`, companyTableName, fieldName, sqlType)
				_, err = db.conn.Exec(alterSQL)
				if err != nil {
					return fmt.Errorf("failed to add field to table for company %s: %w", company, err)
				}

				// Store field metadata
				_, err = db.conn.Exec(`
					INSERT INTO FieldDefinition (company, table_name, field_name, field_type, is_primary_key, field_order)
					VALUES (?, ?, ?, ?, 0, ?)
				`, company, tableName, fieldName, fieldType, fieldOrder)
				if err != nil {
					return fmt.Errorf("failed to store field metadata for company %s: %w", company, err)
				}
			}

			return nil
		}
	} else {
		// Table already exists
		if isPrimaryKey {
			return fmt.Errorf("cannot add primary key field after table is created - add all primary key fields first")
		}

		// Add field to existing table in each company
		for _, company := range companies {
			companyTableName := fmt.Sprintf("%s$%s", company, tableName)

			// Add field using ALTER TABLE
			alterSQL := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" %s`, companyTableName, fieldName, sqlType)
			_, err = db.conn.Exec(alterSQL)
			if err != nil {
				return fmt.Errorf("failed to add field to table for company %s: %w", company, err)
			}

			// Store field metadata
			_, err = db.conn.Exec(`
				INSERT INTO FieldDefinition (company, table_name, field_name, field_type, is_primary_key, field_order)
				VALUES (?, ?, ?, ?, 0, ?)
			`, company, tableName, fieldName, fieldType, fieldOrder)
			if err != nil {
				return fmt.Errorf("failed to store field metadata for company %s: %w", company, err)
			}
		}

		return nil
	}
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
		SELECT field_name, field_type, is_primary_key, field_order
		FROM FieldDefinition
		WHERE company = ? AND table_name = ?
		ORDER BY field_order, id
	`, db.currentCompany, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to list fields: %w", err)
	}
	defer rows.Close()

	var fields []types.FieldInfo
	for rows.Next() {
		var field types.FieldInfo
		var isPK int
		if err := rows.Scan(&field.Name, &field.Type, &isPK, &field.FieldOrder); err != nil {
			return nil, fmt.Errorf("failed to read field: %w", err)
		}
		field.IsPrimaryKey = isPK == 1
		fields = append(fields, field)
	}

	return fields, nil
}

// buildPrimaryKeyWhere builds a WHERE clause for primary key fields
// Returns the WHERE clause string and the values to bind
func (db *Database) buildPrimaryKeyWhere(tableName string, primaryKey map[string]interface{}) (string, []interface{}, error) {
	// Get primary key fields for this table
	fields, err := db.ListFields(tableName)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get fields: %w", err)
	}

	var pkFields []string
	for _, field := range fields {
		if field.IsPrimaryKey {
			pkFields = append(pkFields, field.Name)
		}
	}

	if len(pkFields) == 0 {
		return "", nil, fmt.Errorf("table has no primary key fields defined")
	}

	// Build WHERE clause
	var whereClauses []string
	var values []interface{}

	for _, pkField := range pkFields {
		value, ok := primaryKey[pkField]
		if !ok {
			return "", nil, fmt.Errorf("primary key field '%s' not provided", pkField)
		}
		whereClauses = append(whereClauses, fmt.Sprintf(`"%s" = ?`, pkField))
		values = append(values, value)
	}

	whereClause := strings.Join(whereClauses, " AND ")
	return whereClause, values, nil
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

	// Get primary key fields and validate they are all provided
	fields, err := db.ListFields(tableName)
	if err != nil {
		return 0, fmt.Errorf("failed to get fields: %w", err)
	}

	for _, field := range fields {
		if field.IsPrimaryKey {
			value, ok := record[field.Name]
			if !ok || value == nil || value == "" {
				return 0, fmt.Errorf("primary key field '%s' is required and cannot be empty", field.Name)
			}
		}
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

	_, err = db.conn.Exec(query, values...)
	if err != nil {
		return 0, fmt.Errorf("failed to insert record: %w", err)
	}

	// Return 0 for NAV-style (no auto-increment ID)
	return 0, nil
}

// GetRecord retrieves a single record by primary key
func (db *Database) GetRecord(tableName string, primaryKey map[string]interface{}) (map[string]interface{}, error) {
	if db.conn == nil {
		return nil, fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return nil, fmt.Errorf("no company context set - use EnterCompany() first")
	}

	if tableName == "" {
		return nil, fmt.Errorf("table name cannot be empty")
	}

	// Build WHERE clause from primary key
	whereClause, whereValues, err := db.buildPrimaryKeyWhere(tableName, primaryKey)
	if err != nil {
		return nil, err
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

	// Build SELECT query with primary key WHERE clause
	query := fmt.Sprintf(`SELECT * FROM "%s" WHERE %s`, fullTableName, whereClause)
	row := db.conn.QueryRow(query, whereValues...)

	// Scan into map
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	if err := row.Scan(valuePtrs...); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("record not found")
		}
		return nil, fmt.Errorf("failed to scan record: %w", err)
	}

	record := make(map[string]interface{})
	for i, col := range columns {
		record[col] = values[i]
	}

	return record, nil
}

// UpdateRecord updates a record by primary key
func (db *Database) UpdateRecord(tableName string, primaryKey map[string]interface{}, updates map[string]interface{}) error {
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

	// Build WHERE clause from primary key
	whereClause, whereValues, err := db.buildPrimaryKeyWhere(tableName, primaryKey)
	if err != nil {
		return err
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

	// Append WHERE values
	values = append(values, whereValues...)

	query := fmt.Sprintf(
		`UPDATE "%s" SET %s WHERE %s`,
		fullTableName,
		strings.Join(setClauses, ", "),
		whereClause,
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
		return fmt.Errorf("record not found")
	}

	return nil
}

// DeleteRecord deletes a record by primary key
func (db *Database) DeleteRecord(tableName string, primaryKey map[string]interface{}) error {
	if db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return fmt.Errorf("no company context set - use EnterCompany() first")
	}

	if tableName == "" {
		return fmt.Errorf("table name cannot be empty")
	}

	// Build WHERE clause from primary key
	whereClause, whereValues, err := db.buildPrimaryKeyWhere(tableName, primaryKey)
	if err != nil {
		return err
	}

	// Get full table name
	fullTableName := fmt.Sprintf("%s$%s", db.currentCompany, tableName)

	query := fmt.Sprintf(`DELETE FROM "%s" WHERE %s`, fullTableName, whereClause)
	result, err := db.conn.Exec(query, whereValues...)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("record not found")
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
