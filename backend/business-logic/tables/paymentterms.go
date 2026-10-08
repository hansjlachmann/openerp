package tables

import (
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// PaymentTerms wraps PaymentTermsBase and adds trigger methods
type PaymentTerms struct {
	gtables.PaymentTermsBase
}

// NewPaymentTerms creates a new PaymentTerms instance
func NewPaymentTerms() *PaymentTerms {
	return &PaymentTerms{}
}

// InitWithDBType initializes the record with database context and type and sets up
// triggers. The API creates tables via the tables.Table interface and calls this method,
// so the wiring must live here for triggers and OnValidate_* overrides to fire.
func (t *PaymentTerms) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.PaymentTermsBase.InitWithDBType(db, company, dbType)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
	t.SetSelf(t)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *PaymentTerms) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *PaymentTerms) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *PaymentTerms) OnDelete(db database.Executor, company string) error {
	// TODO: Add checks for related records (if any)
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *PaymentTerms) OnRename() error {
	// TODO: Update related records if needed
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *PaymentTerms) Validate() error {
	if err := ftables.CheckRequired(gtables.PaymentTermsTableName, "code", t.Code.IsEmpty()); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.PaymentTermsTableName, "code", string(t.Code), 10); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.PaymentTermsTableName, "description", string(t.Description), 30); err != nil {
		return err
	}

	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
