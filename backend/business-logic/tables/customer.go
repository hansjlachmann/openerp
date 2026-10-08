package tables

import (
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	"unicode/utf8"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// Customer wraps CustomerBase and adds trigger methods
type Customer struct {
	gtables.CustomerBase
}

// NewCustomer creates a new Customer instance
func NewCustomer() *Customer {
	return &Customer{}
}

// InitWithDBType initializes the record with database context and type and sets up
// triggers. The API creates tables via the tables.Table interface and calls this method,
// so the wiring must live here for triggers and OnValidate_* overrides to fire.
func (t *Customer) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.CustomerBase.InitWithDBType(db, company, dbType)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
	t.SetSelf(t)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *Customer) OnInsert() error {
	t.Status = gtables.Customer_Status.Open
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *Customer) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *Customer) OnDelete(db database.Executor, company string) error {
	// TODO: Add checks for related records (if any)
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *Customer) OnRename() error {
	// TODO: Update related records if needed
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *Customer) Validate() error {
	if err := ftables.CheckRequired(gtables.CustomerTableName, "no", t.No.IsEmpty()); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.CustomerTableName, "no", string(t.No), 20); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.CustomerTableName, "name", string(t.Name), 50); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.CustomerTableName, "address", string(t.Address), 50); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.CustomerTableName, "post_code", string(t.Post_code), 20); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.CustomerTableName, "city", string(t.City), 50); err != nil {
		return err
	}

	return nil
}

// ========================================
// Field Validation Overrides
// ========================================

// OnValidate_Payment_terms_code validates the payment terms relation
func (t *Customer) OnValidate_Payment_terms_code() error {
	// Check if the payment terms exists and is active
	if t.Payment_terms_code != "" && t.Payment_terms_code != types.Code("") {
		var relatedRecord PaymentTerms
		relatedRecord.InitWithDBType(t.GetDB(), t.GetCompany(), t.GetDBType())
		if !relatedRecord.Get(t.Payment_terms_code) {
			return apperrors.RelatedNotFound(gtables.CustomerTableName, "payment_terms_code", t.Payment_terms_code.String())
		}
		if !relatedRecord.Active {
			return apperrors.RelatedInactive(gtables.CustomerTableName, "payment_terms_code", t.Payment_terms_code.String())
		}
	}
	return nil
}

// OnValidate_Name validates the name field
func (t *Customer) OnValidate_Name() error {
	if n := utf8.RuneCountInString(string(t.Name)); n > 0 && n < 3 {
		return apperrors.FieldTooShort(gtables.CustomerTableName, "name", 3)
	}
	return nil
}

// ========================================
// Embedded Method Wrappers
// ========================================

// CopyFilters copies filters from another Customer record
func (t *Customer) CopyFilters(from *Customer) {
	t.CustomerBase.CopyFilters(&from.CustomerBase)
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
