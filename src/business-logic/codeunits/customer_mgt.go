package codeunits

import (
	"fmt"
	"strings"

	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/common"
	"github.com/hansjlachmann/openerp/src/foundation/session"
	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// CustomerManagement - Codeunit 50002: Customer Management
const CustomerManagementID = 50002

type CustomerManagement struct {
	session *session.Session
}

// NewCustomerManagement creates a new instance of the codeunit
func NewCustomerManagement(s *session.Session) *CustomerManagement {
	return &CustomerManagement{
		session: s,
	}
}

// RunCLI executes the Customer Management codeunit from CLI
func (c *CustomerManagement) RunCLI() error {
	var Customer tables.Customer
	CustomerNumber := "001"

	Customer.Init(c.session.GetConnection(), c.session.GetCompany())

	if Customer.FindLast() {
		CustomerNumber = common.IncStr(string(Customer.No))
	}
	fmt.Println("\n✓ Inserting customer number ", CustomerNumber)

	Customer.No = types.NewCode(CustomerNumber)
	Customer.Name = types.NewText("Acme Corporation")
	Customer.Address = types.NewText("124 Main Street")
	Customer.Post_code = types.NewCode("12345")
	Customer.City = types.NewText("New York")
	Customer.Phonenumber = "999000999"
	Customer.Payment_terms_code = types.NewCode("30DAYS")

	// Validate Payment Terms Code exists (Table Relation validation)
	var paymentTerms tables.PaymentTerms
	paymentTerms.Init(c.session.GetConnection(), c.session.GetCompany())
	if !paymentTerms.Get(Customer.Payment_terms_code) {
		return fmt.Errorf("payment terms code '%s' does not exist in Payment Terms table", Customer.Payment_terms_code)
	}
	fmt.Printf("✓ Payment Terms Code validated: %s - %s\n", paymentTerms.Code, paymentTerms.Description)

	if !Customer.Insert(true) { // Run triggers
		return fmt.Errorf("failed to insert customer record")
	}
	fmt.Println("\n✓ Customer record inserted successfully!")

	// Verify by reading it back using table's Get() method
	fmt.Println("\nVerifying inserted customer...")
	var record tables.Customer
	record.Init(c.session.GetConnection(), c.session.GetCompany())
	if !record.Get(types.Code(CustomerNumber)) {
		return fmt.Errorf("failed to retrieve customer record")
	}

	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Printf("No.:                 %s\n", record.No)
	fmt.Printf("Name:                %s\n", record.Name)
	fmt.Printf("Address:             %s\n", record.Address)
	fmt.Printf("Post Code:           %s\n", record.Post_code)
	fmt.Printf("City:                %s\n", record.City)
	fmt.Printf("Phone:               %s\n", record.Phonenumber)
	fmt.Printf("Payment Terms Code:  %s\n", record.Payment_terms_code)
	fmt.Println(strings.Repeat("-", 60))

	fmt.Println("\n✓ Codeunit execution complete!")

	return nil
}

// RunCustomerMgt is the main entry point for running this codeunit from the application
// Gets global session, creates codeunit, executes, and waits for user
func RunCustomerMgt() {
	// Get global session
	sess := session.GetCurrent()
	if sess == nil {
		fmt.Println("✗ Error: No active session")
		return
	}

	// Create codeunit instance
	custMgt := NewCustomerManagement(sess)

	// Execute codeunit
	err := custMgt.RunCLI()
	if err != nil {
		fmt.Printf("\n✗ Error: %v\n", err)
	}

	// Wait for user
	fmt.Print("\nPress Enter to continue...")
	sess.GetScanner().Scan()
}
