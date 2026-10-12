package tables

import (
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// UserTheme wraps UserThemeBase and adds trigger methods
type UserTheme struct {
	gtables.UserThemeBase
}

// NewUserTheme creates a new UserTheme instance
func NewUserTheme() *UserTheme {
	return &UserTheme{}
}

// InitWithDBType initializes the record with database context and type and sets up
// triggers. The API creates tables via the tables.Table interface and calls this method,
// so the wiring must live here for triggers and OnValidate_* overrides to fire.
func (t *UserTheme) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.UserThemeBase.InitWithDBType(db, company, dbType)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
	t.SetSelf(t)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *UserTheme) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *UserTheme) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *UserTheme) OnDelete(db database.Executor, company string) error {
	// TODO: Add checks for related records (if any)
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *UserTheme) OnRename() error {
	// TODO: Update related records if needed
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *UserTheme) Validate() error {
	if err := ftables.CheckMaxLength(gtables.UserThemeTableName, "user_id", string(t.User_id), 50); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.UserThemeTableName, "theme_code", string(t.Theme_code), 50); err != nil {
		return err
	}

	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
// Example:
// func (t *UserTheme) CalculateSomething() error {
//     // Your logic here
//     return nil
// }
