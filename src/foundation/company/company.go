package company

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/hansjlachmann/openerp/src/foundation/database"
)

// Manager handles company operations
type Manager struct {
	db *database.Database
}

// NewManager creates a new company manager
func NewManager(db *database.Database) *Manager {
	return &Manager{db: db}
}

// CreateCompany creates a new company in the database
func (m *Manager) CreateCompany(name string) error {
	if m.db.GetConnection() == nil {
		return fmt.Errorf("database not open")
	}

	// Validate company name
	if err := database.ValidateCompanyName(name); err != nil {
		return err
	}

	_, err := m.db.GetConnection().Exec(`INSERT INTO "Company" (name) VALUES ($1)`, name)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return fmt.Errorf("company '%s' already exists", name)
		}
		return fmt.Errorf("failed to create company: %w", err)
	}

	return nil
}

// EnterCompany sets the current company context (session-based)
func (m *Manager) EnterCompany(name string) error {
	if m.db.GetConnection() == nil {
		return fmt.Errorf("database not open")
	}

	if name == "" {
		return fmt.Errorf("company name cannot be empty")
	}

	// Check if already in a company
	if m.db.GetCurrentCompany() != "" {
		return fmt.Errorf("already in company '%s' - exit first before entering another", m.db.GetCurrentCompany())
	}

	// Verify company exists
	var companyName string
	err := m.db.GetConnection().QueryRow(`SELECT name FROM "Company" WHERE name = $1`, name).Scan(&companyName)
	if err == sql.ErrNoRows {
		return fmt.Errorf("company '%s' does not exist", name)
	}
	if err != nil {
		return fmt.Errorf("failed to verify company: %w", err)
	}

	// Set current company (per-connection state)
	m.db.SetCurrentCompany(name)
	return nil
}

// ExitCompany clears the current company context
func (m *Manager) ExitCompany() error {
	if m.db.GetConnection() == nil {
		return fmt.Errorf("database not open")
	}

	if m.db.GetCurrentCompany() == "" {
		return fmt.Errorf("no company session active")
	}

	m.db.SetCurrentCompany("")
	return nil
}

// DeleteCompany deletes a company and all its Company$Tables
func (m *Manager) DeleteCompany(name string) error {
	if m.db.GetConnection() == nil {
		return fmt.Errorf("database not open")
	}

	if name == "" {
		return fmt.Errorf("company name cannot be empty")
	}

	// Verify company exists
	var companyName string
	err := m.db.GetConnection().QueryRow(`SELECT name FROM "Company" WHERE name = $1`, name).Scan(&companyName)
	if err == sql.ErrNoRows {
		return fmt.Errorf("company '%s' does not exist", name)
	}
	if err != nil {
		return fmt.Errorf("failed to verify company: %w", err)
	}

	// If this is the current company, exit it first
	if m.db.GetCurrentCompany() == name {
		m.db.SetCurrentCompany("")
	}

	// Find all tables belonging to this company (Company$TableName pattern)
	rows, err := m.db.GetConnection().Query(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name LIKE $1
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
		_, err := m.db.GetConnection().Exec(fmt.Sprintf(`DROP TABLE IF EXISTS "%s"`, tableName))
		if err != nil {
			return fmt.Errorf("failed to drop table %s: %w", tableName, err)
		}
	}

	// Delete field definitions for this company
	_, err = m.db.GetConnection().Exec(`DELETE FROM "FieldDefinition" WHERE company = $1`, name)
	if err != nil {
		return fmt.Errorf("failed to delete field definitions: %w", err)
	}

	// Delete company record
	_, err = m.db.GetConnection().Exec(`DELETE FROM "Company" WHERE name = $1`, name)
	if err != nil {
		return fmt.Errorf("failed to delete company: %w", err)
	}

	return nil
}

// ListCompanies returns all companies in the database
func (m *Manager) ListCompanies() ([]string, error) {
	if m.db.GetConnection() == nil {
		return nil, fmt.Errorf("database not open")
	}

	rows, err := m.db.GetConnection().Query(`SELECT name FROM "Company" ORDER BY name`)
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
func (m *Manager) GetCurrentCompany() string {
	return m.db.GetCurrentCompany()
}
