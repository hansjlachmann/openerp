package tables

import (
	"errors"

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

// Init initializes the record with database context and sets up triggers
func (t *UserMember) Init(db database.Executor, company string) {
	t.UserMemberBase.Init(db, company)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
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
	if t.User_id.IsEmpty() {
		return errors.New("user_id is required")
	}
	if len(t.User_id) > 50 {
		return errors.New("user_id cannot exceed 50 characters")
	}
	if t.Role_id.IsEmpty() {
		return errors.New("role_id is required")
	}
	if len(t.Role_id) > 20 {
		return errors.New("role_id cannot exceed 20 characters")
	}
	// company is optional: blank = access to all companies
	if len(t.Company) > 100 {
		return errors.New("company cannot exceed 100 characters")
	}

	return nil
}
