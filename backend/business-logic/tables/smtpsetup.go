package tables

import (
	"errors"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// SMTPSetup wraps SMTPSetupBase and adds trigger methods
type SMTPSetup struct {
	gtables.SMTPSetupBase
}

// NewSMTPSetup creates a new SMTPSetup instance
func NewSMTPSetup() *SMTPSetup {
	return &SMTPSetup{}
}

// Init initializes the record with database context and sets up triggers
func (t *SMTPSetup) Init(db database.Executor, company string) {
	t.SMTPSetupBase.Init(db, company)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *SMTPSetup) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *SMTPSetup) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *SMTPSetup) OnDelete(db database.Executor, company string) error {
	// TODO: Add checks for related records (if any)
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *SMTPSetup) OnRename() error {
	// TODO: Update related records if needed
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *SMTPSetup) Validate() error {
	// primary_key is a blank singleton key (BC-style setup table) - no required check
	if len(t.Primary_key) > 20 {
		return errors.New("primary_key cannot exceed 20 characters")
	}
	if len(t.Smtp_server) > 250 {
		return errors.New("smtp_server cannot exceed 250 characters")
	}
	if len(t.User_id) > 100 {
		return errors.New("user_id cannot exceed 100 characters")
	}
	if len(t.Password) > 250 {
		return errors.New("password cannot exceed 250 characters")
	}
	if len(t.From_address) > 100 {
		return errors.New("from_address cannot exceed 100 characters")
	}

	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
// Example:
// func (t *SMTPSetup) CalculateSomething() error {
//     // Your logic here
//     return nil
// }
