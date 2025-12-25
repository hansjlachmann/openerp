package tables

import (
	"database/sql"
	"errors"
	"time"
)

//go:generate go run ../../../tools/tablegen/main.go

// NewPaymentTerms creates a new PaymentTerms instance
func NewPaymentTerms() *PaymentTerms {
	return &PaymentTerms{
		lastModifiedDateTime: time.Now(),
	}
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *PaymentTerms) OnInsert() error {
	t.lastModifiedDateTime = time.Now()
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *PaymentTerms) OnModify() error {
	t.lastModifiedDateTime = time.Now()
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *PaymentTerms) OnDelete(db *sql.DB, company string) error {
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
	t.lastModifiedDateTime = time.Now()
	// TODO: Update related records if needed
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *PaymentTerms) Validate() error {
	if t.code.IsEmpty() {
		return errors.New("code is required")
	}
	if len(t.code) > 10 {
		return errors.New("code cannot exceed 10 characters")
	}
	if len(t.description) > 100 {
		return errors.New("description cannot exceed 100 characters")
	}
	if t.discountPct < 0 || t.discountPct > 100 {
		return errors.New("discountPct must be between 0 and 100")
	}

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
