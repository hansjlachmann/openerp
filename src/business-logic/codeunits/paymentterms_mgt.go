package codeunits

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// PaymentTermsManagement - Codeunit 50000: Payment Terms Management
// Contains functions to insert, modify, delete and query Payment Terms
const PaymentTermsManagementID = 50000

type PaymentTermsManagement struct {
	db      *sql.DB
	company string
}

// NewPaymentTermsManagement creates a new instance of the codeunit
func NewPaymentTermsManagement(db *sql.DB, company string) *PaymentTermsManagement {
	return &PaymentTermsManagement{
		db:      db,
		company: company,
	}
}

// ========================================
// Insert Operations
// ========================================

// Insert creates a new Payment Terms record
func (c *PaymentTermsManagement) Insert(code types.Code, description types.Text, active bool) error {
	// Create new record
	pt := tables.NewPaymentTerms()

	// Set fields using setters (with validation)
	if err := pt.SetCode(code); err != nil {
		return fmt.Errorf("invalid code: %w", err)
	}

	if err := pt.SetDescription(description); err != nil {
		return fmt.Errorf("invalid description: %w", err)
	}

	if err := pt.SetActive(active); err != nil {
		return fmt.Errorf("invalid active: %w", err)
	}

	// Call OnInsert trigger
	if err := pt.OnInsert(); err != nil {
		return fmt.Errorf("OnInsert trigger failed: %w", err)
	}

	// Insert into database
	tableName := fmt.Sprintf("%s$%s", c.company, tables.PaymentTermsTableName)
	_, err := c.db.Exec(
		fmt.Sprintf(`INSERT INTO "%s" (code, description, active) VALUES (?, ?, ?)`, tableName),
		pt.Code(),
		pt.Description(),
		pt.Active(),
	)

	if err != nil {
		return fmt.Errorf("failed to insert Payment Terms: %w", err)
	}

	return nil
}

// InsertMultiple inserts multiple Payment Terms records (bulk insert)
func (c *PaymentTermsManagement) InsertMultiple(records []struct {
	Code        types.Code
	Description types.Text
	Active      bool
}) error {
	for _, rec := range records {
		if err := c.Insert(rec.Code, rec.Description, rec.Active); err != nil {
			return fmt.Errorf("failed to insert %s: %w", rec.Code, err)
		}
	}
	return nil
}

// ========================================
// Modify Operations
// ========================================

// Modify updates an existing Payment Terms record
func (c *PaymentTermsManagement) Modify(code types.Code, description types.Text, active bool) error {
	// Get existing record first
	existing, err := c.Get(code)
	if err != nil {
		return fmt.Errorf("record not found: %w", err)
	}

	// Update fields using setters
	if err := existing.SetDescription(description); err != nil {
		return fmt.Errorf("invalid description: %w", err)
	}

	if err := existing.SetActive(active); err != nil {
		return fmt.Errorf("invalid active: %w", err)
	}

	// Call OnModify trigger
	if err := existing.OnModify(); err != nil {
		return fmt.Errorf("OnModify trigger failed: %w", err)
	}

	// Update in database
	tableName := fmt.Sprintf("%s$%s", c.company, tables.PaymentTermsTableName)
	_, err = c.db.Exec(
		fmt.Sprintf(`UPDATE "%s" SET description = ?, active = ? WHERE code = ?`, tableName),
		existing.Description(),
		existing.Active(),
		existing.Code(),
	)

	if err != nil {
		return fmt.Errorf("failed to update Payment Terms: %w", err)
	}

	return nil
}

// ========================================
// Delete Operations
// ========================================

// Delete removes a Payment Terms record
func (c *PaymentTermsManagement) Delete(code types.Code) error {
	// Get existing record first
	existing, err := c.Get(code)
	if err != nil {
		return fmt.Errorf("record not found: %w", err)
	}

	// Call OnDelete trigger (checks if record is in use)
	if err := existing.OnDelete(c.db, c.company); err != nil {
		return fmt.Errorf("OnDelete trigger failed: %w", err)
	}

	// Delete from database
	tableName := fmt.Sprintf("%s$%s", c.company, tables.PaymentTermsTableName)
	result, err := c.db.Exec(
		fmt.Sprintf(`DELETE FROM "%s" WHERE code = ?`, tableName),
		code,
	)

	if err != nil {
		return fmt.Errorf("failed to delete Payment Terms: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no record deleted (code: %s)", code)
	}

	return nil
}

// ========================================
// Query Operations
// ========================================

// Get retrieves a single Payment Terms record by code
func (c *PaymentTermsManagement) Get(code types.Code) (*tables.PaymentTerms, error) {
	tableName := fmt.Sprintf("%s$%s", c.company, tables.PaymentTermsTableName)

	var codeStr, descStr string
	var active int

	err := c.db.QueryRow(
		fmt.Sprintf(`SELECT code, description, active FROM "%s" WHERE code = ?`, tableName),
		code,
	).Scan(&codeStr, &descStr, &active)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Payment Terms %s not found", code)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query Payment Terms: %w", err)
	}

	// Create record and populate
	pt := tables.NewPaymentTerms()
	pt.SetCode(types.NewCode(codeStr))
	pt.SetDescription(types.NewText(descStr))
	pt.SetActive(active != 0)

	return pt, nil
}

// Exists checks if a Payment Terms code exists
func (c *PaymentTermsManagement) Exists(code types.Code) (bool, error) {
	tableName := fmt.Sprintf("%s$%s", c.company, tables.PaymentTermsTableName)

	var count int
	err := c.db.QueryRow(
		fmt.Sprintf(`SELECT COUNT(*) FROM "%s" WHERE code = ?`, tableName),
		code,
	).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return count > 0, nil
}

// List retrieves all Payment Terms records
func (c *PaymentTermsManagement) List() ([]*tables.PaymentTerms, error) {
	tableName := fmt.Sprintf("%s$%s", c.company, tables.PaymentTermsTableName)

	rows, err := c.db.Query(
		fmt.Sprintf(`SELECT code, description, active FROM "%s" ORDER BY code`, tableName),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query Payment Terms: %w", err)
	}
	defer rows.Close()

	var results []*tables.PaymentTerms

	for rows.Next() {
		var codeStr, descStr string
		var active int

		if err := rows.Scan(&codeStr, &descStr, &active); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		pt := tables.NewPaymentTerms()
		pt.SetCode(types.NewCode(codeStr))
		pt.SetDescription(types.NewText(descStr))
		pt.SetActive(active != 0)

		results = append(results, pt)
	}

	return results, nil
}

// ListActive retrieves only active Payment Terms records
func (c *PaymentTermsManagement) ListActive() ([]*tables.PaymentTerms, error) {
	tableName := fmt.Sprintf("%s$%s", c.company, tables.PaymentTermsTableName)

	rows, err := c.db.Query(
		fmt.Sprintf(`SELECT code, description, active FROM "%s" WHERE active = 1 ORDER BY code`, tableName),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query active Payment Terms: %w", err)
	}
	defer rows.Close()

	var results []*tables.PaymentTerms

	for rows.Next() {
		var codeStr, descStr string
		var active int

		if err := rows.Scan(&codeStr, &descStr, &active); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		pt := tables.NewPaymentTerms()
		pt.SetCode(types.NewCode(codeStr))
		pt.SetDescription(types.NewText(descStr))
		pt.SetActive(active != 0)

		results = append(results, pt)
	}

	return results, nil
}

// Count returns the total number of Payment Terms records
func (c *PaymentTermsManagement) Count() (int, error) {
	tableName := fmt.Sprintf("%s$%s", c.company, tables.PaymentTermsTableName)

	var count int
	err := c.db.QueryRow(
		fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, tableName),
	).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to count Payment Terms: %w", err)
	}

	return count, nil
}

// ========================================
// Utility Functions
// ========================================

// InitializeDefaultRecords creates standard Payment Terms (like BC default data)
func (c *PaymentTermsManagement) InitializeDefaultRecords() error {
	defaults := []struct {
		Code        types.Code
		Description types.Text
		Active      bool
	}{
		{types.NewCode("0D"), types.NewText("Net Due on Receipt"), true},
		{types.NewCode("14D"), types.NewText("Net 14 Days"), true},
		{types.NewCode("30D"), types.NewText("Net 30 Days"), true},
		{types.NewCode("60D"), types.NewText("Net 60 Days"), true},
		{types.NewCode("COD"), types.NewText("Cash on Delivery"), true},
		{types.NewCode("PREPMT"), types.NewText("Prepayment"), true},
	}

	for _, def := range defaults {
		// Check if already exists
		exists, err := c.Exists(def.Code)
		if err != nil {
			return err
		}
		if exists {
			continue // Skip if already exists
		}

		// Insert
		if err := c.Insert(def.Code, def.Description, def.Active); err != nil {
			return fmt.Errorf("failed to create default %s: %w", def.Code, err)
		}
	}

	return nil
}

// ========================================
// CLI Integration
// ========================================

// RunCLI executes the Payment Terms Management codeunit from CLI
func (c *PaymentTermsManagement) RunCLI() error {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("CODEUNIT 50000: Payment Terms Management")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Company: %s\n", c.company)
	fmt.Println(strings.Repeat("-", 60))

	// Simple function: Insert one test record
	fmt.Println("\nInserting test record...")
	fmt.Println("  Code: TEST")
	fmt.Println("  Description: Test Payment Terms")
	fmt.Println("  Active: Yes")

	err := c.Insert(
		types.NewCode("TEST"),
		types.NewText("Test Payment Terms"),
		true,
	)

	if err != nil {
		return fmt.Errorf("failed to insert test record: %w", err)
	}

	fmt.Println("\n✓ Test record inserted successfully!")

	// Show the inserted record
	fmt.Println("\nVerifying inserted record...")
	record, err := c.Get(types.NewCode("TEST"))
	if err != nil {
		return fmt.Errorf("failed to retrieve record: %w", err)
	}

	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Printf("Code:        %s\n", record.Code())
	fmt.Printf("Description: %s\n", record.Description())
	fmt.Printf("Active:      %v\n", record.Active())
	fmt.Println(strings.Repeat("-", 60))

	fmt.Println("\n✓ Codeunit execution complete!")

	return nil
}
