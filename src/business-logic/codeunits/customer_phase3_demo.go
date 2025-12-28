package codeunits

import (
	"fmt"
	"strings"

	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/session"
	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// CustomerPhase3Demo - Codeunit 50004: Customer Phase 3 Demo
// Demonstrates IsEmpty, ModifyAll, DeleteAll, CopyFilters, GetFilters
const CustomerPhase3DemoID = 50004

type CustomerPhase3Demo struct {
	session *session.Session
}

// NewCustomerPhase3Demo creates a new instance of the codeunit
func NewCustomerPhase3Demo(s *session.Session) *CustomerPhase3Demo {
	return &CustomerPhase3Demo{
		session: s,
	}
}

// RunCLI executes the Customer Phase 3 Demo codeunit from CLI
func (c *CustomerPhase3Demo) RunCLI() error {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("CUSTOMER PHASE 3 DEMO - IsEmpty, ModifyAll, DeleteAll, etc.")
	fmt.Println(strings.Repeat("=", 60))

	// Create test data if needed
	c.createTestData()

	fmt.Println("\n--- Test 1: GetFilters() ---")
	c.testGetFilters()

	fmt.Println("\n--- Test 2: IsEmpty() ---")
	c.testIsEmpty()

	fmt.Println("\n--- Test 3: CopyFilters() ---")
	c.testCopyFilters()

	fmt.Println("\n--- Test 4: ModifyAll() ---")
	c.testModifyAll()

	fmt.Println("\n--- Test 5: DeleteAll() ---")
	c.testDeleteAll()

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("✓ Demo complete!")

	return nil
}

// createTestData creates sample customers if they don't exist
func (c *CustomerPhase3Demo) createTestData() {
	testCustomers := []struct {
		No   string
		Name string
		City string
	}{
		{"TEST01", "Test Customer 1", "Chicago"},
		{"TEST02", "Test Customer 2", "Chicago"},
		{"TEST03", "Test Customer 3", "Boston"},
		{"TEST04", "Test Customer 4", "Boston"},
		{"TEST05", "Test Customer 5", "Seattle"},
	}

	for _, tc := range testCustomers {
		var customer tables.Customer
		customer.Init(c.session.GetConnection(), c.session.GetCompany())

		// Check if exists
		if customer.Get(types.NewCode(tc.No)) {
			continue // Already exists
		}

		// Create new customer
		customer.No = types.NewCode(tc.No)
		customer.Name = types.NewText(tc.Name)
		customer.City = types.NewText(tc.City)
		customer.Address = types.NewText("123 Main St")
		customer.Post_code = types.NewCode("12345")
		customer.Phonenumber = types.NewText("")

		if customer.Insert(true) {
			fmt.Printf("  + Created test customer: %s - %s\n", tc.No, tc.Name)
		}
	}
}

// testGetFilters tests GetFilters method
func (c *CustomerPhase3Demo) testGetFilters() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	// Test 1: Range filter (2 parameters)
	customer.SetRange("no", "TEST01", "TEST03")
	customer.SetFilter("city", "Chicago|Boston")
	fmt.Printf("  Range filter: %s\n", customer.GetFilters())

	// Test 2: Exact match filter (1 parameter) - BC/NAV style
	customer.Reset()
	customer.SetRange("no", "TEST02")  // Exact match
	fmt.Printf("  Exact match filter: %s\n", customer.GetFilters())
	fmt.Printf("✓ Count with exact match: %d\n", customer.Count())
}

// testIsEmpty tests IsEmpty method
func (c *CustomerPhase3Demo) testIsEmpty() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	// Test 1: Check if any TEST customers exist
	customer.SetFilter("no", "TEST*")
	if customer.IsEmpty() {
		fmt.Println("✗ No TEST customers found")
	} else {
		count := customer.Count()
		fmt.Printf("✓ Found TEST customers (count: %d)\n", count)
	}

	// Test 2: Check for non-existent city
	customer.Reset()
	customer.SetFilter("city", "NonExistentCity")
	if customer.IsEmpty() {
		fmt.Println("✓ Correctly detected no customers in NonExistentCity")
	} else {
		fmt.Println("✗ Unexpectedly found customers")
	}
}

// testCopyFilters tests CopyFilters method
func (c *CustomerPhase3Demo) testCopyFilters() {
	var customer1 tables.Customer
	customer1.Init(c.session.GetConnection(), c.session.GetCompany())

	var customer2 tables.Customer
	customer2.Init(c.session.GetConnection(), c.session.GetCompany())

	// Set filters on customer1
	customer1.SetRange("no", "TEST01", "TEST03")
	customer1.SetFilter("city", "Chicago")
	customer1.SetCurrentKey("city", "name")

	fmt.Printf("  Customer1 filters: %s\n", customer1.GetFilters())

	// Copy filters to customer2
	customer2.CopyFilters(&customer1)
	fmt.Printf("  Customer2 filters: %s\n", customer2.GetFilters())

	// Verify both have same count
	count1 := customer1.Count()
	count2 := customer2.Count()
	fmt.Printf("✓ Customer1 count: %d, Customer2 count: %d\n", count1, count2)
}

// testModifyAll tests ModifyAll method
func (c *CustomerPhase3Demo) testModifyAll() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	// Update all Chicago customers' post_code
	customer.SetFilter("city", "Chicago")
	count := customer.Count()
	fmt.Printf("  Found %d customers in Chicago\n", count)

	rowsModified := customer.ModifyAll("post_code", "60601")
	fmt.Printf("✓ Modified %d customer post codes to 60601\n", rowsModified)

	// Verify modification
	customer.Reset()
	customer.SetFilter("post_code", "60601")
	verifyCount := customer.Count()
	fmt.Printf("  Verification: %d customers now have post_code 60601\n", verifyCount)
}

// testDeleteAll tests DeleteAll method
func (c *CustomerPhase3Demo) testDeleteAll() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	// Delete all Seattle customers
	customer.SetFilter("city", "Seattle")
	count := customer.Count()
	fmt.Printf("  Found %d customers in Seattle\n", count)

	if count > 0 {
		rowsDeleted := customer.DeleteAll()
		fmt.Printf("✓ Deleted %d customer(s) from Seattle\n", rowsDeleted)

		// Verify deletion
		customer.Reset()
		customer.SetFilter("city", "Seattle")
		verifyCount := customer.Count()
		fmt.Printf("  Verification: %d customers remain in Seattle\n", verifyCount)
	} else {
		fmt.Println("  No Seattle customers to delete")
	}
}

// RunCustomerPhase3Demo is the main entry point for running this codeunit from the application
func RunCustomerPhase3Demo() {
	// Get global session
	sess := session.GetCurrent()
	if sess == nil {
		fmt.Println("✗ Error: No active session")
		return
	}

	// Create codeunit instance
	demo := NewCustomerPhase3Demo(sess)

	// Execute codeunit
	err := demo.RunCLI()
	if err != nil {
		fmt.Printf("\n✗ Error: %v\n", err)
	}

	// Wait for user
	fmt.Print("\nPress Enter to continue...")
	sess.GetScanner().Scan()
}
