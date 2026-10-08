package tables

import (
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"

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

// InitWithDBType initializes the record with database context and type and sets up
// triggers. The API creates tables via the tables.Table interface and calls this method,
// so the wiring must live here for triggers and OnValidate_* overrides to fire.
func (t *SMTPSetup) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.SMTPSetupBase.InitWithDBType(db, company, dbType)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
	t.SetSelf(t)
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
	if err := ftables.CheckMaxLength(gtables.SMTPSetupTableName, "primary_key", string(t.Primary_key), 20); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.SMTPSetupTableName, "smtp_server", string(t.Smtp_server), 250); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.SMTPSetupTableName, "user_id", string(t.User_id), 100); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.SMTPSetupTableName, "password", string(t.Password), 250); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.SMTPSetupTableName, "from_address", string(t.From_address), 100); err != nil {
		return err
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
