package tables

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// PaymentTerms represents Table 3: Payment Terms
// Based on Microsoft Dynamics Business Central Payment Terms table
type PaymentTerms struct {
	code                 types.Code `db:"code,pk"`
	description          types.Text `db:"description"`
	dueDateCalculation   string     `db:"due_date_calculation"`
	discountDateCalc     string     `db:"discount_date_calculation"`
	discountPct          float64    `db:"discount_pct"`
	calcPmtDiscOnCrMemo  bool       `db:"calc_pmt_disc_on_cr_memos"`
	lastModifiedDateTime time.Time  `db:"last_modified_date_time"`
}

const PaymentTermsTableID = 3
const PaymentTermsTableName = "Payment Terms"

// ========================================
// TableDefinition Interface Implementation
// ========================================

// GetTableID returns the table ID (for Object Registry)
func (p *PaymentTerms) GetTableID() int {
	return PaymentTermsTableID
}

// GetTableName returns the table name
func (p *PaymentTerms) GetTableName() string {
	return PaymentTermsTableName
}

// GetTableSchema returns the CREATE TABLE schema
func (p *PaymentTerms) GetTableSchema() string {
	return GetPaymentTermsTableSchema()
}

// NewPaymentTerms creates a new PaymentTerms instance
func NewPaymentTerms() *PaymentTerms {
	return &PaymentTerms{
		lastModifiedDateTime: time.Now(),
	}
}

// ========================================
// Getters
// ========================================

// Code returns the payment terms code
func (p *PaymentTerms) Code() types.Code {
	return p.code
}

// Description returns the payment terms description
func (p *PaymentTerms) Description() types.Text {
	return p.description
}

// DueDateCalculation returns the due date calculation formula
func (p *PaymentTerms) DueDateCalculation() string {
	return p.dueDateCalculation
}

// DiscountDateCalc returns the discount date calculation formula
func (p *PaymentTerms) DiscountDateCalc() string {
	return p.discountDateCalc
}

// DiscountPct returns the discount percentage
func (p *PaymentTerms) DiscountPct() float64 {
	return p.discountPct
}

// CalcPmtDiscOnCrMemo returns whether to calculate payment discount on credit memos
func (p *PaymentTerms) CalcPmtDiscOnCrMemo() bool {
	return p.calcPmtDiscOnCrMemo
}

// LastModifiedDateTime returns the last modified timestamp
func (p *PaymentTerms) LastModifiedDateTime() time.Time {
	return p.lastModifiedDateTime
}

// ========================================
// Setters with Validation (Field Triggers)
// ========================================

// SetCode sets the payment terms code with validation
func (p *PaymentTerms) SetCode(code types.Code) error {
	if code.IsEmpty() {
		return errors.New("code cannot be empty")
	}

	if len(code) > 10 {
		return errors.New("code cannot exceed 10 characters")
	}

	p.code = code
	return nil
}

// SetDescription sets the description
func (p *PaymentTerms) SetDescription(description types.Text) error {
	if len(description) > 100 {
		return errors.New("description cannot exceed 100 characters")
	}

	p.description = description
	return nil
}

// SetDueDateCalculation sets the due date calculation formula
func (p *PaymentTerms) SetDueDateCalculation(formula string) error {
	// TODO: Validate date formula format
	p.dueDateCalculation = formula
	return nil
}

// SetDiscountDateCalc sets the discount date calculation formula
func (p *PaymentTerms) SetDiscountDateCalc(formula string) error {
	// TODO: Validate date formula format
	p.discountDateCalc = formula
	return nil
}

// SetDiscountPct sets the discount percentage with validation
func (p *PaymentTerms) SetDiscountPct(pct float64) error {
	if pct < 0 || pct > 100 {
		return errors.New("discount % must be between 0 and 100")
	}

	p.discountPct = pct
	return nil
}

// SetCalcPmtDiscOnCrMemo sets whether to calculate payment discount on credit memos
func (p *PaymentTerms) SetCalcPmtDiscOnCrMemo(value bool) {
	p.calcPmtDiscOnCrMemo = value
}

// ========================================
// Table Triggers (OnInsert, OnModify, OnDelete)
// ========================================

// OnInsert trigger - called before inserting a new record
func (p *PaymentTerms) OnInsert() error {
	p.SetLastModified()
	return p.Validate()
}

// OnModify trigger - called before modifying a record
func (p *PaymentTerms) OnModify() error {
	p.SetLastModified()
	return p.Validate()
}

// OnDelete trigger - called before deleting a record
func (p *PaymentTerms) OnDelete(db *sql.DB, company string) error {
	// Check if payment terms is used in other tables
	// For now, just allow deletion
	// TODO: Add checks for related records (sales orders, customers, etc.)

	// Example check (to be implemented when we have customer table):
	// var count int
	// err := db.QueryRow(
	//     fmt.Sprintf(`SELECT COUNT(*) FROM "%s$Customer" WHERE payment_terms_code = $1`, company),
	//     p.code,
	// ).Scan(&count)
	// if err != nil {
	//     return err
	// }
	// if count > 0 {
	//     return fmt.Errorf("cannot delete: payment terms %s is used by %d customers", p.code, count)
	// }

	// Delete related translations (when implemented)
	// _, err = db.Exec(
	//     fmt.Sprintf(`DELETE FROM "%s$Payment Term Translation" WHERE payment_term = $1`, company),
	//     p.code,
	// )

	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (p *PaymentTerms) OnRename() error {
	p.SetLastModified()
	// TODO: Update related records (CRM sync, etc.)
	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// SetLastModified updates the last modified timestamp
func (p *PaymentTerms) SetLastModified() {
	p.lastModifiedDateTime = time.Now()
}

// Validate validates all fields
func (p *PaymentTerms) Validate() error {
	if p.code.IsEmpty() {
		return errors.New("code is required")
	}

	if len(p.code) > 10 {
		return errors.New("code cannot exceed 10 characters")
	}

	if len(p.description) > 100 {
		return errors.New("description cannot exceed 100 characters")
	}

	if p.discountPct < 0 || p.discountPct > 100 {
		return errors.New("discount % must be between 0 and 100")
	}

	return nil
}

// UsePaymentDiscount checks if any payment terms use discount
func UsePaymentDiscount(db *sql.DB, company string) (bool, error) {
	var count int
	err := db.QueryRow(
		fmt.Sprintf(`SELECT COUNT(*) FROM "%s$%s" WHERE discount_pct > 0`, company, PaymentTermsTableName),
	).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetDescriptionInCurrentLanguage returns the description in the user's language
// For now, just returns the default description
// TODO: Implement multi-language support with translation table
func (p *PaymentTerms) GetDescriptionInCurrentLanguage() types.Text {
	return p.description
}

// ========================================
// Database Schema
// ========================================

// GetTableSchema returns the CREATE TABLE SQL for PostgreSQL
func GetPaymentTermsTableSchema() string {
	return `
		code VARCHAR(10) PRIMARY KEY,
		description VARCHAR(100),
		due_date_calculation VARCHAR(50),
		discount_date_calculation VARCHAR(50),
		discount_pct DECIMAL(5,2) CHECK (discount_pct >= 0 AND discount_pct <= 100),
		calc_pmt_disc_on_cr_memos BOOLEAN DEFAULT FALSE,
		last_modified_date_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	`
}

// CreateTable creates the Payment Terms table for the current company
func CreateTable(db *sql.DB, company string) error {
	tableName := fmt.Sprintf("%s$%s", company, PaymentTermsTableName)
	schema := GetPaymentTermsTableSchema()

	createSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (%s)`, tableName, schema)
	_, err := db.Exec(createSQL)
	if err != nil {
		return fmt.Errorf("failed to create Payment Terms table: %w", err)
	}

	return nil
}
