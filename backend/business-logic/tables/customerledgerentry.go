package tables

import (
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// CustomerLedgerEntry wraps CustomerLedgerEntryBase and adds trigger methods
type CustomerLedgerEntry struct {
	gtables.CustomerLedgerEntryBase
}

// NewCustomerLedgerEntry creates a new CustomerLedgerEntry instance
func NewCustomerLedgerEntry() *CustomerLedgerEntry {
	return &CustomerLedgerEntry{}
}

// InitWithDBType initializes the record with database context and type and sets up
// triggers. The API creates tables via the tables.Table interface and calls this method,
// so the wiring must live here for triggers and OnValidate_* overrides to fire.
func (t *CustomerLedgerEntry) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.CustomerLedgerEntryBase.InitWithDBType(db, company, dbType)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
	t.SetSelf(t)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *CustomerLedgerEntry) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *CustomerLedgerEntry) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *CustomerLedgerEntry) OnDelete(db database.Executor, company string) error {
	// TODO: Add checks for related records (if any)
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *CustomerLedgerEntry) OnRename() error {
	// TODO: Update related records if needed
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *CustomerLedgerEntry) Validate() error {
	if err := ftables.CheckMaxLength(gtables.CustomerLedgerEntryTableName, "customer_no", string(t.Customer_no), 20); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.CustomerLedgerEntryTableName, "sell_to_customer_no", string(t.Sell_to_customer_no), 20); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.CustomerLedgerEntryTableName, "document_no", string(t.Document_no), 20); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.CustomerLedgerEntryTableName, "external_document_no", string(t.External_document_no), 20); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.CustomerLedgerEntryTableName, "description", string(t.Description), 100); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.CustomerLedgerEntryTableName, "currency_code", string(t.Currency_code), 10); err != nil {
		return err
	}

	return nil
}

// ========================================
// Field Validation Overrides
// ========================================

// OnValidate_Customer_no validates the customer relation
func (t *CustomerLedgerEntry) OnValidate_Customer_no() error {
	if t.Customer_no != "" && t.Customer_no != types.Code("") {
		var relatedRecord Customer
		relatedRecord.InitWithDBType(t.GetDB(), t.GetCompany(), t.GetDBType())
		if !relatedRecord.Get(t.Customer_no) {
			return apperrors.RelatedNotFound(gtables.CustomerLedgerEntryTableName, "customer_no", t.Customer_no.String())
		}
	}
	return nil
}

// OnValidate_Sell_to_customer_no validates the sell-to customer relation
func (t *CustomerLedgerEntry) OnValidate_Sell_to_customer_no() error {
	if t.Sell_to_customer_no != "" && t.Sell_to_customer_no != types.Code("") {
		var relatedRecord Customer
		relatedRecord.InitWithDBType(t.GetDB(), t.GetCompany(), t.GetDBType())
		if !relatedRecord.Get(t.Sell_to_customer_no) {
			return apperrors.RelatedNotFound(gtables.CustomerLedgerEntryTableName, "sell_to_customer_no", t.Sell_to_customer_no.String())
		}
	}
	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
