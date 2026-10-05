package demodata

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
)

const testCompany = "demo01"

// newTestDB opens an in-memory SQLite DB with the tables demo data writes to.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// One connection: every connection to :memory: is a separate database
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	creators := []interface {
		CreateTableWithDBType(database.Executor, string, database.DBType) error
	}{&tables.CountryRegion{}, &tables.PaymentTerms{}, &tables.Customer{}, &tables.CustomerLedgerEntry{}}
	for _, c := range creators {
		if err := c.CreateTableWithDBType(db, testCompany, database.DBTypeSQLite); err != nil {
			t.Fatalf("create table: %v", err)
		}
	}
	return db
}

func count(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM "` + testCompany + `$` + table + `"`).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

var today = time.Date(2026, 10, 5, 0, 0, 0, 0, time.Local)

func TestParseSize(t *testing.T) {
	for in, want := range map[string]Size{"": Small, "small": Small, " SMALL ": Small, "Large": Large} {
		got, err := ParseSize(in)
		if err != nil || got != want {
			t.Errorf("ParseSize(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	var sizeErr *SizeError
	if _, err := ParseSize("HUGE"); !errors.As(err, &sizeErr) {
		t.Errorf("ParseSize(HUGE) error = %v, want SizeError", err)
	}
}

func TestCreateSmall(t *testing.T) {
	db := newTestDB(t)
	var lastPct int
	counts, err := Create(db, testCompany, database.DBTypeSQLite, Options{
		Size:     Small,
		Today:    today,
		Progress: func(pct int, _ Phase) { lastPct = pct },
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if counts.Countries != 3 || counts.PaymentTerms != 4 || counts.Customers != 20 {
		t.Errorf("counts = %+v, want 3 countries, 4 payment terms, 20 customers", counts)
	}
	if counts.LedgerEntries < 20*6 {
		t.Errorf("only %d ledger entries", counts.LedgerEntries)
	}
	if got := count(t, db, "Customer"); got != 20 {
		t.Errorf("Customer rows = %d, want 20", got)
	}
	if got := count(t, db, "Customer Ledger Entry"); got != counts.LedgerEntries {
		t.Errorf("ledger rows = %d, want %d", got, counts.LedgerEntries)
	}
	if lastPct != 100 {
		t.Errorf("last progress = %d, want 100", lastPct)
	}

	// Mixed countries, with the country's own address and phone format
	var cust tables.Customer
	cust.InitWithDBType(db, testCompany, database.DBTypeSQLite)
	if !cust.Get("C00080") {
		t.Fatal("customer C00080 not found")
	}
	if cust.Country_region_code.String() != "DK" || cust.Name.String() != "Nørrebro Møbler ApS" {
		t.Errorf("C00080 = %s / %s", cust.Country_region_code, cust.Name)
	}
	if cust.Last_order_date.IsZero() {
		t.Error("C00080 has no last order date")
	}
	if cust.Status != tables.Customer_Status.Open {
		t.Errorf("C00080 status = %v, want Open", cust.Status)
	}
	if !cust.Get("C00070") || cust.Status != tables.Customer_Status.Blocked {
		t.Errorf("C00070 status = %v, want Blocked", cust.Status)
	}

	// Balance = open invoice amounts; every closed invoice is settled by a payment
	var open, unbalanced int
	if err := db.QueryRow(`SELECT COUNT(*) FROM "demo01$Customer Ledger Entry" WHERE open = 1`).Scan(&open); err != nil {
		t.Fatal(err)
	}
	if open == 0 {
		t.Error("expected some open invoices")
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM "demo01$Customer Ledger Entry" WHERE open = 0 AND remaining_amt_lcy <> 0`).Scan(&unbalanced); err != nil {
		t.Fatal(err)
	}
	if unbalanced != 0 {
		t.Errorf("%d closed entries still have a remaining amount", unbalanced)
	}
}

func TestCreateIsRepeatable(t *testing.T) {
	sum := func() (int, float64) {
		db := newTestDB(t)
		counts, err := Create(db, testCompany, database.DBTypeSQLite, Options{Today: today})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		var total float64
		if err := db.QueryRow(`SELECT SUM(amount) FROM "demo01$Customer Ledger Entry"`).Scan(&total); err != nil {
			t.Fatal(err)
		}
		return counts.LedgerEntries, total
	}
	n1, t1 := sum()
	n2, t2 := sum()
	if n1 != n2 || t1 != t2 {
		t.Errorf("runs differ: %d entries / %.2f vs %d entries / %.2f", n1, t1, n2, t2)
	}
}

func TestCreateRefusesCompanyWithCustomers(t *testing.T) {
	db := newTestDB(t)
	if _, err := Create(db, testCompany, database.DBTypeSQLite, Options{Today: today}); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	entries := count(t, db, "Customer Ledger Entry")

	_, err := Create(db, testCompany, database.DBTypeSQLite, Options{Today: today})
	if !errors.Is(err, ErrCompanyNotEmpty) {
		t.Fatalf("second Create error = %v, want ErrCompanyNotEmpty", err)
	}
	if got := count(t, db, "Customer Ledger Entry"); got != entries {
		t.Errorf("second run changed the ledger: %d -> %d entries", entries, got)
	}
}

func TestCreateKeepsExistingSetup(t *testing.T) {
	db := newTestDB(t)
	var pt tables.PaymentTerms
	pt.InitWithDBType(db, testCompany, database.DBTypeSQLite)
	pt.Code = "30 DAYS"
	pt.Description = "Our own 30 days"
	if !pt.Insert(true) {
		t.Fatal("insert payment terms")
	}

	counts, err := Create(db, testCompany, database.DBTypeSQLite, Options{Today: today})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if counts.PaymentTerms != 3 {
		t.Errorf("PaymentTerms inserted = %d, want 3 (one existed)", counts.PaymentTerms)
	}
	pt.Get("30 DAYS")
	if pt.Description.String() != "Our own 30 days" {
		t.Errorf("existing payment terms overwritten: %q", pt.Description)
	}
}

func TestGenerateLargeCustomers(t *testing.T) {
	var gen generatorFile
	var setup setupData
	if err := loadYAML("generator.yaml", &gen); err != nil {
		t.Fatal(err)
	}
	if err := loadYAML("setup.yaml", &setup); err != nil {
		t.Fatal(err)
	}
	customers := generateCustomers(newRand(), gen, setup, largeCustomerCount-20)
	if len(customers) != largeCustomerCount-20 {
		t.Fatalf("generated %d customers", len(customers))
	}
	seen := map[string]bool{}
	perCountry := map[string]int{}
	for _, c := range customers {
		if seen[c.No] {
			t.Fatalf("duplicate customer no %s", c.No)
		}
		seen[c.No] = true
		perCountry[c.Country]++
		if len([]rune(c.Name)) > 50 || len([]rune(c.Address)) > 50 || len(c.Phone) > 30 {
			t.Fatalf("value too long for its field: %+v", c)
		}
	}
	if customers[0].No != "C00201" || len(perCountry) != 3 {
		t.Errorf("first no = %s, countries = %v", customers[0].No, perCountry)
	}
}
