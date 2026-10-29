package salesposting_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/hansjlachmann/openerp/backend/business-logic/demodata"
	"github.com/hansjlachmann/openerp/backend/business-logic/itemjnl"
	"github.com/hansjlachmann/openerp/backend/business-logic/salesposting"
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// The demo sales documents: posted through codeunit 80 — invoices, shipments, a credit memo —
// and open quotes, orders and an invoice; the G/L stays balanced
func TestDemoSales(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	must(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	must(t, tables.CreateCompanyTables(db, "demo", database.DBTypeSQLite))
	counts, err := demodata.Create(db, "demo", database.DBTypeSQLite, demodata.Options{
		Today:           time.Date(2026, 10, 6, 0, 0, 0, 0, time.Local),
		ItemJnlPostLine: func(src ftables.Source) demodata.ItemJnlPoster { return itemjnl.NewItemJnlPostLine(src) },
		SalesPost:       func(src ftables.Source) demodata.SalesPoster { return salesposting.NewSalesPost(src) },
	})
	must(t, err)
	if counts.PostedSalesInvoices < 12 || counts.PostedSalesShipments < 12 || counts.PostedSalesCrMemos != 1 ||
		counts.SalesQuotes != 3 || counts.SalesOrders != 3 || counts.SalesInvoices != 1 {
		t.Errorf("counts %+v", counts)
	}
	src := ftables.DBSource(db, "demo", database.DBTypeSQLite)
	salesInvHeader := gtables.NewSalesInvoiceHeader(src)
	if salesInvHeader.Count() != counts.PostedSalesInvoices {
		t.Errorf("posted invoices %d, counted %d", salesInvHeader.Count(), counts.PostedSalesInvoices)
	}
	salesHeader := gtables.NewSalesHeader(src)
	if salesHeader.Count() != counts.SalesQuotes+counts.SalesOrders+counts.SalesInvoices {
		t.Errorf("open documents %d", salesHeader.Count())
	}
	glEntry := gtables.NewGLEntry(src)
	must(t, glEntry.CalcSums("amount"))
	if !glEntry.Amount.Round(2).IsZero() { // SQLite sums decimals as floating point
		t.Errorf("G/L balance %s", glEntry.Amount)
	}
	crMemo := gtables.NewSalesCrMemoHeader(src)
	if !crMemo.FindFirst() || crMemo.Applies_to_doc_no == "" {
		t.Errorf("credit memo applied to %q", crMemo.Applies_to_doc_no)
	}
	// the order shipped in part is still open with what is left
	salesLine := gtables.NewSalesLine(src)
	salesLine.SetRange("document_type", gtables.SalesLine_Document_type.Order)
	salesLine.SetFilter("quantity_shipped", "<>0")
	if salesLine.Count() == 0 {
		t.Error("no open order shipped in part")
	}
}
