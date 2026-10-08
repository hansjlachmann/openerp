package tables

import (
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"

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

// InitWithDBType initializes the record with database context and type and sets up
// triggers. The API creates tables via the tables.Table interface and calls this method,
// so the wiring must live here for triggers and OnValidate_* overrides to fire.
func (t *Menu) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.MenuBase.InitWithDBType(db, company, dbType)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
	t.SetSelf(t)
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
	if err := ftables.CheckRequired(gtables.MenuTableName, "code", t.Code.IsEmpty()); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.MenuTableName, "code", string(t.Code), 20); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.MenuTableName, "description", string(t.Description), 50); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.MenuTableName, "filename", string(t.Filename), 50); err != nil {
		return err
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
