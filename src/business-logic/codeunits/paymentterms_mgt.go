package codeunits

import (
	"fmt"
	"strings"

	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/session"
	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// PaymentTermsManagement - Codeunit 50000: Payment Terms Management
const PaymentTermsManagementID = 50000

type PaymentTermsManagement struct {
	session *session.Session
}

// NewPaymentTermsManagement creates a new instance of the codeunit
func NewPaymentTermsManagement(s *session.Session) *PaymentTermsManagement {
	return &PaymentTermsManagement{
		session: s,
	}
}

// RunCLI executes the Payment Terms Management codeunit from CLI
func (c *PaymentTermsManagement) RunCLI() error {
	var PaymentTerms tables.PaymentTerms
	PaymentTerms.Init(c.session.GetConnection(), c.session.GetCompany())

	// Set fields directly (like BC/NAV)
	PaymentTerms.Code = types.NewCode("30DAYS")
	//PaymentTerms.Description = types.NewText("Test Payment Terms")
	PaymentTerms.Description = "30 Days"
	PaymentTerms.Active = true
	if !PaymentTerms.Insert(true) { // Run triggers
		return fmt.Errorf("failed to insert test record")
	}
	fmt.Println("\n✓ Test record inserted successfully!")

	// Verify by reading it back using table's Get() method
	fmt.Println("\nVerifying inserted record...")
	var record tables.PaymentTerms
	record.Init(c.session.GetConnection(), c.session.GetCompany())
	if !record.Get(types.NewCode("TEST")) {
		return fmt.Errorf("failed to retrieve record")
	}

	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Printf("Code:        %s\n", record.Code)
	fmt.Printf("Description: %s\n", record.Description)
	fmt.Printf("Active:      %v\n", record.Active)
	fmt.Println(strings.Repeat("-", 60))

	fmt.Println("\n✓ Codeunit execution complete!")

	return nil
}

// RunPaymentTermsMgt is the main entry point for running this codeunit from the application
// Gets global session, creates codeunit, executes, and waits for user
func RunPaymentTermsMgt() {
	// Get global session
	sess := session.GetCurrent()
	if sess == nil {
		fmt.Println("✗ Error: No active session")
		return
	}

	// Create codeunit instance
	ptMgt := NewPaymentTermsManagement(sess)

	// Execute codeunit
	err := ptMgt.RunCLI()
	if err != nil {
		fmt.Printf("\n✗ Error: %v\n", err)
	}

	// Wait for user
	fmt.Print("\nPress Enter to continue...")
	sess.GetScanner().Scan()
}
