package codeunits

import (
	"fmt"
	"strings"

	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/session"
	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// OnValidateDemo - Codeunit 50006: OnValidate Triggers Demo
// Demonstrates ValidateField() and OnValidate triggers with automatic table relation validation
const OnValidateDemoID = 50006

type OnValidateDemo struct {
	session *session.Session
}

// NewOnValidateDemo creates a new instance of the codeunit
func NewOnValidateDemo(s *session.Session) *OnValidateDemo {
	return &OnValidateDemo{
		session: s,
	}
}

// RunCLI executes the OnValidate Demo codeunit from CLI
func (c *OnValidateDemo) RunCLI() error {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("ONVALIDATE TRIGGERS DEMO - BC/NAV Style Field Validation")
	fmt.Println(strings.Repeat("=", 60))

	// Ensure we have a payment term to test with
	c.ensurePaymentTermExists()

	fmt.Println("\n--- Test 1: ValidateField with Valid Table Relation ---")
	c.testValidTableRelation()

	fmt.Println("\n--- Test 2: ValidateField with Invalid Table Relation ---")
	c.testInvalidTableRelation()

	fmt.Println("\n--- Test 3: Direct OnValidate Call ---")
	c.testDirectOnValidate()

	fmt.Println("\n--- Test 4: ValidateField Type Conversion ---")
	c.testTypeConversion()

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("✓ Demo complete!")

	return nil
}

// ensurePaymentTermExists creates a test payment term if it doesn't exist
func (c *OnValidateDemo) ensurePaymentTermExists() {
	var paymentTerms tables.PaymentTerms
	paymentTerms.Init(c.session.GetConnection(), c.session.GetCompany())

	// Check if 30DAYS exists
	if !paymentTerms.Get(types.NewCode("30DAYS")) {
		// Create it
		paymentTerms.Code = types.NewCode("30DAYS")
		paymentTerms.Description = types.NewText("Payment within 30 days")
		paymentTerms.Active = true

		if paymentTerms.Insert(true) {
			fmt.Println("  + Created test Payment Term: 30DAYS")
		}
	}
}

// testValidTableRelation demonstrates successful table relation validation
func (c *OnValidateDemo) testValidTableRelation() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	// Use ValidateField to set Payment Terms Code with validation
	err := customer.ValidateField("payment_terms_code", "30DAYS")
	if err != nil {
		fmt.Printf("✗ Validation failed: %v\n", err)
	} else {
		fmt.Println("✓ Payment Terms Code '30DAYS' validated successfully!")
		fmt.Printf("  Field value: %s\n", customer.Payment_terms_code)
	}
}

// testInvalidTableRelation demonstrates failed table relation validation
func (c *OnValidateDemo) testInvalidTableRelation() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	// Try to validate with non-existent payment term
	err := customer.ValidateField("payment_terms_code", "INVALID")
	if err != nil {
		fmt.Printf("✓ Validation correctly rejected invalid code!\n")
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Println("✗ Validation should have failed!")
	}
}

// testDirectOnValidate demonstrates calling OnValidate directly
func (c *OnValidateDemo) testDirectOnValidate() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	// Set the field directly (bypassing validation)
	customer.Payment_terms_code = types.NewCode("30DAYS")
	fmt.Printf("  Set Payment_terms_code directly to: %s\n", customer.Payment_terms_code)

	// Now call OnValidate manually
	err := customer.OnValidate_Payment_terms_code()
	if err != nil {
		fmt.Printf("✗ OnValidate failed: %v\n", err)
	} else {
		fmt.Println("✓ OnValidate_Payment_terms_code() passed!")
	}

	// Try with invalid code
	customer.Payment_terms_code = types.NewCode("BADCODE")
	fmt.Printf("\n  Set Payment_terms_code directly to: %s\n", customer.Payment_terms_code)
	err = customer.OnValidate_Payment_terms_code()
	if err != nil {
		fmt.Printf("✓ OnValidate correctly detected invalid code!\n")
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Println("✗ OnValidate should have failed!")
	}
}

// testTypeConversion demonstrates automatic type conversion in ValidateField
func (c *OnValidateDemo) testTypeConversion() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	// ValidateField accepts string and converts to types.Code
	err := customer.ValidateField("no", "CUST-001")
	if err != nil {
		fmt.Printf("✗ Validation failed: %v\n", err)
	} else {
		fmt.Println("✓ ValidateField auto-converted string to types.Code")
		fmt.Printf("  Field type: types.Code, Value: %s\n", customer.No)
	}

	// Also works with types.Code directly
	err = customer.ValidateField("no", types.NewCode("CUST-002"))
	if err != nil {
		fmt.Printf("✗ Validation failed: %v\n", err)
	} else {
		fmt.Println("✓ ValidateField also accepts types.Code directly")
		fmt.Printf("  Field value: %s\n", customer.No)
	}
}

// RunOnValidateDemo is the main entry point for running this codeunit from the application
func RunOnValidateDemo() {
	// Get global session
	sess := session.GetCurrent()
	if sess == nil {
		fmt.Println("✗ Error: No active session")
		return
	}

	// Create codeunit instance
	demo := NewOnValidateDemo(sess)

	// Execute codeunit
	err := demo.RunCLI()
	if err != nil {
		fmt.Printf("\n✗ Error: %v\n", err)
	}

	// Wait for user
	fmt.Print("\nPress Enter to continue...")
	sess.GetScanner().Scan()
}
