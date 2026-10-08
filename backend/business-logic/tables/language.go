package tables

import (
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"

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

// InitWithDBType initializes the record with database context and type and sets up
// triggers. The API creates tables via the tables.Table interface and calls this method,
// so the wiring must live here for triggers and OnValidate_* overrides to fire.
func (t *Language) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.LanguageBase.InitWithDBType(db, company, dbType)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
	t.SetSelf(t)
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
	if err := ftables.CheckRequired(gtables.LanguageTableName, "code", t.Code.IsEmpty()); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.LanguageTableName, "code", string(t.Code), 10); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.LanguageTableName, "name", string(t.Name), 50); err != nil {
		return err
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

	// Exactly 5 characters with a dash at position 3
	if len(key) != 5 || key[2] != '-' {
		return apperrors.FieldFormat(gtables.LanguageTableName, "translation_key", "xx-XX")
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

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
// Example:
// func (t *Language) CalculateSomething() error {
//     // Your logic here
//     return nil
// }
