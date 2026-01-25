package codeunits

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
)

// CustLedgerEntryGenerate - Codeunit 50010: Generate Random Customer Ledger Entries
const CustLedgerEntryGenerateID = 50010

type CustLedgerEntryGenerate struct {
	db      database.Executor
	company string
}

// NewCustLedgerEntryGenerate creates a new instance of the codeunit
func NewCustLedgerEntryGenerate(db database.Executor, company string) *CustLedgerEntryGenerate {
	return &CustLedgerEntryGenerate{
		db:      db,
		company: company,
	}
}

// Run generates random customer ledger entries for the specified customer
func (c *CustLedgerEntryGenerate) Run(customerNo string, count int) (int, error) {
	if customerNo == "" {
		return 0, fmt.Errorf("customer number is required")
	}

	if count <= 0 {
		count = 5 // Default to 5 entries
	}

	// Verify customer exists
	var customer tables.Customer
	customer.Init(c.db, c.company)
	if !customer.Get(types.NewCode(customerNo)) {
		return 0, fmt.Errorf("customer '%s' not found", customerNo)
	}

	// Get next entry number
	var ledgerEntry tables.CustomerLedgerEntry
	ledgerEntry.Init(c.db, c.company)

	nextEntryNo := 1
	if ledgerEntry.FindLast() {
		nextEntryNo = ledgerEntry.Entry_no + 1
	}

	// Document types: Invoice, Credit Memo, Payment
	docTypes := []tables.CustomerLedgerEntryDocument_type{
		tables.CustomerLedgerEntry_Document_type.Invoice,
		tables.CustomerLedgerEntry_Document_type.CreditMemo,
		tables.CustomerLedgerEntry_Document_type.Payment,
	}
	descriptions := []string{
		"Sales Invoice",
		"Service Invoice",
		"Product Sale",
		"Credit Note",
		"Payment Received",
		"Advance Payment",
	}

	inserted := 0
	for i := 0; i < count; i++ {
		var entry tables.CustomerLedgerEntry
		entry.Init(c.db, c.company)

		entry.Entry_no = nextEntryNo + i
		entry.Customer_no = types.NewCode(customerNo)
		entry.Sell_to_customer_no = types.NewCode(customerNo)

		// Random posting date within last 365 days
		daysAgo := rand.Intn(365)
		postingDate := time.Now().AddDate(0, 0, -daysAgo)
		entry.Posting_date = types.NewDateFromTime(postingDate)
		entry.Document_date = types.NewDateFromTime(postingDate)

		// Random document type
		entry.Document_type = docTypes[rand.Intn(len(docTypes))]

		// Generate document number
		entry.Document_no = types.NewCode(fmt.Sprintf("INV-%06d", nextEntryNo+i))

		// Random description
		entry.Description = types.NewText(descriptions[rand.Intn(len(descriptions))])

		// Random amounts (-5000 to 10000)
		amount := (rand.Float64()*15000 - 5000)
		// Round to 2 decimals
		amount = float64(int(amount*100)) / 100

		entry.Amount = types.NewDecimal(amount)
		entry.Amount_lcy = types.NewDecimal(amount)
		entry.Original_amount_lcy = types.NewDecimal(amount)

		// If invoice, set remaining amount
		if entry.Document_type == tables.CustomerLedgerEntry_Document_type.Invoice {
			entry.Remaining_amount = types.NewDecimal(amount)
			entry.Remaining_amt_lcy = types.NewDecimal(amount)
			entry.Open = true
		} else {
			entry.Remaining_amount = types.NewDecimal(0)
			entry.Remaining_amt_lcy = types.NewDecimal(0)
			entry.Open = false
		}

		entry.Positive = amount > 0

		// Due date (30 days from posting)
		entry.Due_date = types.NewDateFromTime(postingDate.AddDate(0, 0, 30))

		// Insert the entry
		if !entry.Insert(true) {
			return inserted, fmt.Errorf("failed to insert ledger entry %d", entry.Entry_no)
		}
		inserted++
	}

	return inserted, nil
}
