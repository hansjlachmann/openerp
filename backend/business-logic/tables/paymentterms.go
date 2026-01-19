package tables

import (
	"errors"

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

// Init initializes the record with database context and sets up triggers
func (t *PaymentTerms) Init(db database.Executor, company string) {
	t.PaymentTermsBase.Init(db, company)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
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
	if t.Code.IsEmpty() {
		return errors.New("code is required")
	}
	if len(t.Code) > 10 {
		return errors.New("code cannot exceed 10 characters")
	}
	if len(t.Description) > 30 {
		return errors.New("description cannot exceed 30 characters")
	}

	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
