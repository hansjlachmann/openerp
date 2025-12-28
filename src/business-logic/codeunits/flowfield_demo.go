package codeunits

import (
	"database/sql"
	"fmt"

	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// FlowFieldDemo demonstrates FlowField calculations
func FlowFieldDemo(db *sql.DB, company string) {
	fmt.Println("\n========================================")
	fmt.Println("FlowField Demo")
	fmt.Println("========================================")

	// First, run the Customer Ledger Entry demo to ensure we have data
	fmt.Println("\n--- Setting up test data ---")
	CustLedgerEntryDemo(db, company)

	// Now test FlowFields on each customer
	fmt.Println("\n========================================")
	fmt.Println("Testing FlowFields")
	fmt.Println("========================================")

	testCustomers := []string{"CUST-001", "CUST-002", "CUST-003"}

	for _, custNo := range testCustomers {
		customer := tables.NewCustomer()
		customer.Init(db, company)

		if customer.Get(types.NewCode(custNo)) {
			fmt.Printf("\n--- Customer: %s - %s ---\n", custNo, customer.Name.String())

			// Calculate all FlowFields
			customer.CalcFields()

			// Display the calculated values
			fmt.Printf("  Balance (LCY):        %s\n", customer.Balance_lcy.StringFixed(2))
			fmt.Printf("  Sales (LCY):          %s\n", customer.Sales_lcy.StringFixed(2))
			fmt.Printf("  No. of Entries:       %d\n", customer.No_of_ledger_entries)

			// Also demonstrate calculating specific fields only
			customer2 := tables.NewCustomer()
			customer2.Init(db, company)
			customer2.Get(types.NewCode(custNo))
			customer2.CalcFields("balance_lcy") // Calculate only balance
			fmt.Printf("\n  Specific field calculation:")
			fmt.Printf("\n    Balance only:       %s\n", customer2.Balance_lcy.StringFixed(2))

		} else {
			fmt.Printf("\n✗ Customer %s not found\n", custNo)
		}
	}

	// Show detailed breakdown for one customer
	fmt.Println("\n========================================")
	fmt.Println("Detailed Breakdown for CUST-001")
	fmt.Println("========================================")
	showDetailedCustomerBreakdown(db, company, "CUST-001")

	fmt.Println("\n========================================")
	fmt.Println("FlowField Demo Complete!")
	fmt.Println("========================================\n")
}

// showDetailedCustomerBreakdown shows how FlowFields relate to actual ledger entries
func showDetailedCustomerBreakdown(db *sql.DB, company, customerNo string) {
	// Get customer with FlowFields
	customer := tables.NewCustomer()
	customer.Init(db, company)
	customer.Get(types.NewCode(customerNo))
	customer.CalcFields()

	fmt.Printf("\nCustomer: %s - %s\n", customerNo, customer.Name.String())
	fmt.Printf("Credit Limit: %s\n\n", customer.Credit_limit.StringFixed(2))

	// Get all ledger entries
	entry := tables.NewCustomerLedgerEntry()
	entry.Init(db, company)
	entry.SetRange("customer_no", types.NewCode(customerNo), types.NewCode(customerNo))

	fmt.Println("Ledger Entries:")
	fmt.Printf("%-15s %-20s %-20s %12s %12s %8s\n",
		"Type", "Doc. No.", "Description", "Amount", "Remaining", "Open")
	fmt.Println("------------------------------------------------------------------------------------")

	totalAmount := types.ZeroDecimal()
	totalRemaining := types.ZeroDecimal()
	totalSales := types.ZeroDecimal()
	count := 0

	if entry.FindSet() {
		for {
			openStr := " "
			if entry.Open {
				openStr = "✓"
			}

			fmt.Printf("%-15s %-20s %-20s %12s %12s %8s\n",
				entry.Document_type.String(),
				entry.Document_no.String(),
				entry.Description.String(),
				entry.Amount_lcy.StringFixed(2),
				entry.Remaining_amt_lcy.StringFixed(2),
				openStr)

			totalAmount = totalAmount.Add(entry.Amount_lcy)
			if entry.Open {
				totalRemaining = totalRemaining.Add(entry.Remaining_amt_lcy)
			}
			totalSales = totalSales.Add(entry.Sales_lcy)
			count++

			if !entry.Next() {
				break
			}
		}
	}

	fmt.Println("------------------------------------------------------------------------------------")
	fmt.Printf("Total Amount: %s\n", totalAmount.StringFixed(2))
	fmt.Printf("Total Remaining (Open only): %s\n", totalRemaining.StringFixed(2))
	fmt.Printf("Total Sales: %s\n", totalSales.StringFixed(2))
	fmt.Printf("Entry Count: %d\n\n", count)

	// Compare with FlowFields
	fmt.Println("FlowField Values (should match totals above):")
	fmt.Printf("  Balance (LCY):        %s", customer.Balance_lcy.StringFixed(2))
	if customer.Balance_lcy.Equal(totalRemaining) {
		fmt.Printf(" ✓\n")
	} else {
		fmt.Printf(" ✗ (Expected: %s)\n", totalRemaining.StringFixed(2))
	}

	fmt.Printf("  Sales (LCY):          %s", customer.Sales_lcy.StringFixed(2))
	if customer.Sales_lcy.Equal(totalSales) {
		fmt.Printf(" ✓\n")
	} else {
		fmt.Printf(" ✗ (Expected: %s)\n", totalSales.StringFixed(2))
	}

	fmt.Printf("  No. of Entries:       %d", customer.No_of_ledger_entries)
	if customer.No_of_ledger_entries == count {
		fmt.Printf(" ✓\n")
	} else {
		fmt.Printf(" ✗ (Expected: %d)\n", count)
	}
}
