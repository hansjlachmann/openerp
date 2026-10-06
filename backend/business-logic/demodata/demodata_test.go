package demodata

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
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

// CalcFieldsForRecords (list pages) must give the same FlowField values as CalcFields
// per record (card page), including records without ledger entries and key lists
// longer than one query chunk.
func TestCalcFieldsForRecordsMatchesCalcFields(t *testing.T) {
	db := newTestDB(t)
	if _, err := Create(db, testCompany, database.DBTypeSQLite, Options{Today: today}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	var cust tables.Customer
	cust.InitWithDBType(db, testCompany, database.DBTypeSQLite)
	var records []map[string]interface{}
	if cust.FindSet() {
		records = append(records, cust.ToMap())
		for cust.Next() {
			records = append(records, cust.ToMap())
		}
	}
	if len(records) != 20 {
		t.Fatalf("read %d customers, want 20", len(records))
	}
	// Expected values: CalcFields per record, after the result set is closed (the
	// test DB has a single connection)
	want := map[string]map[string]interface{}{}
	for _, rec := range records {
		var one tables.Customer
		one.InitWithDBType(db, testCompany, database.DBTypeSQLite)
		no := rec["no"].(string)
		if !one.Get(no) {
			t.Fatalf("customer %s not found", no)
		}
		one.CalcFields()
		calc := one.ToMap()
		want[no] = map[string]interface{}{
			"balance_lcy": calc["balance_lcy"], "sales_lcy": calc["sales_lcy"], "no_of_ledger_entries": calc["no_of_ledger_entries"],
		}
	}
	// Few keys: one grouped query with an IN list
	few := make([]map[string]interface{}, len(records))
	for i, rec := range records {
		few[i] = map[string]interface{}{"no": rec["no"]}
	}
	cust.CalcFieldsForRecords(few)
	for _, rec := range few {
		no := rec["no"].(string)
		for field, w := range want[no] {
			if fmt.Sprint(rec[field]) != fmt.Sprint(w) {
				t.Errorf("IN list: %s.%s = %v, want %v", no, field, rec[field], w)
			}
		}
	}

	// Many keys (more than one chunk, incl. customers without entries): one grouped
	// query over the whole ledger
	for i := 0; i < 1200; i++ {
		no := fmt.Sprintf("X%05d", i)
		records = append(records, map[string]interface{}{"no": no})
		want[no] = map[string]interface{}{"balance_lcy": "0", "sales_lcy": "0", "no_of_ledger_entries": 0}
	}

	cust.CalcFieldsForRecords(records)

	for _, rec := range records {
		no := rec["no"].(string)
		for field, w := range want[no] {
			if fmt.Sprint(rec[field]) != fmt.Sprint(w) {
				t.Errorf("%s.%s = %v, want %v", no, field, rec[field], w)
			}
		}
	}
	if fmt.Sprint(want["C00010"]["no_of_ledger_entries"]) == "0" {
		t.Error("C00010 should have ledger entries")
	}
}

func TestCalcFieldsForRecordsOnlyRequested(t *testing.T) {
	db := newTestDB(t)
	if _, err := Create(db, testCompany, database.DBTypeSQLite, Options{Today: today}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	var cust tables.Customer
	cust.InitWithDBType(db, testCompany, database.DBTypeSQLite)
	records := []map[string]interface{}{{"no": "C00010"}}
	cust.CalcFieldsForRecords(records, "sales_lcy")
	if _, ok := records[0]["sales_lcy"]; !ok {
		t.Error("sales_lcy not calculated")
	}
	if _, ok := records[0]["balance_lcy"]; ok {
		t.Error("balance_lcy calculated although not requested")
	}
}

// Customer FlowFields read SIFT totals (key customer_open of Customer Ledger Entry). They
// must equal sums over the entries — also after entries are changed through the table API
// (Modify closes an invoice, Delete removes one).
func TestFlowFieldsFromSIFTMatchEntries(t *testing.T) {
	db := newTestDB(t)
	if _, err := Create(db, testCompany, database.DBTypeSQLite, Options{Today: today}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	const entries = `"demo01$Customer Ledger Entry"`

	check := func(label string) {
		t.Helper()
		rows, err := db.Query(`SELECT c.no,
			(SELECT COALESCE(SUM(CAST(remaining_amt_lcy AS REAL)), 0) FROM ` + entries + ` e WHERE e.customer_no = c.no AND e.open = 1),
			(SELECT COALESCE(SUM(CAST(sales_lcy AS REAL)), 0) FROM ` + entries + ` e WHERE e.customer_no = c.no),
			(SELECT COUNT(*) FROM ` + entries + ` e WHERE e.customer_no = c.no)
			FROM "demo01$Customer" c`)
		if err != nil {
			t.Fatal(err)
		}
		want := map[string][3]float64{}
		for rows.Next() {
			var no string
			var balance, sales, count float64
			if err := rows.Scan(&no, &balance, &sales, &count); err != nil {
				t.Fatal(err)
			}
			want[no] = [3]float64{balance, sales, count}
		}
		_ = rows.Close()

		for no, w := range want {
			var cust tables.Customer
			cust.InitWithDBType(db, testCompany, database.DBTypeSQLite)
			if !cust.Get(no) {
				t.Fatalf("customer %s not found", no)
			}
			cust.CalcFields()
			got := [3]float64{cust.Balance_lcy.Float64(), cust.Sales_lcy.Float64(), float64(cust.No_of_ledger_entries)}
			for i := range got {
				if math.Abs(got[i]-w[i]) > 0.005 {
					t.Fatalf("%s: %s FlowFields (balance, sales, count) = %v, want %v", label, no, got, w)
				}
			}
		}
	}
	check("after demo data")

	// Close the first open invoice through the table API (moves it to the closed totals)
	var entry tables.CustomerLedgerEntry
	entry.InitWithDBType(db, testCompany, database.DBTypeSQLite)
	entry.SetRange("open", true)
	if !entry.FindFirst() {
		t.Fatal("no open entry")
	}
	entry.Open = false
	entry.Remaining_amt_lcy = types.NewDecimal(0)
	entry.Remaining_amount = types.NewDecimal(0)
	if !entry.Modify(true) {
		t.Fatal("Modify failed")
	}
	check("after closing an invoice")

	// Delete an entry
	entry.Reset()
	if !entry.FindLast() || !entry.Delete(true) {
		t.Fatal("Delete failed")
	}
	check("after deleting an entry")

	// The totals table is what the FlowFields read: it holds at most 2 rows per customer
	var totalsRows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM "demo01$Customer Ledger Entry$SIFT$customer_open"`).Scan(&totalsRows); err != nil {
		t.Fatal(err)
	}
	if totalsRows == 0 || totalsRows > 40 {
		t.Errorf("totals rows = %d, want 1..40 for 20 customers", totalsRows)
	}
}

// Customer Date Filter (FlowFilter) applied to Posting Date: FlowFields with a date filter
// read the customer_open_date totals and must equal sums over the entries in that period,
// on the card (CalcFields) and in the list (CalcFieldsForRecords); without a filter they
// are unchanged.
func TestDateFilterFlowFields(t *testing.T) {
	db := newTestDB(t)
	if _, err := Create(db, testCompany, database.DBTypeSQLite, Options{Today: today}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	const entries = `"demo01$Customer Ledger Entry"`
	expected := func(no, cond string, args ...interface{}) [3]float64 {
		t.Helper()
		var balance, sales, count float64
		q := `SELECT
			(SELECT COALESCE(SUM(CAST(remaining_amt_lcy AS REAL)), 0) FROM ` + entries + ` WHERE customer_no = ? AND open = 1 AND ` + cond + `),
			(SELECT COALESCE(SUM(CAST(sales_lcy AS REAL)), 0) FROM ` + entries + ` WHERE customer_no = ? AND ` + cond + `),
			(SELECT COUNT(*) FROM ` + entries + ` WHERE customer_no = ? AND ` + cond + `)`
		all := append(append(append([]interface{}{no}, args...), append([]interface{}{no}, args...)...), append([]interface{}{no}, args...)...)
		if err := db.QueryRow(q, all...).Scan(&balance, &sales, &count); err != nil {
			t.Fatal(err)
		}
		return [3]float64{balance, sales, count}
	}
	same := func(label string, got, want [3]float64) {
		t.Helper()
		for i := range got {
			if math.Abs(got[i]-want[i]) > 0.005 {
				t.Errorf("%s: (balance, sales, count) = %v, want %v", label, got, want)
				return
			}
		}
	}

	cases := []struct {
		filter string
		cond   string
		args   []interface{}
	}{
		{"", "1=1", nil},
		{"2026-04-01..2026-06-30", "posting_date BETWEEN ? AND ?", []interface{}{"2026-04-01", "2026-06-30"}},
		{"..2026-03-31", "posting_date <= ?", []interface{}{"2026-03-31"}},
		{"2026-09-01..", "posting_date >= ?", []interface{}{"2026-09-01"}},
		{"2026-01-01..2026-01-31|2026-08-01..2026-08-31", "(posting_date BETWEEN ? AND ? OR posting_date BETWEEN ? AND ?)", []interface{}{"2026-01-01", "2026-01-31", "2026-08-01", "2026-08-31"}},
	}
	for _, tc := range cases {
		// Card path
		for _, no := range []string{"C00010", "C00080", "C00150"} {
			var cust tables.Customer
			cust.InitWithDBType(db, testCompany, database.DBTypeSQLite)
			if !cust.Get(no) {
				t.Fatalf("customer %s not found", no)
			}
			if err := cust.SetFlowFilter("date_filter", tc.filter); err != nil {
				t.Fatalf("SetFlowFilter(%q): %v", tc.filter, err)
			}
			cust.CalcFields()
			got := [3]float64{cust.Balance_lcy.Float64(), cust.Sales_lcy.Float64(), float64(cust.No_of_ledger_entries)}
			same("card "+no+" filter "+tc.filter, got, expected(no, tc.cond, tc.args...))
		}

		// List path (all 20 customers at once)
		var list tables.Customer
		list.InitWithDBType(db, testCompany, database.DBTypeSQLite)
		if err := list.SetFlowFilter("date_filter", tc.filter); err != nil {
			t.Fatal(err)
		}
		var records []map[string]interface{}
		if list.FindSet() {
			records = append(records, list.ToMap())
			for list.Next() {
				records = append(records, list.ToMap())
			}
		}
		list.CalcFieldsForRecords(records)
		for _, rec := range records {
			no := rec["no"].(string)
			var got [3]float64
			for i, field := range []string{"balance_lcy", "sales_lcy", "no_of_ledger_entries"} {
				if _, err := fmt.Sscan(fmt.Sprint(rec[field]), &got[i]); err != nil {
					t.Fatalf("%s %s = %v: %v", no, field, rec[field], err)
				}
			}
			same("list "+no+" filter "+tc.filter, got, expected(no, tc.cond, tc.args...))
		}
	}

	// Invalid expressions and unknown FlowFilter fields are rejected
	var cust tables.Customer
	cust.InitWithDBType(db, testCompany, database.DBTypeSQLite)
	for _, bad := range []string{"31.03.26", "2026-13-01..", "yesterday"} {
		if err := cust.SetFlowFilter("date_filter", bad); err == nil {
			t.Errorf("SetFlowFilter(%q) accepted", bad)
		}
	}
	if err := cust.SetFlowFilter("name", "x"); err == nil {
		t.Error("SetFlowFilter on a normal field accepted")
	}
}

// Renaming a customer (changing its No.) carries the new No. over to its Customer Ledger
// Entries (BC/NAV Rename) — Customer No. and Sell-to Customer No. — and the SIFT totals move
// with them. A rename rolled back with its transaction leaves everything as it was.
func TestRenameCustomerUpdatesLedgerEntries(t *testing.T) {
	db := newTestDB(t)
	if _, err := Create(db, testCompany, database.DBTypeSQLite, Options{Today: today}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	count := func(where string, args ...interface{}) int {
		t.Helper()
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM "demo01$Customer Ledger Entry" WHERE `+where, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	balance := func(no string) float64 {
		t.Helper()
		var c tables.Customer
		c.InitWithDBType(db, testCompany, database.DBTypeSQLite)
		if !c.Get(no) {
			t.Fatalf("customer %s not found", no)
		}
		c.CalcFields()
		return c.Balance_lcy.Float64()
	}
	entries := count("customer_no = ?", "C00010")
	sellTo := count("sell_to_customer_no = ?", "C00010")
	wantBalance := balance("C00010")
	if entries == 0 {
		t.Fatal("C00010 has no entries")
	}

	rename := func(from, to string) *tables.Customer {
		var c tables.Customer
		c.InitWithDBType(db, testCompany, database.DBTypeSQLite)
		if !c.Get(from) {
			t.Fatalf("customer %s not found", from)
		}
		c.No = types.NewCode(to)
		return &c
	}

	// Rolled back: nothing changes (load the record first: the test DB has one connection,
	// which the transaction then holds)
	c := rename("C00010", "C99999")
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	c.SetDB(tx)
	if !c.Modify(true) {
		t.Fatal("Modify in transaction failed")
	}
	_ = tx.Rollback()
	if count("customer_no = ?", "C00010") != entries || count("customer_no = ?", "C99999") != 0 {
		t.Fatal("rolled-back rename changed the ledger entries")
	}

	// Committed rename: entries and totals follow
	c = rename("C00010", "C99999")
	if !c.Modify(true) {
		t.Fatalf("rename failed: %v", c.TriggerError())
	}
	if got := count("customer_no = ?", "C99999"); got != entries {
		t.Errorf("entries on C99999 = %d, want %d", got, entries)
	}
	if got := count("sell_to_customer_no = ?", "C99999"); got != sellTo {
		t.Errorf("sell-to entries on C99999 = %d, want %d", got, sellTo)
	}
	if got := count("customer_no = ? OR sell_to_customer_no = ?", "C00010", "C00010"); got != 0 {
		t.Errorf("%d entries still refer to C00010", got)
	}
	if got := balance("C99999"); math.Abs(got-wantBalance) > 0.005 {
		t.Errorf("balance after rename = %.2f, want %.2f", got, wantBalance)
	}
	var totalsRows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM "demo01$Customer Ledger Entry$SIFT$customer_open" WHERE customer_no = 'C00010'`).Scan(&totalsRows); err != nil || totalsRows != 0 {
		t.Errorf("SIFT totals still on C00010 (%d rows, %v)", totalsRows, err)
	}
}

// HEAVY: 200 customers with about 1,000 entries each over two years (generation only — the
// insert is measured on Postgres, not in unit tests).
func TestGenerateHeavy(t *testing.T) {
	var gen generatorFile
	var setup setupData
	var small customersFile
	for name, out := range map[string]interface{}{"generator.yaml": &gen, "setup.yaml": &setup, "customers.yaml": &small} {
		if err := loadYAML(name, out); err != nil {
			t.Fatal(err)
		}
	}
	rng := newRand()
	customers := append(append([]customerData{}, small.Customers...), generateCustomers(rng, gen, setup, heavyCustomerCount-len(small.Customers))...)
	dueDays := map[string]int{}
	for _, pt := range setup.PaymentTerms {
		dueDays[pt.Code] = pt.DueDays
	}
	entries := generateEntries(rng, customers, dueDays, today, Heavy)
	if len(customers) != 200 {
		t.Fatalf("%d customers", len(customers))
	}
	perCustomer := len(entries) / len(customers)
	if perCustomer < 800 || perCustomer > 1100 {
		t.Errorf("%d entries per customer, want about 1,000", perCustomer)
	}
	oldest := today
	for _, e := range entries {
		if e.date.Before(oldest) {
			oldest = e.date
		}
	}
	if span := today.Sub(oldest).Hours() / 24; span < 700 {
		t.Errorf("entries span %.0f days, want about 730", span)
	}
	if size, err := ParseSize("heavy"); err != nil || size != Heavy {
		t.Errorf("ParseSize(heavy) = %v, %v", size, err)
	}
	t.Logf("HEAVY: %d customers, %d entries", len(customers), len(entries))
}
