package database

import (
	"database/sql"
	"fmt"

	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// Repository provides generic CRUD operations with trigger support
type Repository struct {
	db *Database
}

// NewRepository creates a new repository instance
func NewRepository(db *Database) *Repository {
	return &Repository{db: db}
}

// Insert inserts a record and calls OnInsert trigger if implemented
func (r *Repository) Insert(tableName string, record interface{}) error {
	if r.db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if r.db.currentCompany == "" {
		return fmt.Errorf("no company context set")
	}

	// Call OnInsert trigger if implemented
	if trigger, ok := record.(types.BeforeInsert); ok {
		if err := trigger.OnInsert(); err != nil {
			return fmt.Errorf("OnInsert trigger failed: %w", err)
		}
	}

	// Call Validate if implemented
	if validator, ok := record.(types.Validator); ok {
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// TODO: Implement actual insert logic based on struct fields
	// This will require reflection to read struct tags and build INSERT statement

	return nil
}

// Update updates a record and calls OnModify trigger if implemented
func (r *Repository) Update(record interface{}) error {
	if r.db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if r.db.currentCompany == "" {
		return fmt.Errorf("no company context set")
	}

	// Call OnModify trigger if implemented
	if trigger, ok := record.(types.BeforeModify); ok {
		if err := trigger.OnModify(); err != nil {
			return fmt.Errorf("OnModify trigger failed: %w", err)
		}
	}

	// Call Validate if implemented
	if validator, ok := record.(types.Validator); ok {
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// TODO: Implement actual update logic

	return nil
}

// Delete deletes a record and calls OnDelete trigger if implemented
func (r *Repository) Delete(tableName string, record interface{}) error {
	if r.db.conn == nil {
		return fmt.Errorf("database not open")
	}

	if r.db.currentCompany == "" {
		return fmt.Errorf("no company context set")
	}

	// Call OnDelete trigger if implemented
	if trigger, ok := record.(types.BeforeDelete); ok {
		if err := trigger.OnDelete(r.db.conn, r.db.currentCompany); err != nil {
			return fmt.Errorf("OnDelete trigger failed: %w", err)
		}
	}

	// TODO: Implement actual delete logic

	return nil
}

// Exec executes a raw SQL query
func (r *Repository) Exec(query string, args ...interface{}) (sql.Result, error) {
	if r.db.conn == nil {
		return nil, fmt.Errorf("database not open")
	}

	return r.db.conn.Exec(query, args...)
}

// Query executes a raw SQL query and returns rows
func (r *Repository) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if r.db.conn == nil {
		return nil, fmt.Errorf("database not open")
	}

	return r.db.conn.Query(query, args...)
}

// QueryRow executes a raw SQL query and returns a single row
func (r *Repository) QueryRow(query string, args ...interface{}) *sql.Row {
	return r.db.conn.QueryRow(query, args...)
}
