package codeunits

import (
	"fmt"
	"strings"

	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/session"
)

// PaymentTermsInsert10 - Codeunit 50001: Insert 10 Payment Terms Records
// Inserts 10 test records with sequential codes (TEST001, TEST002, etc.)
const PaymentTermsInsert10ID = 50001

type PaymentTermsInsert10 struct {
	session *session.Session
}

// NewPaymentTermsInsert10 creates a new instance of the codeunit
func NewPaymentTermsInsert10(s *session.Session) *PaymentTermsInsert10 {
	return &PaymentTermsInsert10{
		session: s,
	}
}

// RunCLI executes the codeunit from CLI
func (c *PaymentTermsInsert10) RunCLI() error {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("CODEUNIT 50001: Test is Empty")

	var Customer tables.Customer
	Customer.Init(c.session.GetConnection(), c.session.GetCompany())
	Customer.SetRange(string(Customer.No), "100", "101")
	fmt.Printf("Customer Isempty", Customer.IsEmpty())

	/*
		for i := 1; i <= 20; i++ {
			var PaymentTerms tables.PaymentTerms
			PaymentTerms.Init(c.session.GetConnection(), c.session.GetCompany())
			code := fmt.Sprintf("TEST%03d", i) // %03d formats as 001, 002, 003...
			recordExists := PaymentTerms.Get(types.NewCode(code))
			PaymentTerms.Code = types.NewCode(code)
			PaymentTerms.Description = types.NewText(fmt.Sprintf("Test Payment Terms %d", i))
			PaymentTerms.Active = false
			if !recordExists {
				if PaymentTerms.Insert(true) {
					fmt.Printf("  %d. %s - ✓ Inserted successfully\n", i, code)
				}
			} else {
				if PaymentTerms.Modify(true) {
					fmt.Printf("  %d. %s - ✓ Modified successfully\n", i, code)
				}
			}
		}
	*/

	fmt.Println("\n✓ Codeunit execution complete!")

	return nil
}

// RunPaymentTermsInsert10 is the main entry point for running this codeunit from the application
// Gets global session, creates codeunit, executes, and waits for user
func RunPaymentTermsInsert10() {
	// Get global session
	sess := session.GetCurrent()
	if sess == nil {
		fmt.Println("✗ Error: No active session")
		return
	}

	// Create codeunit instance
	ptInsert10 := NewPaymentTermsInsert10(sess)

	// Execute codeunit
	err := ptInsert10.RunCLI()
	if err != nil {
		fmt.Printf("\n✗ Error: %v\n", err)
	}

	// Wait for user
	fmt.Print("\nPress Enter to continue...")
	sess.GetScanner().Scan()
}
