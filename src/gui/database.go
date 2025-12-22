package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/hansjlachmann/openerp/src/gui/types"
)

// Database wraps the foundation database functionality for GUI use
type Database struct {
	conn           *sql.DB
	currentCompany string
	dbPath         string
}

// OpenDatabase opens or creates a database file
func (db *Database) OpenDatabase(dbPath string) error {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	db.conn = conn
	db.dbPath = dbPath

	// Create Company table if it doesn't exist
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS Company (
			name TEXT PRIMARY KEY,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create Company table: %w", err)
	}

	// Create FieldDefinition table
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
		return fmt.Errorf("failed to create FieldDefinition table: %w", err)
	}

	// Add new columns if they don't exist (backward compatibility)
	conn.Exec(`ALTER TABLE FieldDefinition ADD COLUMN is_primary_key INTEGER DEFAULT 0`)
	conn.Exec(`ALTER TABLE FieldDefinition ADD COLUMN field_order INTEGER DEFAULT 0`)

	return nil
}

// CreateCompany creates a new company
func (db *Database) CreateCompany(companyName string) error {
	if db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if companyName == "" {
		return fmt.Errorf("company name cannot be empty")
	}

	_, err := db.conn.Exec(`INSERT INTO Company (name) VALUES (?)`, companyName)
	if err != nil {
		return fmt.Errorf("failed to create company: %w", err)
	}

	return nil
}

// ListCompanies returns all companies
func (db *Database) ListCompanies() ([]string, error) {
	if db.conn == nil {
		return nil, fmt.Errorf("database not open")
	}

	rows, err := db.conn.Query(`SELECT name FROM Company ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to list companies: %w", err)
	}
	defer rows.Close()

	var companies []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to read company: %w", err)
		}
		companies = append(companies, name)
	}

	return companies, nil
}

// EnterCompany sets the current company context
func (db *Database) EnterCompany(companyName string) error {
	if db.conn == nil {
		return fmt.Errorf("database not open")
	}

	// Verify company exists
	var exists int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM Company WHERE name = ?`, companyName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to verify company: %w", err)
	}

	if exists == 0 {
		return fmt.Errorf("company '%s' does not exist", companyName)
	}

	db.currentCompany = companyName
	return nil
}

// CreateTable registers a new table
func (db *Database) CreateTable(tableName string) error {
	if db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return fmt.Errorf("no company context set")
	}

	if tableName == "" {
		return fmt.Errorf("table name cannot be empty")
	}

	// Validate table name
	if strings.ContainsAny(tableName, " $\"'`\\") {
		return fmt.Errorf("table name contains invalid characters")
	}

	// Check if table already exists
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM FieldDefinition WHERE table_name = ?`, tableName).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("table '%s' already exists", tableName)
	}

	// Get all companies
	companies, err := db.ListCompanies()
	if err != nil {
		return fmt.Errorf("failed to get companies: %w", err)
	}

	// Insert marker record for all companies
	for _, company := range companies {
		_, err = db.conn.Exec(`
			INSERT INTO FieldDefinition (company, table_name, field_name, field_type, is_primary_key, field_order)
			VALUES (?, ?, '__table_registered__', 'marker', 0, 0)
		`, company, tableName)
		if err != nil {
			return fmt.Errorf("failed to register table for company %s: %w", company, err)
		}
	}

	return nil
}

// ListTables returns all tables for current company
func (db *Database) ListTables() ([]string, error) {
	if db.conn == nil {
		return nil, fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return nil, fmt.Errorf("no company context set")
	}

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

// DeleteTable deletes a table from all companies
func (db *Database) DeleteTable(tableName string) error {
	if db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return fmt.Errorf("no company context set")
	}

	companies, err := db.ListCompanies()
	if err != nil {
		return fmt.Errorf("failed to get companies: %w", err)
	}

	// Delete table for each company
	for _, company := range companies {
		companyTableName := fmt.Sprintf("%s$%s", company, tableName)
		_, err = db.conn.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS "%s"`, companyTableName))
		if err != nil {
			return fmt.Errorf("failed to delete table for company %s: %w", company, err)
		}
	}

	// Delete all field metadata
	_, err = db.conn.Exec(`DELETE FROM FieldDefinition WHERE table_name = ?`, tableName)
	if err != nil {
		return fmt.Errorf("failed to delete field metadata: %w", err)
	}

	return nil
}

// AddField adds a field to a table
func (db *Database) AddField(tableName, fieldName, fieldType string, isPrimaryKey bool) error {
	if db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if db.currentCompany == "" {
		return fmt.Errorf("no company context set")
	}

	// Validate inputs
	if tableName == "" || fieldName == "" {
		return fmt.Errorf("table name and field name cannot be empty")
	}

	if strings.ContainsAny(fieldName, " $\"'`\\") {
		return fmt.Errorf("field name contains invalid characters")
	}

	// Validate field type
	validTypes := map[string]string{
		"Text":    "TEXT",
		"Boolean": "INTEGER",
		"Date":    "TEXT",
		"Decimal": "REAL",
		"Integer": "INTEGER",
	}

	sqlType, ok := validTypes[fieldType]
	if !ok {
		return fmt.Errorf("invalid field type '%s'", fieldType)
	}

	// Check if field already exists
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

	// Get field count for ordering
	var fieldCount int
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM FieldDefinition
		WHERE table_name = ? AND field_name != '__table_registered__'
		LIMIT 1
	`, tableName).Scan(&fieldCount)
	if err != nil {
		return fmt.Errorf("failed to count fields: %w", err)
	}

	fieldOrder := fieldCount + 1

	// Check if SQL table exists
	fullTableName := fmt.Sprintf("%s$%s", db.currentCompany, tableName)
	var tableExists int
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name = ?
	`, fullTableName).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	companies, err := db.ListCompanies()
	if err != nil {
		return fmt.Errorf("failed to get companies: %w", err)
	}

	if tableExists == 0 {
		// Table doesn't exist - store metadata only for PK fields
		if isPrimaryKey {
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
			// First non-PK field - create table with all PK fields
			return db.ensureTableExists(tableName)
		}
	} else {
		// Table exists
		if isPrimaryKey {
			return fmt.Errorf("cannot add primary key field after table is created")
		}

		// Add field to existing table
		for _, company := range companies {
			companyTableName := fmt.Sprintf("%s$%s", company, tableName)
			alterSQL := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" %s`, companyTableName, fieldName, sqlType)
			_, err = db.conn.Exec(alterSQL)
			if err != nil {
				return fmt.Errorf("failed to add field to table for company %s: %w", company, err)
			}

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
		return nil, fmt.Errorf("no company context set")
	}

	rows, err := db.conn.Query(`
		SELECT field_name, field_type, is_primary_key, field_order
		FROM FieldDefinition
		WHERE company = ? AND table_name = ? AND field_name != '__table_registered__'
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

// ensureTableExists creates the SQL table from metadata if it doesn't exist
func (db *Database) ensureTableExists(tableName string) error {
	fullTableName := fmt.Sprintf("%s$%s", db.currentCompany, tableName)
	var tableExists int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name = ?
	`, fullTableName).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if tableExists > 0 {
		return nil // Already exists
	}

	// Get fields from metadata
	fields, err := db.ListFields(tableName)
	if err != nil {
		return fmt.Errorf("failed to get fields: %w", err)
	}

	if len(fields) == 0 {
		return fmt.Errorf("cannot create table: no fields defined")
	}

	// Separate PK and regular fields
	var pkFields []types.FieldInfo
	var regularFields []types.FieldInfo
	for _, field := range fields {
		if field.IsPrimaryKey {
			pkFields = append(pkFields, field)
		} else {
			regularFields = append(regularFields, field)
		}
	}

	if len(pkFields) == 0 {
		return fmt.Errorf("cannot create table: no primary key fields defined")
	}

	validTypes := map[string]string{
		"Text":    "TEXT",
		"Boolean": "INTEGER",
		"Date":    "TEXT",
		"Decimal": "REAL",
		"Integer": "INTEGER",
	}

	companies, err := db.ListCompanies()
	if err != nil {
		return fmt.Errorf("failed to get companies: %w", err)
	}

	// Create table for each company
	for _, company := range companies {
		companyTableName := fmt.Sprintf("%s$%s", company, tableName)

		var allFieldDefs []string
		var pkNames []string

		for _, field := range pkFields {
			sqlType := validTypes[field.Type]
			allFieldDefs = append(allFieldDefs, fmt.Sprintf(`"%s" %s NOT NULL`, field.Name, sqlType))
			pkNames = append(pkNames, fmt.Sprintf(`"%s"`, field.Name))
		}

		for _, field := range regularFields {
			sqlType := validTypes[field.Type]
			allFieldDefs = append(allFieldDefs, fmt.Sprintf(`"%s" %s`, field.Name, sqlType))
		}

		allFieldDefs = append(allFieldDefs, "created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP")

		createSQL := fmt.Sprintf(`
			CREATE TABLE "%s" (
				%s,
				PRIMARY KEY (%s)
			)
		`, companyTableName, strings.Join(allFieldDefs, ", "), strings.Join(pkNames, ", "))

		_, err = db.conn.Exec(createSQL)
		if err != nil {
			return fmt.Errorf("failed to create table for company %s: %w", company, err)
		}
	}

	return nil
}

// CRUD Operations

func (db *Database) InsertRecord(tableName string, record map[string]interface{}) (int64, error) {
	if err := db.ensureTableExists(tableName); err != nil {
		return 0, err
	}

	fields, err := db.ListFields(tableName)
	if err != nil {
		return 0, err
	}

	// Validate PK fields
	for _, field := range fields {
		if field.IsPrimaryKey {
			value, ok := record[field.Name]
			if !ok || value == nil || value == "" {
				return 0, fmt.Errorf("primary key field '%s' is required", field.Name)
			}
		}
	}

	fullTableName := fmt.Sprintf("%s$%s", db.currentCompany, tableName)

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

	return 0, nil
}

func (db *Database) GetRecord(tableName string, primaryKey map[string]interface{}) (map[string]interface{}, error) {
	if err := db.ensureTableExists(tableName); err != nil {
		return nil, err
	}

	whereClause, whereValues, err := db.buildPrimaryKeyWhere(tableName, primaryKey)
	if err != nil {
		return nil, err
	}

	fullTableName := fmt.Sprintf("%s$%s", db.currentCompany, tableName)

	columnsQuery := fmt.Sprintf(`PRAGMA table_info("%s")`, fullTableName)
	rows, err := db.conn.Query(columnsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}

	query := fmt.Sprintf(`SELECT * FROM "%s" WHERE %s`, fullTableName, whereClause)
	row := db.conn.QueryRow(query, whereValues...)

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	if err := row.Scan(valuePtrs...); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("record not found")
		}
		return nil, err
	}

	record := make(map[string]interface{})
	for i, col := range columns {
		record[col] = values[i]
	}

	return record, nil
}

func (db *Database) UpdateRecord(tableName string, primaryKey map[string]interface{}, updates map[string]interface{}) error {
	if err := db.ensureTableExists(tableName); err != nil {
		return err
	}

	whereClause, whereValues, err := db.buildPrimaryKeyWhere(tableName, primaryKey)
	if err != nil {
		return err
	}

	fullTableName := fmt.Sprintf("%s$%s", db.currentCompany, tableName)

	var setClauses []string
	var values []interface{}

	for key, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf(`"%s" = ?`, key))
		values = append(values, value)
	}

	values = append(values, whereValues...)

	query := fmt.Sprintf(
		`UPDATE "%s" SET %s WHERE %s`,
		fullTableName,
		strings.Join(setClauses, ", "),
		whereClause,
	)

	result, err := db.conn.Exec(query, values...)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("record not found")
	}

	return nil
}

func (db *Database) DeleteRecord(tableName string, primaryKey map[string]interface{}) error {
	if err := db.ensureTableExists(tableName); err != nil {
		return err
	}

	whereClause, whereValues, err := db.buildPrimaryKeyWhere(tableName, primaryKey)
	if err != nil {
		return err
	}

	fullTableName := fmt.Sprintf("%s$%s", db.currentCompany, tableName)

	query := fmt.Sprintf(`DELETE FROM "%s" WHERE %s`, fullTableName, whereClause)
	result, err := db.conn.Exec(query, whereValues...)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("record not found")
	}

	return nil
}

func (db *Database) ListRecords(tableName string) ([]map[string]interface{}, error) {
	if err := db.ensureTableExists(tableName); err != nil {
		return nil, err
	}

	fullTableName := fmt.Sprintf("%s$%s", db.currentCompany, tableName)

	columnsQuery := fmt.Sprintf(`PRAGMA table_info("%s")`, fullTableName)
	columnRows, err := db.conn.Query(columnsQuery)
	if err != nil {
		return nil, err
	}
	defer columnRows.Close()

	var columns []string
	for columnRows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		if err := columnRows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}

	query := fmt.Sprintf(`SELECT * FROM "%s"`, fullTableName)
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
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
			return nil, err
		}

		record := make(map[string]interface{})
		for i, col := range columns {
			record[col] = values[i]
		}
		records = append(records, record)
	}

	return records, nil
}

func (db *Database) buildPrimaryKeyWhere(tableName string, primaryKey map[string]interface{}) (string, []interface{}, error) {
	fields, err := db.ListFields(tableName)
	if err != nil {
		return "", nil, err
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
