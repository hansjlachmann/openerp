package tables

import (
	"errors"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// Company wraps CompanyBase and adds trigger methods
type Company struct {
	gtables.CompanyBase
}

// NewCompany creates a new Company instance
func NewCompany() *Company {
	return &Company{}
}

// Init initializes the record with database context and sets up triggers
func (t *Company) Init(db database.Executor, company string) {
	t.CompanyBase.Init(db, company)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *Company) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *Company) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *Company) OnDelete(db database.Executor, company string) error {
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *Company) OnRename() error {
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *Company) Validate() error {
	if t.Name.IsEmpty() {
		return errors.New("name is required")
	}
	if len(t.Name) > 100 {
		return errors.New("name cannot exceed 100 characters")
	}

	return nil
}
