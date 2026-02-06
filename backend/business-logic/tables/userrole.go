package tables

import (
	"errors"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// UserRole wraps UserRoleBase and adds trigger methods
type UserRole struct {
	gtables.UserRoleBase
}

// NewUserRole creates a new UserRole instance
func NewUserRole() *UserRole {
	return &UserRole{}
}

// Init initializes the record with database context and sets up triggers
func (t *UserRole) Init(db database.Executor, company string) {
	t.UserRoleBase.Init(db, company)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *UserRole) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *UserRole) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *UserRole) OnDelete(db database.Executor, company string) error {
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *UserRole) OnRename() error {
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *UserRole) Validate() error {
	if t.Code.IsEmpty() {
		return errors.New("code is required")
	}
	if len(t.Code) > 20 {
		return errors.New("code cannot exceed 20 characters")
	}
	if len(t.Description) > 50 {
		return errors.New("description cannot exceed 50 characters")
	}

	return nil
}
