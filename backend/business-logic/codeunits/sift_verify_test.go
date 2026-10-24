package codeunits

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/hansjlachmann/openerp/backend/business-logic/demodata"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

func TestSIFTVerifyCodeunit(t *testing.T) {
	const company = "demo01"
	db, err := sql.Open("sqlite3", "file:siftverify?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	for _, tbl := range []interface {
		CreateTableWithDBType(database.Executor, string, database.DBType) error
	}{&gtables.CountryRegion{}, &gtables.PaymentTerms{}, &gtables.Customer{}, &gtables.CustomerLedgerEntry{}, &gtables.JobQueue{}, &gtables.JobQueueEntry{},
		// the demo data's master data tables
		&gtables.Currency{}, &gtables.FinanceChargeTerms{}, &gtables.PriceGroup{}, &gtables.StandardText{},
		&gtables.ShipmentMethod{}, &gtables.Department{}, &gtables.Project{}, &gtables.SalespersonPurchaser{},
		&gtables.Location{}, &gtables.GeneralLedgerSetup{}, &gtables.CurrencyExchangeRate{},
		&gtables.GenBusinessPostingGroup{}, &gtables.GenProductPostingGroup{}, &gtables.TaxGroup{},
		&gtables.TaxBusinessPostingGroup{}, &gtables.TaxProductPostingGroup{}, &gtables.TaxArea{}, &gtables.GLAccount{},
		&gtables.GLEntry{}, &gtables.SourceCode{}, &gtables.ReasonCode{},
		&gtables.CustomerPostingGroup{}, &gtables.CustomerDiscountGroup{}, &gtables.PaymentMethod{},
		&gtables.ShippingAgent{}, &gtables.Territory{}, &gtables.PostCode{}, &gtables.ReminderTerms{},
		&gtables.ReminderLevel{}, &gtables.ReminderText{}, &gtables.NoSeries{}, &gtables.NoSeriesLine{},
		&gtables.NoSeriesRelationship{}, &gtables.SalesReceivablesSetup{}, &gtables.AccountingPeriod{}, &gtables.CommentLine{},
		&gtables.ShipToAddress{}, &gtables.CustomerBankAccount{}, &gtables.Vendor{}, &gtables.VendorLedgerEntry{},
		&gtables.VendorPostingGroup{}, &gtables.PurchasesPayablesSetup{}, &gtables.OrderAddress{}, &gtables.VendorBankAccount{},
		&gtables.BankAccount{}, &gtables.BankAccountLedgerEntry{}, &gtables.CheckLedgerEntry{}, &gtables.BankAccountPostingGroup{},
		&gtables.FixedAsset{}, &gtables.FALedgerEntry{}, &gtables.FASetup{}, &gtables.FAPostingGroup{}, &gtables.FAClass{}, &gtables.FASubclass{},
		&gtables.FALocation{}, &gtables.DepreciationBook{}, &gtables.FADepreciationBook{}, &gtables.MaintenanceLedgerEntry{}, &gtables.Maintenance{},
		&gtables.MaintenanceRegistration{}, &gtables.MainAssetComponent{}, &gtables.Insurance{}, &gtables.InsuranceType{}, &gtables.InsCoverageLedgerEntry{}} {
		if err := tbl.CreateTableWithDBType(db, company, database.DBTypeSQLite); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := demodata.Create(db, company, database.DBTypeSQLite, demodata.Options{Today: time.Date(2026, 10, 6, 0, 0, 0, 0, time.Local)}); err != nil {
		t.Fatal(err)
	}

	run := func(parameter string) (string, error) {
		job := &gtables.JobQueue{}
		job.InitWithDBType(db, company, database.DBTypeSQLite)
		job.No = types.NewCode("SIFTCHECK")
		job.Description = types.NewText("Verify SIFT")
		job.Parameter = types.NewText(parameter)
		result, err := NewSIFTVerify(db, company, database.DBTypeSQLite).Run(job)
		msg := ""
		if result.Dialog != nil {
			msg = result.Dialog.Message
		}
		return msg, err
	}
	lastEntryStatus := func() gtables.JobQueueEntryStatus {
		var e gtables.JobQueueEntry
		e.InitWithDBType(db, company, database.DBTypeSQLite)
		if !e.FindLast() {
			t.Fatal("no Job Queue Entry logged")
		}
		return e.Status
	}

	// Correct totals: success, the 19 keys checked (customer, vendor, bank, FA, maintenance and
	// insurance coverage ledger, G/L entries)
	msg, err := run("")
	if err != nil || !strings.Contains(msg, "19 SIFT keys") {
		t.Fatalf("verify on correct totals: %q, %v", msg, err)
	}
	if lastEntryStatus() != gtables.JobQueueEntry_Status.Success {
		t.Error("entry not logged as Success")
	}

	// Corrupt a total behind the triggers' back
	if _, err := db.Exec(`UPDATE "demo01$Customer Ledger Entry$SIFT$customer_open" SET remaining_amt_lcy = remaining_amt_lcy + 100 WHERE customer_no = 'C00010'`); err != nil {
		t.Fatal(err)
	}
	if _, err := run("VERIFY"); err == nil || !strings.Contains(err.Error(), "customer_open") {
		t.Fatalf("verify on corrupted totals: err = %v, want the differing key named", err)
	}
	if lastEntryStatus() != gtables.JobQueueEntry_Status.Error {
		t.Error("entry not logged as Error")
	}

	// REPAIR rebuilds, then everything verifies again
	if msg, err := run("repair"); err != nil || !strings.Contains(msg, "customer_open") {
		t.Fatalf("repair: %q, %v", msg, err)
	}
	if _, err := run("VERIFY"); err != nil {
		t.Fatalf("verify after repair: %v", err)
	}

	if _, err := run("FIXIT"); err == nil {
		t.Error("unknown parameter accepted")
	}
}
