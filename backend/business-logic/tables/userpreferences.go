package tables

import (
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// UserPreferences wraps UserPreferencesBase and adds trigger methods
type UserPreferences struct {
	gtables.UserPreferencesBase
}

// NewUserPreferences creates a new UserPreferences instance
func NewUserPreferences() *UserPreferences {
	return &UserPreferences{}
}

// Init initializes the record with database context and sets up triggers
func (t *UserPreferences) Init(db database.Executor, company string) {
	t.UserPreferencesBase.Init(db, company)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *UserPreferences) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *UserPreferences) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *UserPreferences) OnDelete(db database.Executor, company string) error {
	// No related records to check
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *UserPreferences) OnRename() error {
	// TODO: Update related records if needed
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *UserPreferences) Validate() error {
	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
