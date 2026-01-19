package tables

import (
	"errors"

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

// Init initializes the record with database context and sets up triggers
func (t *CustomerLedgerEntry) Init(db database.Executor, company string) {
	t.CustomerLedgerEntryBase.Init(db, company)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
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
	if len(t.Customer_no) > 20 {
		return errors.New("customer_no cannot exceed 20 characters")
	}
	if len(t.Sell_to_customer_no) > 20 {
		return errors.New("sell_to_customer_no cannot exceed 20 characters")
	}
	if len(t.Document_no) > 20 {
		return errors.New("document_no cannot exceed 20 characters")
	}
	if len(t.External_document_no) > 20 {
		return errors.New("external_document_no cannot exceed 20 characters")
	}
	if len(t.Description) > 100 {
		return errors.New("description cannot exceed 100 characters")
	}
	if len(t.Currency_code) > 10 {
		return errors.New("currency_code cannot exceed 10 characters")
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
		relatedRecord.Init(t.GetDB(), t.GetCompany())
		if !relatedRecord.Get(t.Customer_no) {
			return errors.New("customer_no does not exist in Customer table")
		}
	}
	return nil
}

// OnValidate_Sell_to_customer_no validates the sell-to customer relation
func (t *CustomerLedgerEntry) OnValidate_Sell_to_customer_no() error {
	if t.Sell_to_customer_no != "" && t.Sell_to_customer_no != types.Code("") {
		var relatedRecord Customer
		relatedRecord.Init(t.GetDB(), t.GetCompany())
		if !relatedRecord.Get(t.Sell_to_customer_no) {
			return errors.New("sell_to_customer_no does not exist in Customer table")
		}
	}
	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
