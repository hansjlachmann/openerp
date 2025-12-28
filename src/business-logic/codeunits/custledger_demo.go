package codeunits

import (
	"database/sql"
	"fmt"

	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// CustLedgerEntryDemo demonstrates Customer Ledger Entry table operations
func CustLedgerEntryDemo(db *sql.DB, company string) {
	fmt.Println("\n========================================")
	fmt.Println("Customer Ledger Entry Demo")
	fmt.Println("========================================")

	// Create test customers first
	fmt.Println("\n--- Creating Test Customers ---")
	createTestCustomers(db, company)

	// Insert various ledger entries
	fmt.Println("\n--- Inserting Customer Ledger Entries ---")

	entries := []struct {
		customerNo   string
		docType      tables.CustomerLedgerEntryDocument_type
		docNo        string
		description  string
		amount       string
		remainingAmt string
		open         bool
		daysOffset   int
	}{
		// Customer 10000: Invoices
		{"CUST-001", tables.CustomerLedgerEntry_Document_type.Invoice, "INV-2024-001", "December Invoice", "15000.00", "15000.00", true, -30},
		{"CUST-001", tables.CustomerLedgerEntry_Document_type.Invoice, "INV-2024-002", "November Invoice", "25000.00", "25000.00", true, -60},
		{"CUST-001", tables.CustomerLedgerEntry_Document_type.Invoice, "INV-2023-050", "Old Invoice (Paid)", "10000.00", "0.00", false, -180},

		// Customer 10000: Payments
		{"CUST-001", tables.CustomerLedgerEntry_Document_type.Payment, "PMT-2024-005", "Payment December", "-10000.00", "0.00", false, -15},
		{"CUST-001", tables.CustomerLedgerEntry_Document_type.Payment, "PMT-2024-006", "Partial Payment November", "-15000.00", "0.00", false, -45},

		// Customer 20000: Invoices and Credit Memos
		{"CUST-002", tables.CustomerLedgerEntry_Document_type.Invoice, "INV-2024-010", "Product Sale", "50000.00", "50000.00", true, -20},
		{"CUST-002", tables.CustomerLedgerEntry_Document_type.CreditMemo, "CM-2024-001", "Return - Damaged Goods", "-5000.00", "-5000.00", true, -10},
		{"CUST-002", tables.CustomerLedgerEntry_Document_type.Payment, "PMT-2024-015", "Payment received", "-20000.00", "0.00", false, -5},

		// Customer 30000: Finance charges and reminders
		{"CUST-003", tables.CustomerLedgerEntry_Document_type.Invoice, "INV-2023-100", "Overdue Invoice", "8000.00", "8000.00", true, -120},
		{"CUST-003", tables.CustomerLedgerEntry_Document_type.FinanceChargeMemo, "FCM-2024-001", "Late Payment Charge", "200.00", "200.00", true, -30},
		{"CUST-003", tables.CustomerLedgerEntry_Document_type.Reminder, "REM-2024-001", "Payment Reminder", "0.00", "0.00", true, -30},
	}

	entryNo := 1
	for _, e := range entries {
		entry := tables.NewCustomerLedgerEntry()
		entry.Init(db, company)

		entry.Entry_no = entryNo
		entry.Customer_no = types.NewCode(e.customerNo)
		entry.Sell_to_customer_no = types.NewCode(e.customerNo)
		entry.Document_type = e.docType
		entry.Document_no = types.NewCode(e.docNo)
		entry.Description = types.NewText(e.description)

		// Set posting date relative to today
		entry.Posting_date = types.Today().AddDays(e.daysOffset)
		entry.Document_date = entry.Posting_date
		entry.Due_date = entry.Posting_date.AddDays(30)

		// Set amounts
		amount, _ := types.NewDecimalFromString(e.amount)
		remainingAmt, _ := types.NewDecimalFromString(e.remainingAmt)

		entry.Amount = amount
		entry.Remaining_amount = remainingAmt
		entry.Amount_lcy = amount
		entry.Remaining_amt_lcy = remainingAmt
		entry.Original_amount_lcy = amount

		// Set sales/profit (for invoices only)
		if e.docType == tables.CustomerLedgerEntry_Document_type.Invoice {
			entry.Sales_lcy = amount
			// Assume 30% profit margin
			entry.Profit_lcy = amount.Mul(types.NewDecimal(0.30))
		}

		// Status fields
		entry.Open = e.open
		entry.Positive = amount.IsPositive()

		// Currency (blank = local currency)
		entry.Currency_code = types.NewCode("")

		// System fields
		entry.User_id = types.NewCode("ADMIN")
		entry.Transaction_no = entryNo

		if entry.Insert(true) {
			fmt.Printf("  ✓ Entry %d: %s - %s %s (%s) = %s%s\n",
				entryNo,
				e.customerNo,
				e.docType.String(),
				e.docNo,
				e.description,
				entry.Currency_code,
				amount.StringFixed(2))
		} else {
			fmt.Printf("  ✗ Failed to insert entry %d\n", entryNo)
		}

		entryNo++
	}

	// Show summary by customer
	fmt.Println("\n--- Customer Ledger Summary ---")
	showCustomerBalance(db, company, "CUST-001")
	showCustomerBalance(db, company, "CUST-002")
	showCustomerBalance(db, company, "CUST-003")

	// Show open entries
	fmt.Println("\n--- Open Entries (Unpaid) ---")
	showOpenEntries(db, company)

	fmt.Println("\n========================================")
	fmt.Println("Demo Complete!")
	fmt.Println("========================================\n")
}

// createTestCustomers creates test customers if they don't exist
func createTestCustomers(db *sql.DB, company string) {
	customers := []struct {
		no   string
		name string
	}{
		{"CUST-001", "ABC Manufacturing Ltd."},
		{"CUST-002", "XYZ Retail Corp."},
		{"CUST-003", "Global Services Inc."},
	}

	for _, c := range customers {
		customer := tables.NewCustomer()
		customer.Init(db, company)

		// Check if customer exists
		if !customer.Get(types.NewCode(c.no)) {
			customer.No = types.NewCode(c.no)
			customer.Name = types.NewText(c.name)
			customer.Status = tables.Customer_Status.Open
			customer.Credit_limit = types.NewDecimal(100000.00)

			if customer.Insert(true) {
				fmt.Printf("  ✓ Created customer: %s - %s\n", c.no, c.name)
			}
		} else {
			fmt.Printf("  ⊙ Customer exists: %s - %s\n", c.no, c.name)
		}
	}
}

// showCustomerBalance calculates and displays customer balance
func showCustomerBalance(db *sql.DB, company, customerNo string) {
	entry := tables.NewCustomerLedgerEntry()
	entry.Init(db, company)
	entry.SetRange("customer_no", types.NewCode(customerNo), types.NewCode(customerNo))

	totalBalance := types.ZeroDecimal()
	openBalance := types.ZeroDecimal()
	invoiceCount := 0
	paymentCount := 0

	if entry.FindSet() {
		for {
			totalBalance = totalBalance.Add(entry.Amount_lcy)
			if entry.Open {
				openBalance = openBalance.Add(entry.Remaining_amt_lcy)
			}

			if entry.Document_type == tables.CustomerLedgerEntry_Document_type.Invoice {
				invoiceCount++
			} else if entry.Document_type == tables.CustomerLedgerEntry_Document_type.Payment {
				paymentCount++
			}

			if !entry.Next() {
				break
			}
		}
	}

	fmt.Printf("\nCustomer: %s\n", customerNo)
	fmt.Printf("  Total Activity:  %s\n", totalBalance.StringFixed(2))
	fmt.Printf("  Open Balance:    %s\n", openBalance.StringFixed(2))
	fmt.Printf("  Invoices:        %d\n", invoiceCount)
	fmt.Printf("  Payments:        %d\n", paymentCount)
}

// showOpenEntries displays all open (unpaid) entries
func showOpenEntries(db *sql.DB, company string) {
	entry := tables.NewCustomerLedgerEntry()
	entry.Init(db, company)
	entry.SetRange("open", true, true)

	count := 0
	totalOpen := types.ZeroDecimal()

	fmt.Printf("%-12s %-20s %-20s %-15s %12s\n", "Customer", "Doc. Type", "Doc. No.", "Date", "Remaining")
	fmt.Println("--------------------------------------------------------------------------------")

	if entry.FindSet() {
		for {
			fmt.Printf("%-12s %-20s %-20s %-15s %12s\n",
				entry.Customer_no.String(),
				entry.Document_type.String(),
				entry.Document_no.String(),
				entry.Posting_date.String(),
				entry.Remaining_amt_lcy.StringFixed(2))

			totalOpen = totalOpen.Add(entry.Remaining_amt_lcy)
			count++

			if !entry.Next() {
				break
			}
		}
	}

	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("Total Open Entries: %d, Total Amount: %s\n", count, totalOpen.StringFixed(2))
}
