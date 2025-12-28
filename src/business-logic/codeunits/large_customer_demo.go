package codeunits

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// CreateLargeCustomerDataset creates a large dataset of 100,000+ ledger entries for customer CUST-001
// Codeunit ID: 50011
func CreateLargeCustomerDataset(db *sql.DB, company string) {
	rand.Seed(time.Now().UnixNano())

	fmt.Println("========================================")
	fmt.Println("Codeunit 50011 - Create Large Dataset")
	fmt.Println("========================================\n")

	// Check if customer CUST-001 exists, if not create it
	fmt.Println("Checking for customer CUST-001...")
	cust := tables.NewCustomer()
	cust.Init(db, company)

	if !cust.Get(types.NewCode("CUST-001")) {
		fmt.Println("Customer CUST-001 not found. Creating...")
		cust.No = types.NewCode("CUST-001")
		cust.Name = types.NewText("Large Volume Customer Ltd.")
		cust.Status = tables.Customer_Status.Open
		cust.Credit_limit = types.NewDecimal(10000000.00)
		cust.Last_order_date = types.Today()
		cust.Created_at = types.Now()

		if !cust.Insert(false) {
			fmt.Println("✗ Failed to create customer CUST-001")
			return
		}
		fmt.Println("✓ Customer CUST-001 created\n")
	} else {
		fmt.Println("✓ Customer CUST-001 already exists\n")
	}

	// Find the maximum entry_no currently in the table
	var maxEntryNo int
	tableName := fmt.Sprintf("%s$Customer Ledger Entry", company)
	query := fmt.Sprintf(`SELECT COALESCE(MAX(entry_no), 0) FROM "%s"`, tableName)
	err := db.QueryRow(query).Scan(&maxEntryNo)
	if err != nil {
		fmt.Printf("✗ Failed to query max entry_no: %v\n", err)
		return
	}

	fmt.Printf("Current max entry_no: %d\n", maxEntryNo)
	startEntryNo := maxEntryNo + 1

	// Create 100,000+ ledger entries
	numEntries := 100000
	fmt.Printf("Creating %d ledger entries with random amounts...\n", numEntries)
	fmt.Printf("Entry numbers will range from %d to %d\n", startEntryNo, startEntryNo+numEntries-1)
	fmt.Println("(This may take a few minutes...)\n")

	start := time.Now()
	batchSize := 1000
	commitBatchSize := 1000 // Commit transaction every N inserts

	// Enable WAL mode for better concurrent write performance
	db.Exec("PRAGMA journal_mode=WAL")

	// Start transaction for batch inserts
	_, err = db.Exec("BEGIN TRANSACTION")
	if err != nil {
		fmt.Printf("✗ Failed to start transaction: %v\n", err)
		return
	}

	for i := 0; i < numEntries; i++ {
		entryNo := startEntryNo + i
		entry := tables.NewCustomerLedgerEntry()
		entry.Init(db, company)

		entry.Entry_no = entryNo
		entry.Customer_no = types.NewCode("CUST-001")
		entry.Sell_to_customer_no = types.NewCode("CUST-001")

		// Randomly choose document type
		// 60% invoices, 30% payments, 10% credit memos
		docTypeRand := rand.Intn(100)

		var amount types.Decimal
		if docTypeRand < 60 {
			// Invoice - positive amount
			entry.Document_type = tables.CustomerLedgerEntry_Document_type.Invoice
			entry.Document_no = types.NewCode(fmt.Sprintf("INV-%08d", entryNo))
			entry.Description = types.NewText("Sales Invoice")

			// Random amount between 100 and 50,000
			amount = types.NewDecimalFromInt(int64(100 + rand.Intn(49900)))
			entry.Sales_lcy = amount
			entry.Profit_lcy = amount.Mul(types.NewDecimal(0.25))
		} else if docTypeRand < 90 {
			// Payment - negative amount
			entry.Document_type = tables.CustomerLedgerEntry_Document_type.Payment
			entry.Document_no = types.NewCode(fmt.Sprintf("PMT-%08d", entryNo))
			entry.Description = types.NewText("Payment Received")

			// Random amount between -100 and -30,000
			amount = types.NewDecimalFromInt(int64(-(100 + rand.Intn(29900))))
		} else {
			// Credit Memo - negative amount
			entry.Document_type = tables.CustomerLedgerEntry_Document_type.CreditMemo
			entry.Document_no = types.NewCode(fmt.Sprintf("CM-%08d", entryNo))
			entry.Description = types.NewText("Credit Memo")

			// Random amount between -50 and -5,000
			amount = types.NewDecimalFromInt(int64(-(50 + rand.Intn(4950))))
		}

		entry.Amount = amount
		entry.Amount_lcy = amount
		entry.Original_amount_lcy = amount

		// Randomly mark entries as open or closed
		// 40% open, 60% closed
		if rand.Intn(100) < 40 {
			entry.Open = true

			// For open entries, remaining = original (not yet paid)
			if entry.Document_type == tables.CustomerLedgerEntry_Document_type.Invoice {
				// Random remaining between 10% and 100% of original
				remainingPercent := 0.1 + (rand.Float64() * 0.9)
				entry.Remaining_amount = amount.Mul(types.NewDecimal(remainingPercent))
				entry.Remaining_amt_lcy = entry.Remaining_amount
			} else {
				entry.Remaining_amount = amount
				entry.Remaining_amt_lcy = amount
			}
		} else {
			entry.Open = false
			entry.Remaining_amount = types.ZeroDecimal()
			entry.Remaining_amt_lcy = types.ZeroDecimal()
		}

		// Random dates within last 2 years
		daysAgo := rand.Intn(730)
		entry.Posting_date = types.Today().AddDays(-daysAgo)
		entry.Document_date = entry.Posting_date
		entry.Due_date = entry.Posting_date.AddDays(30)

		entry.Currency_code = types.NewCode("")
		entry.Positive = amount.IsPositive()
		entry.User_id = types.NewCode("ADMIN")
		entry.Transaction_no = entryNo

		if !entry.Insert(false) {
			fmt.Printf("\n✗ Failed to insert entry %d\n", entryNo)
			db.Exec("ROLLBACK")
			return
		}

		// Commit transaction and start a new one every commitBatchSize inserts
		if (i+1)%commitBatchSize == 0 {
			if _, err := db.Exec("COMMIT"); err != nil {
				fmt.Printf("\n✗ Failed to commit transaction: %v\n", err)
				return
			}
			// Start new transaction
			if _, err := db.Exec("BEGIN TRANSACTION"); err != nil {
				fmt.Printf("\n✗ Failed to start new transaction: %v\n", err)
				return
			}
		}

		// Progress indicator
		if (i+1)%batchSize == 0 {
			elapsed := time.Since(start)
			rate := float64(i+1) / elapsed.Seconds()
			remaining := time.Duration(float64(numEntries-(i+1))/rate) * time.Second

			fmt.Printf("\r  Progress: %d/%d (%.1f%%) | Rate: %.0f/sec | ETA: %v     ",
				i+1, numEntries, float64(i+1)*100.0/float64(numEntries), rate, remaining.Round(time.Second))
		}
	}

	// Final commit for remaining records
	if _, err := db.Exec("COMMIT"); err != nil {
		fmt.Printf("\n✗ Failed to commit final transaction: %v\n", err)
		return
	}

	elapsed := time.Since(start)
	fmt.Printf("\r✓ Created %d entries in %v (%.0f entries/sec)          \n\n",
		numEntries, elapsed, float64(numEntries)/elapsed.Seconds())

	fmt.Println("========================================")
	fmt.Println("Dataset Creation Complete!")
	fmt.Println("========================================")
}

// CalcFieldsLargeCustomer calculates and displays FlowFields for customer CUST-001
// Codeunit ID: 50012
func CalcFieldsLargeCustomer(db *sql.DB, company string) {
	fmt.Println("========================================")
	fmt.Println("Codeunit 50012 - Calculate FlowFields")
	fmt.Println("========================================\n")

	fmt.Println("Loading customer CUST-001...")
	customer := tables.NewCustomer()
	customer.Init(db, company)

	if !customer.Get(types.NewCode("CUST-001")) {
		fmt.Println("✗ Customer CUST-001 not found")
		fmt.Println("Please run Codeunit 50011 first to create the dataset.")
		return
	}
	fmt.Printf("✓ Customer loaded: %s\n\n", customer.Name.String())

	// Calculate FlowFields with timing
	fmt.Println("Calculating FlowFields (Balance, Sales, Entry Count)...")
	fmt.Println("Dataset: 100,000+ entries")
	fmt.Println("Indexes: customer_open (customer_no, open)")
	fmt.Println()

	calcStart := time.Now()
	customer.CalcFields()
	calcDuration := time.Since(calcStart)

	// Display results
	fmt.Println("========================================")
	fmt.Println("Results")
	fmt.Println("========================================\n")

	fmt.Printf("Customer: %s - %s\n\n", customer.No.String(), customer.Name.String())
	fmt.Printf("FlowField Values:\n")
	fmt.Printf("  Balance (LCY):        %s\n", customer.Balance_lcy.StringFixed(2))
	fmt.Printf("  Sales (LCY):          %s\n", customer.Sales_lcy.StringFixed(2))
	fmt.Printf("  No. of Entries:       %d\n\n", customer.No_of_ledger_entries)

	fmt.Printf("Performance:\n")
	fmt.Printf("  Calculation Time:     %v\n", calcDuration)
	fmt.Printf("  Entries Processed:    %d\n", customer.No_of_ledger_entries)
	if customer.No_of_ledger_entries > 0 {
		fmt.Printf("  Time per Entry:       %v\n\n", calcDuration/time.Duration(customer.No_of_ledger_entries))
	}

	// Verify by manual calculation
	fmt.Println("========================================")
	fmt.Println("Verification (Manual Count)")
	fmt.Println("========================================\n")

	verifyStart := time.Now()

	entry := tables.NewCustomerLedgerEntry()
	entry.Init(db, company)
	entry.SetRange("customer_no", types.NewCode("CUST-001"), types.NewCode("CUST-001"))

	totalBalance := types.ZeroDecimal()
	totalSales := types.ZeroDecimal()
	count := 0
	openCount := 0

	if entry.FindSet() {
		for {
			if entry.Open {
				totalBalance = totalBalance.Add(entry.Remaining_amt_lcy)
				openCount++
			}
			totalSales = totalSales.Add(entry.Sales_lcy)
			count++

			if !entry.Next() {
				break
			}
		}
	}

	verifyDuration := time.Since(verifyStart)

	fmt.Printf("Manual Calculation Results:\n")
	fmt.Printf("  Total Balance (Open):  %s\n", totalBalance.StringFixed(2))
	fmt.Printf("  Total Sales:           %s\n", totalSales.StringFixed(2))
	fmt.Printf("  Total Entries:         %d\n", count)
	fmt.Printf("  Open Entries:          %d\n\n", openCount)

	fmt.Printf("Manual Calculation Time:   %v\n", verifyDuration)
	fmt.Printf("FlowField Calculation Time: %v\n\n", calcDuration)

	// Check if results match
	fmt.Println("Validation:")
	if customer.Balance_lcy.Equal(totalBalance) {
		fmt.Println("  ✓ Balance matches")
	} else {
		fmt.Printf("  ✗ Balance mismatch: FlowField=%s, Manual=%s\n",
			customer.Balance_lcy.StringFixed(2), totalBalance.StringFixed(2))
	}

	if customer.Sales_lcy.Equal(totalSales) {
		fmt.Println("  ✓ Sales matches")
	} else {
		fmt.Printf("  ✗ Sales mismatch: FlowField=%s, Manual=%s\n",
			customer.Sales_lcy.StringFixed(2), totalSales.StringFixed(2))
	}

	if customer.No_of_ledger_entries == count {
		fmt.Println("  ✓ Entry count matches")
	} else {
		fmt.Printf("  ✗ Entry count mismatch: FlowField=%d, Manual=%d\n",
			customer.No_of_ledger_entries, count)
	}

	fmt.Println("\n========================================")
	fmt.Println("FlowField Calculation Complete!")
	fmt.Println("========================================")
}
