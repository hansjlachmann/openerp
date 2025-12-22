package objects

import (
	"database/sql"
	"fmt"
)

// TableDefinition represents a table that can be initialized
type TableDefinition interface {
	GetTableID() int
	GetTableName() string
	GetTableSchema() string
	CreateTable(db *sql.DB, company string) error
}

// InitializeCompanyTables creates all registered tables for a new company
func (or *ObjectRegistry) InitializeCompanyTables(db *sql.DB, companyName string) error {
	// Get all registered table IDs
	tableIDs := or.ListTables()

	if len(tableIDs) == 0 {
		return fmt.Errorf("no tables registered in object registry")
	}

	successCount := 0
	failedTables := []string{}

	// Create each registered table
	for _, tableID := range tableIDs {
		tableInterface, ok := or.GetTable(tableID)
		if !ok {
			continue
		}

		// Type assert to TableDefinition
		tableDef, ok := tableInterface.(TableDefinition)
		if !ok {
			// Skip if table doesn't implement TableDefinition
			// This allows us to have table structs registered that don't need initialization
			continue
		}

		// Create the table for this company
		err := tableDef.CreateTable(db, companyName)
		if err != nil {
			failedTables = append(failedTables, fmt.Sprintf("Table %d (%s): %v", tableID, tableDef.GetTableName(), err))
			continue
		}

		successCount++
	}

	if len(failedTables) > 0 {
		errorMsg := fmt.Sprintf("Failed to create %d table(s):\n", len(failedTables))
		for _, failure := range failedTables {
			errorMsg += "  - " + failure + "\n"
		}
		return fmt.Errorf(errorMsg)
	}

	if successCount == 0 {
		return fmt.Errorf("no tables were initialized (registered tables may not implement TableDefinition interface)")
	}

	return nil
}

// GetTableCount returns the number of registered tables
func (or *ObjectRegistry) GetTableCount() int {
	return len(or.tables)
}
