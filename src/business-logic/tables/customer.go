package tables

import (
	"database/sql"
	"errors"

	"github.com/hansjlachmann/openerp/src/foundation/types"
)

//go:generate go run ../../../tools/tablegen/main.go

// NewCustomer creates a new Customer instance
func NewCustomer() *Customer {
	return &Customer{}
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *Customer) OnInsert() error {
	t.Status = Customer_Status.Open
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *Customer) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *Customer) OnDelete(db *sql.DB, company string) error {
	// TODO: Add checks for related records (if any)
	// Example:
	// var count int
	// err := db.QueryRow(
	//     fmt.Sprintf(`SELECT COUNT(*) FROM "%s$OtherTable" WHERE customer_code = $1`, company),
	//     t.primaryKeyValue,
	// ).Scan(&count)
	// if count > 0 {
	//     return fmt.Errorf("cannot delete: Customer is used by %d records", count)
	// }

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
// Field Validation Hooks
// ========================================
// These methods are called by auto-generated OnValidate triggers in customer_gen.go
// Add your custom field validation logic here

// CustomValidate_No - Custom validation for no field
func (t *Customer) CustomValidate_No() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for no:
	// if len(t.No) < 3 {
	//     return errors.New("no must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Name - Custom validation for name field
func (t *Customer) CustomValidate_Name() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example: Require name to be at least 3 characters
	if len(t.Name) > 0 && len(t.Name) < 3 {
		return errors.New("name must be at least 3 characters")
	}

	return nil
}

// CustomValidate_Address - Custom validation for address field
func (t *Customer) CustomValidate_Address() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for address:
	// if len(t.Address) < 3 {
	//     return errors.New("address must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Post_code - Custom validation for post_code field
func (t *Customer) CustomValidate_Post_code() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for post_code:
	// if len(t.Post_code) < 3 {
	//     return errors.New("post_code must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_City - Custom validation for city field
func (t *Customer) CustomValidate_City() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for city:
	// if len(t.City) < 3 {
	//     return errors.New("city must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Phonenumber - Custom validation for phonenumber field
func (t *Customer) CustomValidate_Phonenumber() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for phonenumber:
	// if len(t.Phonenumber) < 3 {
	//     return errors.New("phonenumber must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Payment_terms_code - Custom validation for payment_terms_code field
func (t *Customer) CustomValidate_Payment_terms_code() error {
	// Table relation validation - payment_terms_code must exist in PaymentTerms
	if t.Payment_terms_code != "" && t.Payment_terms_code != types.Code("") {
		var relatedRecord PaymentTerms
		relatedRecord.Init(t.db, t.company)
		if !relatedRecord.Get(t.Payment_terms_code) {
			return errors.New("payment_terms_code does not exist in PaymentTerms table")
		}

		// *** ADD YOUR CUSTOM LOGIC HERE ***
		// You can access the related record:
		if !relatedRecord.Active {
			return errors.New("payment terms is inactive and cannot be used")
		}
	}

	return nil
}

// CustomValidate_Status - Custom validation for status field
func (t *Customer) CustomValidate_Status() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example: Prevent changing to Posted status if certain conditions aren't met
	// if t.Status == Customer_Status.Posted {
	//     if t.Name == "" {
	//         return errors.New("cannot post customer without name")
	//     }
	// }

	return nil
}

// CustomValidate_Credit_limit - Custom validation for credit_limit field
func (t *Customer) CustomValidate_Credit_limit() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example: Ensure credit limit is non-negative
	// if t.Credit_limit.IsNegative() {
	//     return errors.New("credit limit cannot be negative")
	// }

	return nil
}

// CustomValidate_Last_order_date - Custom validation for last_order_date field
func (t *Customer) CustomValidate_Last_order_date() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example: Ensure last order date is not in the future
	// if !t.Last_order_date.IsZero() && t.Last_order_date.After(types.Today()) {
	//     return errors.New("last order date cannot be in the future")
	// }

	return nil
}

// CustomValidate_Created_at - Custom validation for created_at field
func (t *Customer) CustomValidate_Created_at() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example: Ensure created timestamp is not in the future
	// if !t.Created_at.IsZero() && t.Created_at.After(types.Now()) {
	//     return errors.New("created timestamp cannot be in the future")
	// }

	return nil
}

// CustomValidate_Profile_photo - Custom validation for profile_photo field
func (t *Customer) CustomValidate_Profile_photo() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example: Validate photo size
	// if len(t.Profile_photo) > 5*1024*1024 {
	//     return errors.New("profile photo cannot exceed 5MB")
	// }

	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
// Example:
// func (t *Customer) CalculateSomething() error {
//     // Your logic here
//     return nil
// }
