package tables

import (
	"errors"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// CountryRegion wraps CountryRegionBase and adds trigger methods
type CountryRegion struct {
	gtables.CountryRegionBase
}

// NewCountryRegion creates a new CountryRegion instance
func NewCountryRegion() *CountryRegion {
	return &CountryRegion{}
}

// Init initializes the record with database context and sets up triggers
func (t *CountryRegion) Init(db database.Executor, company string) {
	t.InitWithDBType(db, company, database.DBTypeSQLite)
}

// InitWithDBType initializes the record with database context and type and sets up
// triggers. The API creates tables via the tables.Table interface and calls this method,
// so the wiring must live here for triggers and OnValidate_* overrides to fire.
func (t *CountryRegion) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.CountryRegionBase.InitWithDBType(db, company, dbType)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
	t.SetSelf(t)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *CountryRegion) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *CountryRegion) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *CountryRegion) OnDelete(db database.Executor, company string) error {
	// TODO: Add checks for related records (if any)
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *CountryRegion) OnRename() error {
	// TODO: Update related records if needed
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *CountryRegion) Validate() error {
	if t.Code.IsEmpty() {
		return errors.New("code is required")
	}
	if len(t.Code) > 10 {
		return errors.New("code cannot exceed 10 characters")
	}
	if len(t.Name) > 50 {
		return errors.New("name cannot exceed 50 characters")
	}

	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
