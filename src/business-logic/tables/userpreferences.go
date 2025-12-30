package tables

import (
	"github.com/hansjlachmann/openerp/src/foundation/database"
)

//go:generate go run ../../../tools/tablegen/main.go

// NewUserPreferences creates a new UserPreferences instance
func NewUserPreferences() *UserPreferences {
	return &UserPreferences{}
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
// Field Validation Hooks
// ========================================
// These methods are called by auto-generated OnValidate triggers in userPreferences_gen.go
// Add your custom field validation logic here

// CustomValidate_User_id - Custom validation for user_id field
func (t *UserPreferences) CustomValidate_User_id() error {
	// User ID is required
	return nil
}

// CustomValidate_Page_id - Custom validation for page_id field
func (t *UserPreferences) CustomValidate_Page_id() error {
	// Page ID is required
	return nil
}

// CustomValidate_Preference_type - Custom validation for preference_type field
func (t *UserPreferences) CustomValidate_Preference_type() error {
	// Type is required
	return nil
}

// CustomValidate_Preference_name - Custom validation for preference_name field
func (t *UserPreferences) CustomValidate_Preference_name() error {
	// Name is required
	return nil
}

// CustomValidate_Preference_data - Custom validation for preference_data field
func (t *UserPreferences) CustomValidate_Preference_data() error {
	// Data is required
	return nil
}

// CustomValidate_Created_at - Custom validation for created_at field
func (t *UserPreferences) CustomValidate_Created_at() error {
	return nil
}

// CustomValidate_Updated_at - Custom validation for updated_at field
func (t *UserPreferences) CustomValidate_Updated_at() error {
	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
// Example:
// func (t *UserPreferences) CalculateSomething() error {
//     // Your logic here
//     return nil
// }
