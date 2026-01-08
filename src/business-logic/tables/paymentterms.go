package tables

import (
	"errors"

	"github.com/hansjlachmann/openerp/src/foundation/database"
)

//go:generate go run ../../../tools/tablegen/main.go

// NewPaymentTerms creates a new PaymentTerms instance
func NewPaymentTerms() *PaymentTerms {
	return &PaymentTerms{
	}
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
	// Example:
	// var count int
	// err := db.QueryRow(
	//     fmt.Sprintf(`SELECT COUNT(*) FROM "%s$OtherTable" WHERE paymentTerms_code = $1`, company),
	//     t.primaryKeyValue,
	// ).Scan(&count)
	// if count > 0 {
	//     return fmt.Errorf("cannot delete: Payment Terms is used by %d records", count)
	// }

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
// Field Validation Hooks
// ========================================
// These methods are called by auto-generated OnValidate triggers in paymentterms_gen.go
// Add your custom field validation logic here

// CustomValidate_Code - Custom validation for code field
func (t *PaymentTerms) CustomValidate_Code() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for code:
	// if len(t.Code) < 3 {
	//     return errors.New("code must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Description - Custom validation for description field
func (t *PaymentTerms) CustomValidate_Description() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for description:
	// if len(t.Description) < 3 {
	//     return errors.New("description must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Active - Custom validation for active field
func (t *PaymentTerms) CustomValidate_Active() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***

	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
// Example:
// func (t *PaymentTerms) CalculateSomething() error {
//     // Your logic here
//     return nil
// }
