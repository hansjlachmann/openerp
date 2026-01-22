package tables

import (
	"errors"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// Language wraps LanguageBase and adds trigger methods
type Language struct {
	gtables.LanguageBase
}

// NewLanguage creates a new Language instance
func NewLanguage() *Language {
	return &Language{}
}

// Init initializes the record with database context and sets up triggers
func (t *Language) Init(db database.Executor, company string) {
	t.LanguageBase.Init(db, company)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *Language) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *Language) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *Language) OnDelete(db database.Executor, company string) error {
	// TODO: Add checks for related records (if any)
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *Language) OnRename() error {
	// TODO: Update related records if needed
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *Language) Validate() error {
	if t.Code.IsEmpty() {
		return errors.New("code is required")
	}
	if len(t.Code) > 10 {
		return errors.New("code cannot exceed 10 characters")
	}
	if len(t.Name) > 50 {
		return errors.New("name cannot exceed 50 characters")
	}

	// Validate translation_key format
	if !t.Translation_key.IsEmpty() {
		if err := t.validateTranslationKey(); err != nil {
			return err
		}
	}

	return nil
}

// validateTranslationKey validates the translation key format (xx-XX)
func (t *Language) validateTranslationKey() error {
	key := t.Translation_key.String()

	// Must be exactly 5 characters
	if len(key) != 5 {
		return errors.New("translation key must be exactly 5 characters (format: xx-XX)")
	}

	// Position 3 (index 2) must be a dash
	if key[2] != '-' {
		return errors.New("translation key must have a dash (-) at position 3 (format: xx-XX)")
	}

	return nil
}

// OnValidate_Translation_key validates the translation_key field when changed
func (t *Language) OnValidate_Translation_key() error {
	if !t.Translation_key.IsEmpty() {
		return t.validateTranslationKey()
	}
	return nil
}

// ValidateField overrides the base ValidateField to use custom validation for translation_key
func (t *Language) ValidateField(fieldName string, value interface{}) error {
	// Call base validation first (sets the field value)
	if err := t.LanguageBase.ValidateField(fieldName, value); err != nil {
		return err
	}

	// Add custom validation for translation_key
	if fieldName == "translation_key" || fieldName == "Translation_key" {
		return t.OnValidate_Translation_key()
	}

	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
// Example:
// func (t *Language) CalculateSomething() error {
//     // Your logic here
//     return nil
// }
