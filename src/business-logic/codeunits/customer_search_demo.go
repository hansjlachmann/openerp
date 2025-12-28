package codeunits

import (
	"fmt"
	"strings"

	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/session"
	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// CustomerSearchDemo - Codeunit 50003: Customer Search Demo
// Demonstrates SetRange, FindFirst, FindLast functionality
const CustomerSearchDemoID = 50003

type CustomerSearchDemo struct {
	session *session.Session
}

// NewCustomerSearchDemo creates a new instance of the codeunit
func NewCustomerSearchDemo(s *session.Session) *CustomerSearchDemo {
	return &CustomerSearchDemo{
		session: s,
	}
}

// RunCLI executes the Customer Search Demo codeunit from CLI
func (c *CustomerSearchDemo) RunCLI() error {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("CUSTOMER SEARCH DEMO - SetRange, FindFirst, FindLast")
	fmt.Println(strings.Repeat("=", 60))

	// Create test data if needed
	c.createTestData()

	fmt.Println("\n--- Test 1: FindFirst (no filters) ---")
	c.testFindFirst()

	fmt.Println("\n--- Test 2: FindLast (no filters) ---")
	c.testFindLast()

	fmt.Println("\n--- Test 3: SetRange + FindFirst ---")
	c.testRangeFirst()

	fmt.Println("\n--- Test 4: SetRange + FindLast ---")
	c.testRangeLast()

	fmt.Println("\n--- Test 5: Count (no filters) ---")
	c.testCountAll()

	fmt.Println("\n--- Test 6: SetRange + Count ---")
	c.testCountRange()

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("PHASE 2: FindSet, Next, SetFilter, SetCurrentKey")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\n--- Test 7: FindSet + Next (iterate all) ---")
	c.testFindSetAll()

	fmt.Println("\n--- Test 8: SetRange + FindSet + Next ---")
	c.testRangeFindSet()

	fmt.Println("\n--- Test 9: SetFilter (BC/NAV expression) ---")
	c.testSetFilter()

	fmt.Println("\n--- Test 10: SetCurrentKey (custom sort) ---")
	c.testSetCurrentKey()

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("✓ Demo complete!")

	return nil
}

// createTestData creates sample customers if they don't exist
func (c *CustomerSearchDemo) createTestData() {
	testCustomers := []struct {
		No   string
		Name string
		City string
	}{
		{"001", "Alpha Corp", "New York"},
		{"002", "Beta Industries", "Los Angeles"},
		{"003", "Gamma LLC", "Chicago"},
		{"004", "Delta Systems", "Houston"},
		{"005", "Epsilon Tech", "Phoenix"},
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

// testFindFirst tests FindFirst without filters
func (c *CustomerSearchDemo) testFindFirst() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	if customer.FindFirst() {
		fmt.Printf("✓ Found first customer:\n")
		fmt.Printf("  No:   %s\n", customer.No)
		fmt.Printf("  Name: %s\n", customer.Name)
		fmt.Printf("  City: %s\n", customer.City)
	} else {
		fmt.Println("✗ No customers found")
	}
}

// testFindLast tests FindLast without filters
func (c *CustomerSearchDemo) testFindLast() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	if customer.FindLast() {
		fmt.Printf("✓ Found last customer:\n")
		fmt.Printf("  No:   %s\n", customer.No)
		fmt.Printf("  Name: %s\n", customer.Name)
		fmt.Printf("  City: %s\n", customer.City)
	} else {
		fmt.Println("✗ No customers found")
	}
}

// testRangeFirst tests SetRange + FindFirst
func (c *CustomerSearchDemo) testRangeFirst() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	// Set range: customers with No between "002" and "004"
	customer.SetRange("no", "002", "004")

	if customer.FindFirst() {
		fmt.Printf("✓ Found first customer in range [002..004]:\n")
		fmt.Printf("  No:   %s\n", customer.No)
		fmt.Printf("  Name: %s\n", customer.Name)
		fmt.Printf("  City: %s\n", customer.City)
	} else {
		fmt.Println("✗ No customers found in range")
	}
}

// testRangeLast tests SetRange + FindLast
func (c *CustomerSearchDemo) testRangeLast() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	// Set range: customers with No between "002" and "004"
	customer.SetRange("no", "002", "004")

	if customer.FindLast() {
		fmt.Printf("✓ Found last customer in range [002..004]:\n")
		fmt.Printf("  No:   %s\n", customer.No)
		fmt.Printf("  Name: %s\n", customer.Name)
		fmt.Printf("  City: %s\n", customer.City)
	} else {
		fmt.Println("✗ No customers found in range")
	}
}

// testCountAll tests Count without filters
func (c *CustomerSearchDemo) testCountAll() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	count := customer.Count()
	fmt.Printf("✓ Total customers in database: %d\n", count)
}

// testCountRange tests SetRange + Count
func (c *CustomerSearchDemo) testCountRange() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())
	customer.SetRange("no", "002", "004")
	fmt.Printf("✓ Customers in range [002..004]: %d\n", customer.Count())
}

// testFindSetAll tests FindSet + Next iteration through all records
func (c *CustomerSearchDemo) testFindSetAll() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	fmt.Println("✓ Iterating through all customers:")
	if customer.FindSet() {
		count := 0
		for {
			count++
			fmt.Printf("  %d. No: %s, Name: %s, City: %s\n", count, customer.No, customer.Name, customer.City)
			if !customer.Next() {
				break
			}
		}
		fmt.Printf("✓ Total records iterated: %d\n", count)
	} else {
		fmt.Println("✗ No customers found")
	}
}

// testRangeFindSet tests SetRange + FindSet + Next
func (c *CustomerSearchDemo) testRangeFindSet() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	// Set range: customers with No between "002" and "004"
	customer.SetRange("no", "002", "004")

	fmt.Println("✓ Iterating through customers in range [002..004]:")
	if customer.FindSet() {
		count := 0
		for {
			count++
			fmt.Printf("  %d. No: %s, Name: %s\n", count, customer.No, customer.Name)
			if !customer.Next() {
				break
			}
		}
		fmt.Printf("✓ Found %d customers in range\n", count)
	} else {
		fmt.Println("✗ No customers found in range")
	}
}

// testSetFilter tests SetFilter with BC/NAV filter expression
func (c *CustomerSearchDemo) testSetFilter() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	// Filter: customers with No = "001" OR "003" OR "005"
	// BC/NAV expression: "001|003|005"
	customer.SetFilter("no", "001|003|005")

	fmt.Println("✓ Customers matching filter '001|003|005':")
	if customer.FindSet() {
		count := 0
		for {
			count++
			fmt.Printf("  %d. No: %s, Name: %s\n", count, customer.No, customer.Name)
			if !customer.Next() {
				break
			}
		}
		fmt.Printf("✓ Found %d customers\n", count)
	} else {
		fmt.Println("✗ No customers found")
	}
}

// testSetCurrentKey tests SetCurrentKey for custom sorting
func (c *CustomerSearchDemo) testSetCurrentKey() {
	var customer tables.Customer
	customer.Init(c.session.GetConnection(), c.session.GetCompany())

	// Sort by city instead of default (no)
	customer.SetCurrentKey("city", "name")

	fmt.Println("✓ Customers sorted by City, Name:")
	if customer.FindSet() {
		count := 0
		for {
			count++
			fmt.Printf("  %d. City: %s, Name: %s, No: %s\n", count, customer.City, customer.Name, customer.No)
			if !customer.Next() {
				break
			}
		}
		fmt.Printf("✓ Total records: %d\n", count)
	} else {
		fmt.Println("✗ No customers found")
	}
}

// RunCustomerSearchDemo is the main entry point for running this codeunit from the application
func RunCustomerSearchDemo() {
	// Get global session
	sess := session.GetCurrent()
	if sess == nil {
		fmt.Println("✗ Error: No active session")
		return
	}

	// Create codeunit instance
	demo := NewCustomerSearchDemo(sess)

	// Execute codeunit
	err := demo.RunCLI()
	if err != nil {
		fmt.Printf("\n✗ Error: %v\n", err)
	}

	// Wait for user
	fmt.Print("\nPress Enter to continue...")
	sess.GetScanner().Scan()
}
