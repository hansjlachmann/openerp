package tables

import (
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// UserMember wraps UserMemberBase and adds trigger methods
type UserMember struct {
	gtables.UserMemberBase
}

// NewUserMember creates a new UserMember instance
func NewUserMember() *UserMember {
	return &UserMember{}
}

// InitWithDBType initializes the record with database context and type and sets up
// triggers. The API creates tables via the tables.Table interface and calls this method,
// so the wiring must live here for triggers and OnValidate_* overrides to fire.
func (t *UserMember) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.UserMemberBase.InitWithDBType(db, company, dbType)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
	t.SetSelf(t)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *UserMember) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *UserMember) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *UserMember) OnDelete(db database.Executor, company string) error {
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *UserMember) OnRename() error {
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *UserMember) Validate() error {
	if err := ftables.CheckRequired(gtables.UserMemberTableName, "user_id", t.User_id.IsEmpty()); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.UserMemberTableName, "user_id", string(t.User_id), 50); err != nil {
		return err
	}
	if err := ftables.CheckRequired(gtables.UserMemberTableName, "role_id", t.Role_id.IsEmpty()); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.UserMemberTableName, "role_id", string(t.Role_id), 20); err != nil {
		return err
	}
	// company is optional: blank = access to all companies
	if err := ftables.CheckMaxLength(gtables.UserMemberTableName, "company", string(t.Company), 100); err != nil {
		return err
	}

	return nil
}
