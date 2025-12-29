package codeunits

import (
	"database/sql"
	"fmt"

	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/i18n"
)

// TranslationDemo demonstrates the multilanguage translation system
// Codeunit ID: 50014
func TranslationDemo(db *sql.DB, company string) {
	fmt.Println("\n========================================")
	fmt.Println("Translation Demo (Codeunit 50014)")
	fmt.Println("========================================\n")

	ts := i18n.GetInstance()

	// Display supported languages
	languages := ts.GetSupportedLanguages()
	fmt.Printf("Supported languages: %v\n", languages)
	fmt.Printf("Default language: %s\n\n", ts.GetDefaultLanguage())

	// Demo 1: Table captions
	fmt.Println("--- Demo 1: Table Captions ---")
	demoTableCaptions()

	// Demo 2: Field captions
	fmt.Println("\n--- Demo 2: Field Captions ---")
	demoFieldCaptions()

	// Demo 3: Option captions
	fmt.Println("\n--- Demo 3: Option Captions ---")
	demoOptionCaptions()

	// Demo 4: Using table methods
	fmt.Println("\n--- Demo 4: Using Table Methods ---")
	demoTableMethods(db, company)

	fmt.Println("\n✓ Translation demo complete!")
}

// demoTableCaptions demonstrates table caption translation
func demoTableCaptions() {
	ts := i18n.GetInstance()

	tables := []string{"Customer", "Payment_terms", "Customer_ledger_entry"}

	for _, tableName := range tables {
		enCaption := ts.TableCaption(tableName, "en-US")
		noCaption := ts.TableCaption(tableName, "nb-NO")
		fmt.Printf("  %-25s | EN: %-30s | NO: %s\n", tableName, enCaption, noCaption)
	}
}

// demoFieldCaptions demonstrates field caption translation
func demoFieldCaptions() {
	ts := i18n.GetInstance()

	// Customer table fields
	fields := []struct {
		table string
		field string
	}{
		{"Customer", "no"},
		{"Customer", "name"},
		{"Customer", "address"},
		{"Customer", "payment_terms_code"},
		{"Customer", "credit_limit"},
		{"Customer", "status"},
		{"Payment_terms", "code"},
		{"Payment_terms", "description"},
		{"Payment_terms", "discount_percent"},
	}

	for _, f := range fields {
		enCaption := ts.FieldCaption(f.table, f.field, "en-US")
		noCaption := ts.FieldCaption(f.table, f.field, "nb-NO")
		fmt.Printf("  %-30s | EN: %-25s | NO: %s\n", f.table+"."+f.field, enCaption, noCaption)
	}
}

// demoOptionCaptions demonstrates option field value translation
func demoOptionCaptions() {
	ts := i18n.GetInstance()

	// Customer status options
	statusOptions := []string{"open", "blocked", "closed"}

	fmt.Println("  Customer.Status options:")
	for _, option := range statusOptions {
		enCaption := ts.OptionCaption("Customer", "status", option, "en-US")
		noCaption := ts.OptionCaption("Customer", "status", option, "nb-NO")
		fmt.Printf("    %-10s | EN: %-10s | NO: %s\n", option, enCaption, noCaption)
	}

	// Customer Ledger Entry document type options
	docTypeOptions := []string{" ", "payment", "invoice", "credit_memo", "finance_charge_memo", "reminder"}

	fmt.Println("\n  Customer_ledger_entry.Document_type options:")
	for _, option := range docTypeOptions {
		enCaption := ts.OptionCaption("Customer_ledger_entry", "document_type", option, "en-US")
		noCaption := ts.OptionCaption("Customer_ledger_entry", "document_type", option, "nb-NO")
		displayOption := option
		if option == " " {
			displayOption = "(blank)"
		}
		fmt.Printf("    %-20s | EN: %-20s | NO: %s\n", displayOption, enCaption, noCaption)
	}
}

// demoTableMethods demonstrates using translation methods on table instances
func demoTableMethods(db *sql.DB, company string) {
	// Customer table
	var customer tables.Customer
	customer.Init(db, company)

	fmt.Println("  Customer table:")
	fmt.Printf("    English caption: %s\n", customer.GetCaption("en-US"))
	fmt.Printf("    Norwegian caption: %s\n", customer.GetCaption("nb-NO"))
	fmt.Printf("    Field 'no' (EN): %s\n", customer.GetFieldCaption("no", "en-US"))
	fmt.Printf("    Field 'no' (NO): %s\n", customer.GetFieldCaption("no", "nb-NO"))
	fmt.Printf("    Field 'name' (EN): %s\n", customer.GetFieldCaption("name", "en-US"))
	fmt.Printf("    Field 'name' (NO): %s\n", customer.GetFieldCaption("name", "nb-NO"))
	fmt.Printf("    Status option 'open' (EN): %s\n", customer.GetOptionCaption("status", "open", "en-US"))
	fmt.Printf("    Status option 'open' (NO): %s\n", customer.GetOptionCaption("status", "open", "nb-NO"))

	// Payment Terms table
	var paymentTerms tables.PaymentTerms
	paymentTerms.Init(db, company)

	fmt.Println("\n  Payment Terms table:")
	fmt.Printf("    English caption: %s\n", paymentTerms.GetCaption("en-US"))
	fmt.Printf("    Norwegian caption: %s\n", paymentTerms.GetCaption("nb-NO"))
	fmt.Printf("    Field 'code' (EN): %s\n", paymentTerms.GetFieldCaption("code", "en-US"))
	fmt.Printf("    Field 'code' (NO): %s\n", paymentTerms.GetFieldCaption("code", "nb-NO"))
}
