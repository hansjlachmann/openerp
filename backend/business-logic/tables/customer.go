package tables

import (
	"errors"

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

// Init initializes the record with database context and sets up triggers
func (t *Customer) Init(db database.Executor, company string) {
	t.CustomerBase.Init(db, company)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
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
	if t.No.IsEmpty() {
		return errors.New("no is required")
	}
	if len(t.No) > 20 {
		return errors.New("no cannot exceed 20 characters")
	}
	if len(t.Name) > 50 {
		return errors.New("name cannot exceed 50 characters")
	}
	if len(t.Address) > 50 {
		return errors.New("address cannot exceed 50 characters")
	}
	if len(t.Post_code) > 20 {
		return errors.New("post_code cannot exceed 20 characters")
	}
	if len(t.City) > 50 {
		return errors.New("city cannot exceed 50 characters")
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
		relatedRecord.Init(t.GetDB(), t.GetCompany())
		if !relatedRecord.Get(t.Payment_terms_code) {
			return errors.New("payment terms code does not exist")
		}
		if !relatedRecord.Active {
			return errors.New("payment terms is inactive and cannot be used")
		}
	}
	return nil
}

// OnValidate_Name validates the name field
func (t *Customer) OnValidate_Name() error {
	if len(t.Name) > 0 && len(t.Name) < 3 {
		return errors.New("name must be at least 3 characters")
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
