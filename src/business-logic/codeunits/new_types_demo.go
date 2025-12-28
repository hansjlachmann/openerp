package codeunits

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// NewTypesDemo demonstrates the new data types (Decimal, Date, DateTime, BLOB)
func NewTypesDemo(db *sql.DB, company string) {
	fmt.Println("\n========================================")
	fmt.Println("New Data Types Demo")
	fmt.Println("========================================")

	// Create a test customer with new fields
	customer := tables.NewCustomer()
	customer.Init(db, company)

	// Set basic fields
	customer.No = types.NewCode("TEST-DEC-001")
	customer.Name = types.NewText("Decimal Test Customer")
	customer.Address = types.NewText("123 Finance Street")
	customer.City = types.NewText("MoneyVille")

	// Test Decimal field
	fmt.Println("\n--- Testing Decimal Field ---")
	customer.Credit_limit = types.NewDecimal(50000.50)
	fmt.Printf("Credit Limit (from float64): %s\n", customer.Credit_limit.StringFixed(2))

	creditFromString, _ := types.NewDecimalFromString("75250.75")
	customer.Credit_limit = creditFromString
	fmt.Printf("Credit Limit (from string):  %s\n", customer.Credit_limit.StringFixed(2))

	// Test decimal arithmetic
	discount := types.NewDecimal(0.15) // 15% discount
	discountedLimit := customer.Credit_limit.Mul(types.NewDecimal(1.0).Sub(discount))
	fmt.Printf("After 15%% discount:          %s\n", discountedLimit.StringFixed(2))

	// Test Date field
	fmt.Println("\n--- Testing Date Field ---")
	customer.Last_order_date = types.Today()
	fmt.Printf("Last Order Date (today):     %s\n", customer.Last_order_date.String())

	// Add 30 days
	futureDate := customer.Last_order_date.AddDays(30)
	fmt.Printf("30 days from today:          %s\n", futureDate.String())

	// Create from string
	specificDate, _ := types.NewDateFromString("2024-06-15")
	customer.Last_order_date = specificDate
	fmt.Printf("Specific date:               %s\n", customer.Last_order_date.String())

	// Test DateTime field
	fmt.Println("\n--- Testing DateTime Field ---")
	customer.Created_at = types.Now()
	fmt.Printf("Created at (now):            %s\n", customer.Created_at.String())

	// Add 2 hours
	twoHoursLater := customer.Created_at.AddHours(2)
	fmt.Printf("2 hours later:               %s\n", twoHoursLater.String())

	// Create from time.Time
	specificTime := time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC)
	customer.Created_at = types.NewDateTimeFromTime(specificTime)
	fmt.Printf("Specific datetime:           %s\n", customer.Created_at.String())

	// Test BLOB field
	fmt.Println("\n--- Testing BLOB Field ---")
	sampleImage := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	customer.Profile_photo = sampleImage
	fmt.Printf("BLOB size:                   %d bytes\n", len(customer.Profile_photo))
	fmt.Printf("BLOB header (hex):           % X\n", customer.Profile_photo[:8])

	// Insert customer
	fmt.Println("\n--- Database Operations ---")
	if customer.Insert(true) {
		fmt.Println("✓ Customer inserted successfully")

		// Read it back
		customer2 := tables.NewCustomer()
		customer2.Init(db, company)
		if customer2.Get(types.NewCode("TEST-DEC-001")) {
			fmt.Println("✓ Customer retrieved successfully")
			fmt.Printf("  Credit Limit:     %s\n", customer2.Credit_limit.StringFixed(2))
			fmt.Printf("  Last Order Date:  %s\n", customer2.Last_order_date.String())
			fmt.Printf("  Created At:       %s\n", customer2.Created_at.String())
			fmt.Printf("  Photo BLOB size:  %d bytes\n", len(customer2.Profile_photo))

			// Test ValidateField with new types
			fmt.Println("\n--- Testing ValidateField ---")

			// Test Decimal validation
			err := customer2.ValidateField("credit_limit", types.NewDecimal(100000.00))
			if err == nil {
				fmt.Printf("✓ Decimal ValidateField: New credit limit = %s\n", customer2.Credit_limit.StringFixed(2))
			} else {
				fmt.Printf("✗ Decimal ValidateField failed: %v\n", err)
			}

			// Test Date validation (from string)
			err = customer2.ValidateField("last_order_date", "2025-01-15")
			if err == nil {
				fmt.Printf("✓ Date ValidateField:    New date = %s\n", customer2.Last_order_date.String())
			} else {
				fmt.Printf("✗ Date ValidateField failed: %v\n", err)
			}

			// Test DateTime validation (from time.Time)
			newTime := time.Date(2025, 6, 1, 14, 30, 0, 0, time.UTC)
			err = customer2.ValidateField("created_at", newTime)
			if err == nil {
				fmt.Printf("✓ DateTime ValidateField: New datetime = %s\n", customer2.Created_at.String())
			} else {
				fmt.Printf("✗ DateTime ValidateField failed: %v\n", err)
			}

			// Test Modify with new types
			fmt.Println("\n--- Testing Modify ---")
			customer2.Credit_limit = types.NewDecimal(125000.00)
			customer2.Last_order_date = types.Today().AddDays(7)
			if customer2.Modify(true) {
				fmt.Println("✓ Customer modified successfully")
			} else {
				fmt.Println("✗ Failed to modify customer")
			}

			// Clean up
			fmt.Println("\n--- Cleanup ---")
			if customer2.Delete(true) {
				fmt.Println("✓ Test customer deleted")
			}
		} else {
			fmt.Println("✗ Failed to retrieve customer")
		}
	} else {
		fmt.Println("✗ Failed to insert customer")
	}

	fmt.Println("\n========================================")
	fmt.Println("Demo Complete!")
	fmt.Println("========================================\n")
}
