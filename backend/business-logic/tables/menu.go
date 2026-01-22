package tables

import (
	"errors"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// Menu wraps MenuBase and adds trigger methods
type Menu struct {
	gtables.MenuBase
}

// NewMenu creates a new Menu instance
func NewMenu() *Menu {
	return &Menu{}
}

// Init initializes the record with database context and sets up triggers
func (t *Menu) Init(db database.Executor, company string) {
	t.MenuBase.Init(db, company)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *Menu) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *Menu) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *Menu) OnDelete(db database.Executor, company string) error {
	// TODO: Add checks for related records (if any)
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *Menu) OnRename() error {
	// TODO: Update related records if needed
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *Menu) Validate() error {
	if t.Code.IsEmpty() {
		return errors.New("code is required")
	}
	if len(t.Code) > 20 {
		return errors.New("code cannot exceed 20 characters")
	}
	if len(t.Description) > 50 {
		return errors.New("description cannot exceed 50 characters")
	}
	if len(t.Filename) > 50 {
		return errors.New("filename cannot exceed 50 characters")
	}

	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
// Example:
// func (t *Menu) CalculateSomething() error {
//     // Your logic here
//     return nil
// }
