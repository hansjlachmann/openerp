package tables

import (
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// ThemeAsset wraps ThemeAssetBase and adds trigger methods
type ThemeAsset struct {
	gtables.ThemeAssetBase
}

// NewThemeAsset creates a new ThemeAsset instance
func NewThemeAsset() *ThemeAsset {
	return &ThemeAsset{}
}

// InitWithDBType initializes the record with database context and type and sets up
// triggers. The API creates tables via the tables.Table interface and calls this method,
// so the wiring must live here for triggers and OnValidate_* overrides to fire.
func (t *ThemeAsset) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.ThemeAssetBase.InitWithDBType(db, company, dbType)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
	t.SetSelf(t)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *ThemeAsset) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *ThemeAsset) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *ThemeAsset) OnDelete(db database.Executor, company string) error {
	// TODO: Add checks for related records (if any)
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *ThemeAsset) OnRename() error {
	// TODO: Update related records if needed
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *ThemeAsset) Validate() error {
	if err := ftables.CheckMaxLength(gtables.ThemeAssetTableName, "theme_code", string(t.Theme_code), 50); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.ThemeAssetTableName, "name", string(t.Name), 20); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.ThemeAssetTableName, "content_type", string(t.Content_type), 30); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.ThemeAssetTableName, "sha256", string(t.Sha256), 64); err != nil {
		return err
	}

	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
// Example:
// func (t *ThemeAsset) CalculateSomething() error {
//     // Your logic here
//     return nil
// }
