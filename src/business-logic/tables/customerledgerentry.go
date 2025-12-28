package tables

import (
	"errors"

	"github.com/hansjlachmann/openerp/src/foundation/database"
	"github.com/hansjlachmann/openerp/src/foundation/types"
)

//go:generate go run ../../../tools/tablegen/main.go

// NewCustomerLedgerEntry creates a new CustomerLedgerEntry instance
func NewCustomerLedgerEntry() *CustomerLedgerEntry {
	return &CustomerLedgerEntry{
	}
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
	// Example:
	// var count int
	// err := db.QueryRow(
	//     fmt.Sprintf(`SELECT COUNT(*) FROM "%s$OtherTable" WHERE customerLedgerEntry_code = $1`, company),
	//     t.primaryKeyValue,
	// ).Scan(&count)
	// if count > 0 {
	//     return fmt.Errorf("cannot delete: Customer Ledger Entry is used by %d records", count)
	// }

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
	if len(t.Customer_posting_group) > 10 {
		return errors.New("customer_posting_group cannot exceed 10 characters")
	}
	if len(t.Department_code) > 10 {
		return errors.New("department_code cannot exceed 10 characters")
	}
	if len(t.Project_code) > 10 {
		return errors.New("project_code cannot exceed 10 characters")
	}
	if len(t.Salesperson_code) > 10 {
		return errors.New("salesperson_code cannot exceed 10 characters")
	}
	if len(t.User_id) > 20 {
		return errors.New("user_id cannot exceed 20 characters")
	}
	if len(t.Source_code) > 10 {
		return errors.New("source_code cannot exceed 10 characters")
	}
	if len(t.Reason_code) > 10 {
		return errors.New("reason_code cannot exceed 10 characters")
	}
	if len(t.Journal_batch_name) > 10 {
		return errors.New("journal_batch_name cannot exceed 10 characters")
	}
	if len(t.Applies_to_doc_no) > 20 {
		return errors.New("applies_to_doc_no cannot exceed 20 characters")
	}
	if len(t.Applies_to_id) > 20 {
		return errors.New("applies_to_id cannot exceed 20 characters")
	}
	if len(t.On_hold) > 3 {
		return errors.New("on_hold cannot exceed 3 characters")
	}
	if len(t.Bal_account_no) > 20 {
		return errors.New("bal_account_no cannot exceed 20 characters")
	}

	return nil
}

// ========================================
// Field Validation Hooks
// ========================================
// These methods are called by auto-generated OnValidate triggers in customerLedgerEntry_gen.go
// Add your custom field validation logic here

// CustomValidate_Entry_no - Custom validation for entry_no field
func (t *CustomerLedgerEntry) CustomValidate_Entry_no() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for entry_no:
	// if len(t.Entry_no) < 3 {
	//     return errors.New("entry_no must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Customer_no - Custom validation for customer_no field
func (t *CustomerLedgerEntry) CustomValidate_Customer_no() error {
	// Table relation validation - customer_no must exist in Customer
	if t.Customer_no != "" && t.Customer_no != types.Code("") {
		var relatedRecord Customer
		relatedRecord.Init(t.db, t.company)
		if !relatedRecord.Get(t.Customer_no) {
			return errors.New("customer_no does not exist in Customer table")
		}

		// *** ADD YOUR CUSTOM LOGIC HERE ***
		// You can access the related record:
		// if !relatedRecord.Active {
		//     return errors.New("Customer is inactive")
		// }
	}

	return nil
}

// CustomValidate_Sell_to_customer_no - Custom validation for sell_to_customer_no field
func (t *CustomerLedgerEntry) CustomValidate_Sell_to_customer_no() error {
	// Table relation validation - sell_to_customer_no must exist in Customer
	if t.Sell_to_customer_no != "" && t.Sell_to_customer_no != types.Code("") {
		var relatedRecord Customer
		relatedRecord.Init(t.db, t.company)
		if !relatedRecord.Get(t.Sell_to_customer_no) {
			return errors.New("sell_to_customer_no does not exist in Customer table")
		}

		// *** ADD YOUR CUSTOM LOGIC HERE ***
		// You can access the related record:
		// if !relatedRecord.Active {
		//     return errors.New("Customer is inactive")
		// }
	}

	return nil
}

// CustomValidate_Posting_date - Custom validation for posting_date field
func (t *CustomerLedgerEntry) CustomValidate_Posting_date() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for posting_date:
	// if len(t.Posting_date) < 3 {
	//     return errors.New("posting_date must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Document_date - Custom validation for document_date field
func (t *CustomerLedgerEntry) CustomValidate_Document_date() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for document_date:
	// if len(t.Document_date) < 3 {
	//     return errors.New("document_date must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Document_type - Custom validation for document_type field
func (t *CustomerLedgerEntry) CustomValidate_Document_type() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for document_type:
	// if len(t.Document_type) < 3 {
	//     return errors.New("document_type must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Document_no - Custom validation for document_no field
func (t *CustomerLedgerEntry) CustomValidate_Document_no() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for document_no:
	// if len(t.Document_no) < 3 {
	//     return errors.New("document_no must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_External_document_no - Custom validation for external_document_no field
func (t *CustomerLedgerEntry) CustomValidate_External_document_no() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for external_document_no:
	// if len(t.External_document_no) < 3 {
	//     return errors.New("external_document_no must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Description - Custom validation for description field
func (t *CustomerLedgerEntry) CustomValidate_Description() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for description:
	// if len(t.Description) < 3 {
	//     return errors.New("description must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Currency_code - Custom validation for currency_code field
func (t *CustomerLedgerEntry) CustomValidate_Currency_code() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for currency_code:
	// if len(t.Currency_code) < 3 {
	//     return errors.New("currency_code must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Amount - Custom validation for amount field
func (t *CustomerLedgerEntry) CustomValidate_Amount() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for amount:
	// if len(t.Amount) < 3 {
	//     return errors.New("amount must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Remaining_amount - Custom validation for remaining_amount field
func (t *CustomerLedgerEntry) CustomValidate_Remaining_amount() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for remaining_amount:
	// if len(t.Remaining_amount) < 3 {
	//     return errors.New("remaining_amount must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Closed_by_amount - Custom validation for closed_by_amount field
func (t *CustomerLedgerEntry) CustomValidate_Closed_by_amount() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for closed_by_amount:
	// if len(t.Closed_by_amount) < 3 {
	//     return errors.New("closed_by_amount must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Original_amount_lcy - Custom validation for original_amount_lcy field
func (t *CustomerLedgerEntry) CustomValidate_Original_amount_lcy() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for original_amount_lcy:
	// if len(t.Original_amount_lcy) < 3 {
	//     return errors.New("original_amount_lcy must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Remaining_amt_lcy - Custom validation for remaining_amt_lcy field
func (t *CustomerLedgerEntry) CustomValidate_Remaining_amt_lcy() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for remaining_amt_lcy:
	// if len(t.Remaining_amt_lcy) < 3 {
	//     return errors.New("remaining_amt_lcy must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Amount_lcy - Custom validation for amount_lcy field
func (t *CustomerLedgerEntry) CustomValidate_Amount_lcy() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for amount_lcy:
	// if len(t.Amount_lcy) < 3 {
	//     return errors.New("amount_lcy must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Closed_by_amount_lcy - Custom validation for closed_by_amount_lcy field
func (t *CustomerLedgerEntry) CustomValidate_Closed_by_amount_lcy() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for closed_by_amount_lcy:
	// if len(t.Closed_by_amount_lcy) < 3 {
	//     return errors.New("closed_by_amount_lcy must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Sales_lcy - Custom validation for sales_lcy field
func (t *CustomerLedgerEntry) CustomValidate_Sales_lcy() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for sales_lcy:
	// if len(t.Sales_lcy) < 3 {
	//     return errors.New("sales_lcy must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Profit_lcy - Custom validation for profit_lcy field
func (t *CustomerLedgerEntry) CustomValidate_Profit_lcy() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for profit_lcy:
	// if len(t.Profit_lcy) < 3 {
	//     return errors.New("profit_lcy must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Inv_discount_lcy - Custom validation for inv_discount_lcy field
func (t *CustomerLedgerEntry) CustomValidate_Inv_discount_lcy() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for inv_discount_lcy:
	// if len(t.Inv_discount_lcy) < 3 {
	//     return errors.New("inv_discount_lcy must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Pmt_discount_date - Custom validation for pmt_discount_date field
func (t *CustomerLedgerEntry) CustomValidate_Pmt_discount_date() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for pmt_discount_date:
	// if len(t.Pmt_discount_date) < 3 {
	//     return errors.New("pmt_discount_date must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Pmt_disc_possible - Custom validation for pmt_disc_possible field
func (t *CustomerLedgerEntry) CustomValidate_Pmt_disc_possible() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for pmt_disc_possible:
	// if len(t.Pmt_disc_possible) < 3 {
	//     return errors.New("pmt_disc_possible must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Pmt_disc_given_lcy - Custom validation for pmt_disc_given_lcy field
func (t *CustomerLedgerEntry) CustomValidate_Pmt_disc_given_lcy() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for pmt_disc_given_lcy:
	// if len(t.Pmt_disc_given_lcy) < 3 {
	//     return errors.New("pmt_disc_given_lcy must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Customer_posting_group - Custom validation for customer_posting_group field
func (t *CustomerLedgerEntry) CustomValidate_Customer_posting_group() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for customer_posting_group:
	// if len(t.Customer_posting_group) < 3 {
	//     return errors.New("customer_posting_group must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Department_code - Custom validation for department_code field
func (t *CustomerLedgerEntry) CustomValidate_Department_code() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for department_code:
	// if len(t.Department_code) < 3 {
	//     return errors.New("department_code must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Project_code - Custom validation for project_code field
func (t *CustomerLedgerEntry) CustomValidate_Project_code() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for project_code:
	// if len(t.Project_code) < 3 {
	//     return errors.New("project_code must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Salesperson_code - Custom validation for salesperson_code field
func (t *CustomerLedgerEntry) CustomValidate_Salesperson_code() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for salesperson_code:
	// if len(t.Salesperson_code) < 3 {
	//     return errors.New("salesperson_code must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_User_id - Custom validation for user_id field
func (t *CustomerLedgerEntry) CustomValidate_User_id() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for user_id:
	// if len(t.User_id) < 3 {
	//     return errors.New("user_id must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Source_code - Custom validation for source_code field
func (t *CustomerLedgerEntry) CustomValidate_Source_code() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for source_code:
	// if len(t.Source_code) < 3 {
	//     return errors.New("source_code must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Reason_code - Custom validation for reason_code field
func (t *CustomerLedgerEntry) CustomValidate_Reason_code() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for reason_code:
	// if len(t.Reason_code) < 3 {
	//     return errors.New("reason_code must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Journal_batch_name - Custom validation for journal_batch_name field
func (t *CustomerLedgerEntry) CustomValidate_Journal_batch_name() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for journal_batch_name:
	// if len(t.Journal_batch_name) < 3 {
	//     return errors.New("journal_batch_name must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Transaction_no - Custom validation for transaction_no field
func (t *CustomerLedgerEntry) CustomValidate_Transaction_no() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for transaction_no:
	// if len(t.Transaction_no) < 3 {
	//     return errors.New("transaction_no must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Applies_to_doc_type - Custom validation for applies_to_doc_type field
func (t *CustomerLedgerEntry) CustomValidate_Applies_to_doc_type() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for applies_to_doc_type:
	// if len(t.Applies_to_doc_type) < 3 {
	//     return errors.New("applies_to_doc_type must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Applies_to_doc_no - Custom validation for applies_to_doc_no field
func (t *CustomerLedgerEntry) CustomValidate_Applies_to_doc_no() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for applies_to_doc_no:
	// if len(t.Applies_to_doc_no) < 3 {
	//     return errors.New("applies_to_doc_no must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Applies_to_id - Custom validation for applies_to_id field
func (t *CustomerLedgerEntry) CustomValidate_Applies_to_id() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for applies_to_id:
	// if len(t.Applies_to_id) < 3 {
	//     return errors.New("applies_to_id must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Open - Custom validation for open field
func (t *CustomerLedgerEntry) CustomValidate_Open() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for open:
	// if len(t.Open) < 3 {
	//     return errors.New("open must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Positive - Custom validation for positive field
func (t *CustomerLedgerEntry) CustomValidate_Positive() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for positive:
	// if len(t.Positive) < 3 {
	//     return errors.New("positive must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_On_hold - Custom validation for on_hold field
func (t *CustomerLedgerEntry) CustomValidate_On_hold() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for on_hold:
	// if len(t.On_hold) < 3 {
	//     return errors.New("on_hold must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Due_date - Custom validation for due_date field
func (t *CustomerLedgerEntry) CustomValidate_Due_date() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for due_date:
	// if len(t.Due_date) < 3 {
	//     return errors.New("due_date must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Closed_by_entry_no - Custom validation for closed_by_entry_no field
func (t *CustomerLedgerEntry) CustomValidate_Closed_by_entry_no() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for closed_by_entry_no:
	// if len(t.Closed_by_entry_no) < 3 {
	//     return errors.New("closed_by_entry_no must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Closed_at_date - Custom validation for closed_at_date field
func (t *CustomerLedgerEntry) CustomValidate_Closed_at_date() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for closed_at_date:
	// if len(t.Closed_at_date) < 3 {
	//     return errors.New("closed_at_date must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Bal_account_type - Custom validation for bal_account_type field
func (t *CustomerLedgerEntry) CustomValidate_Bal_account_type() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for bal_account_type:
	// if len(t.Bal_account_type) < 3 {
	//     return errors.New("bal_account_type must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Bal_account_no - Custom validation for bal_account_no field
func (t *CustomerLedgerEntry) CustomValidate_Bal_account_no() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for bal_account_no:
	// if len(t.Bal_account_no) < 3 {
	//     return errors.New("bal_account_no must be at least 3 characters")
	// }

	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
// Example:
// func (t *CustomerLedgerEntry) CalculateSomething() error {
//     // Your logic here
//     return nil
// }
