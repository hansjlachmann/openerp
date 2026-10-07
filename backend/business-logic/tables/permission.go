package tables

import (
	"errors"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// Permission wraps PermissionBase and adds trigger methods
type Permission struct {
	gtables.PermissionBase
}

// NewPermission creates a new Permission instance
func NewPermission() *Permission {
	return &Permission{}
}

// InitWithDBType initializes the record with database context and type and sets up
// triggers. The API creates tables via the tables.Table interface and calls this method,
// so the wiring must live here for triggers and OnValidate_* overrides to fire.
func (t *Permission) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.PermissionBase.InitWithDBType(db, company, dbType)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
	t.SetSelf(t)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *Permission) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *Permission) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *Permission) OnDelete(db database.Executor, company string) error {
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *Permission) OnRename() error {
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *Permission) Validate() error {
	if t.Role_id.IsEmpty() {
		return errors.New("role_id is required")
	}
	if len(t.Role_id) > 20 {
		return errors.New("role_id cannot exceed 20 characters")
	}
	if t.Table_name.IsEmpty() {
		return errors.New("table_name is required")
	}
	if len(t.Table_name) > 100 {
		return errors.New("table_name cannot exceed 100 characters")
	}
	return nil
}
