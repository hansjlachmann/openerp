package sift

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
)

const testCompany = "TESTSIFT"
const testTable = "Entry"

var testSpec = KeySpec{
	Name:   "acct_open",
	Fields: []Column{{Name: "acct", Kind: KindText}, {Name: "open", Kind: KindBool}},
	Sums:   []Column{{Name: "amount", Kind: KindDecimal}, {Name: "qty", Kind: KindInt}},
}

type testDB struct {
	db     *sql.DB
	dbType database.DBType
}

func (d testDB) entry() string { return testCompany + "$" + testTable }

func (d testDB) exec(t *testing.T, ex database.Executor, query string, args ...any) {
	t.Helper()
	if _, err := ex.Exec(placeholders(d.dbType, query), args...); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
}

// databases returns SQLite (always) and Postgres (when TEST_POSTGRES_DSN is set), each with
// an empty entry table.
func databases(t *testing.T) []testDB {
	t.Helper()
	var out []testDB

	lite, err := sql.Open("sqlite3", "file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lite.Close() })
	out = append(out, testDB{lite, database.DBTypeSQLite})

	if dsn := os.Getenv("TEST_POSTGRES_DSN"); dsn != "" {
		pg, err := sql.Open("postgres", dsn)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = pg.Close() })
		out = append(out, testDB{pg, database.DBTypePostgres})
	}

	for _, d := range out {
		cleanup(d)
		t.Cleanup(func() { cleanup(d) })
		amount, open := "TEXT", "INTEGER"
		if d.dbType == database.DBTypePostgres {
			amount, open = "NUMERIC", "BOOLEAN"
		}
		d.exec(t, d.db, fmt.Sprintf(`CREATE TABLE %s (no INTEGER PRIMARY KEY, acct VARCHAR(20), open %s, amount %s, qty INTEGER, descr VARCHAR(50))`, q(d.entry()), open, amount))
	}
	return out
}

func cleanup(d testDB) {
	key := BuildKey(d.dbType, testCompany, testTable, d.entry(), testSpec)
	for _, stmt := range key.Drop {
		_, _ = d.db.Exec(stmt)
	}
	_, _ = d.db.Exec(`DROP TABLE IF EXISTS ` + q(d.entry()))
	_, _ = d.db.Exec(`DELETE FROM "` + definitionTable + `" WHERE company = '` + testCompany + `'`)
}

type total struct {
	amount float64
	qty    int64
	cnt    int64
}

// expected adds up the entries directly (what the totals must equal).
func expected(t *testing.T, d testDB, sums []string) map[string]total {
	t.Helper()
	cols := "COALESCE(SUM(CAST(amount AS REAL)), 0), COALESCE(SUM(qty), 0)"
	if d.dbType == database.DBTypePostgres {
		cols = "COALESCE(SUM(amount), 0)::float8, COALESCE(SUM(qty), 0)"
	}
	blank := "0"
	if d.dbType == database.DBTypePostgres {
		blank = "FALSE"
	}
	rows, err := d.db.Query(fmt.Sprintf(`SELECT COALESCE(acct, ''), COALESCE(open, %s), %s, COUNT(*) FROM %s GROUP BY 1, 2`, blank, cols, q(d.entry())))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	return scanTotals(t, rows, sums)
}

func actual(t *testing.T, d testDB, sums []string) map[string]total {
	t.Helper()
	cols := "0.0"
	if slices.Contains(sums, "amount") {
		cols = "amount"
		if d.dbType == database.DBTypePostgres {
			cols = "amount::float8"
		}
	}
	if slices.Contains(sums, "qty") {
		cols += ", qty"
	} else {
		cols += ", 0"
	}
	rows, err := d.db.Query(fmt.Sprintf(`SELECT acct, open, %s, cnt FROM %s`, cols, q(TableName(testCompany, testTable, testSpec.Name))))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	return scanTotals(t, rows, sums)
}

func scanTotals(t *testing.T, rows *sql.Rows, sums []string) map[string]total {
	t.Helper()
	out := map[string]total{}
	for rows.Next() {
		var acct string
		var open bool
		var tot total
		if err := rows.Scan(&acct, &open, &tot.amount, &tot.qty, &tot.cnt); err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(sums, "amount") {
			tot.amount = 0
		}
		if !slices.Contains(sums, "qty") {
			tot.qty = 0
		}
		out[fmt.Sprintf("%s|%v", acct, open)] = tot
	}
	return out
}

func assertConsistent(t *testing.T, d testDB, sums []string) {
	t.Helper()
	want, got := expected(t, d, sums), actual(t, d, sums)
	if len(want) != len(got) {
		t.Fatalf("%s: %d totals rows, want %d (no row with cnt 0 may remain)\n got %v\nwant %v", d.dbType, len(got), len(want), got, want)
	}
	for k, w := range want {
		g, ok := got[k]
		if !ok || g.cnt != w.cnt || g.qty != w.qty || math.Abs(g.amount-w.amount) > 0.005 {
			t.Fatalf("%s: totals[%s] = %+v, want %+v", d.dbType, k, g, w)
		}
	}
}

func sumNames(spec KeySpec) []string {
	var out []string
	for _, c := range spec.Sums {
		out = append(out, c.Name)
	}
	return out
}

func syncSpec(t *testing.T, d testDB, spec KeySpec) {
	t.Helper()
	keys := []Key{BuildKey(d.dbType, testCompany, testTable, d.entry(), spec)}
	if err := Sync(d.db, d.dbType, testCompany, testTable, keys); err != nil {
		t.Fatalf("%s Sync: %v", d.dbType, err)
	}
}

// Random inserts, modifies (sums, key fields), deletes and rolled-back transactions keep
// the totals equal to the entries.
func TestTotalsFollowEveryChange(t *testing.T) {
	for _, d := range databases(t) {
		// Entries before the SIFT key exists: the initial build must include them
		for i := 1; i <= 20; i++ {
			d.exec(t, d.db, `INSERT INTO `+q(d.entry())+` (no, acct, open, amount, qty) VALUES (?, ?, ?, ?, ?)`, i, fmt.Sprintf("A%d", i%3), i%2 == 0, fmt.Sprintf("%d.25", i), i)
		}
		syncSpec(t, d, testSpec)
		assertConsistent(t, d, sumNames(testSpec))

		rng := rand.New(rand.NewSource(1))
		next := 21
		for step := range 400 {
			acct := fmt.Sprintf("A%d", rng.Intn(4))
			amount := fmt.Sprintf("%.2f", rng.Float64()*1000-300)
			no := 1 + rng.Intn(next)
			switch op := rng.Intn(10); {
			case op < 4:
				d.exec(t, d.db, `INSERT INTO `+q(d.entry())+` (no, acct, open, amount, qty) VALUES (?, ?, ?, ?, ?)`, next, acct, rng.Intn(2) == 0, amount, rng.Intn(50))
				next++
			case op < 6:
				d.exec(t, d.db, `UPDATE `+q(d.entry())+` SET amount = ?, qty = ? WHERE no = ?`, amount, rng.Intn(50), no)
			case op == 6:
				d.exec(t, d.db, `UPDATE `+q(d.entry())+` SET acct = ?, open = ? WHERE no = ?`, acct, rng.Intn(2) == 0, no) // moves between totals rows
			case op == 7:
				d.exec(t, d.db, `UPDATE `+q(d.entry())+` SET descr = ? WHERE no = ?`, "x", no) // not a SIFT column
			case op == 8:
				d.exec(t, d.db, `DELETE FROM `+q(d.entry())+` WHERE no = ?`, no)
			default:
				// A transaction that is rolled back leaves no trace in the totals
				tx, err := d.db.Begin()
				if err != nil {
					t.Fatal(err)
				}
				d.exec(t, tx, `INSERT INTO `+q(d.entry())+` (no, acct, open, amount, qty) VALUES (?, ?, ?, ?, ?)`, 100000+step, acct, true, "999", 1)
				d.exec(t, tx, `DELETE FROM `+q(d.entry())+` WHERE no = ?`, no)
				_ = tx.Rollback()
			}
		}
		assertConsistent(t, d, sumNames(testSpec))

		// NULL key values count as blank; bulk statements are covered too
		d.exec(t, d.db, `INSERT INTO `+q(d.entry())+` (no, acct, open, amount, qty) VALUES (?, NULL, NULL, NULL, NULL)`, 900000)
		d.exec(t, d.db, `UPDATE `+q(d.entry())+` SET amount = amount WHERE acct = 'A1'`)
		d.exec(t, d.db, `UPDATE `+q(d.entry())+` SET open = NOT open WHERE acct = 'A2'`)
		d.exec(t, d.db, `DELETE FROM `+q(d.entry())+` WHERE acct = 'A3'`)
		assertConsistent(t, d, sumNames(testSpec))

		if d.dbType == database.DBTypePostgres {
			d.exec(t, d.db, `TRUNCATE `+q(d.entry()))
			assertConsistent(t, d, sumNames(testSpec))
		}
	}
}

// Sync rebuilds only when the definition changes; added and removed sum fields and
// removed keys are handled.
func TestSyncRebuildsOnDefinitionChange(t *testing.T) {
	for _, d := range databases(t) {
		for i := 1; i <= 10; i++ {
			d.exec(t, d.db, `INSERT INTO `+q(d.entry())+` (no, acct, open, amount, qty) VALUES (?, ?, ?, ?, ?)`, i, "A", i%2 == 0, "10", 1)
		}
		syncSpec(t, d, testSpec)
		totals := q(TableName(testCompany, testTable, testSpec.Name))

		// Same definition: nothing is rebuilt (a manual change to the totals survives)
		d.exec(t, d.db, `UPDATE `+totals+` SET cnt = 777`)
		syncSpec(t, d, testSpec)
		var cnt int
		if err := d.db.QueryRow(`SELECT MAX(cnt) FROM ` + totals).Scan(&cnt); err != nil || cnt != 777 {
			t.Fatalf("%s: unchanged definition was rebuilt (cnt %d, %v)", d.dbType, cnt, err)
		}

		// A sum field removed → rebuilt without it, correct again
		fewer := KeySpec{Name: testSpec.Name, Fields: testSpec.Fields, Sums: testSpec.Sums[:1]}
		syncSpec(t, d, fewer)
		assertConsistent(t, d, sumNames(fewer))
		if _, err := d.db.Exec(`SELECT qty FROM ` + totals); err == nil {
			t.Fatalf("%s: removed sum field qty still in the totals table", d.dbType)
		}
		d.exec(t, d.db, `INSERT INTO `+q(d.entry())+` (no, acct, open, amount, qty) VALUES (?, ?, ?, ?, ?)`, 50, "B", true, "5", 9)
		assertConsistent(t, d, sumNames(fewer))

		// The sum field added back → rebuilt with it, filled from the entries
		syncSpec(t, d, testSpec)
		assertConsistent(t, d, sumNames(testSpec))

		// Key removed → totals table and triggers dropped; entries still writable
		if err := Sync(d.db, d.dbType, testCompany, testTable, nil); err != nil {
			t.Fatal(err)
		}
		if _, err := d.db.Exec(`SELECT 1 FROM ` + totals); err == nil {
			t.Fatalf("%s: totals table of a removed key still exists", d.dbType)
		}
		d.exec(t, d.db, `INSERT INTO `+q(d.entry())+` (no, acct, open, amount, qty) VALUES (?, ?, ?, ?, ?)`, 51, "B", true, "5", 9)
	}
}

// Parallel transactions posting to the same totals row lose no update (Postgres only:
// SQLite has a single writer).
func TestConcurrentPostingsPostgres(t *testing.T) {
	var pg *testDB
	for _, d := range databases(t) {
		if d.dbType == database.DBTypePostgres {
			d := d
			pg = &d
		}
	}
	if pg == nil {
		t.Skip("TEST_POSTGRES_DSN not set")
	}
	syncSpec(t, *pg, testSpec)

	var wg sync.WaitGroup
	for w := range 8 {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := range 25 {
				tx, err := pg.db.Begin()
				if err != nil {
					t.Error(err)
					return
				}
				no := w*1000 + i
				if _, err := tx.Exec(`INSERT INTO `+q(pg.entry())+` (no, acct, open, amount, qty) VALUES ($1, 'HOT', true, 1.01, 1)`, no); err != nil {
					t.Error(err)
					_ = tx.Rollback()
					return
				}
				if i%5 == 4 {
					_ = tx.Rollback()
				} else if err := tx.Commit(); err != nil {
					t.Error(err)
				}
			}
		}(w)
	}
	wg.Wait()
	assertConsistent(t, *pg, sumNames(testSpec))
	var cnt int
	var amount float64
	if err := pg.db.QueryRow(`SELECT cnt, amount::float8 FROM `+q(TableName(testCompany, testTable, testSpec.Name))+` WHERE acct = 'HOT'`).Scan(&cnt, &amount); err != nil {
		t.Fatal(err)
	}
	if cnt != 160 || math.Abs(amount-161.6) > 1e-9 {
		t.Errorf("HOT totals = %d / %.2f, want 160 / 161.60", cnt, amount)
	}
}

func TestObjectName(t *testing.T) {
	if got := ObjectName("short$Table$SIFT$key"); got != "short$Table$SIFT$key" {
		t.Errorf("short name changed: %s", got)
	}
	long1 := strings.Repeat("a", 40) + "$Customer Ledger Entry$SIFT$customer_open"
	long2 := strings.Repeat("a", 40) + "$Customer Ledger Entry$SIFT$customer_oper"
	n1, n2 := ObjectName(long1), ObjectName(long2)
	if len(n1) > maxIdentifier || len(n2) > maxIdentifier || n1 == n2 {
		t.Errorf("long names: %q (%d) %q (%d)", n1, len(n1), n2, len(n2))
	}
	if ObjectName(long1) != n1 {
		t.Error("ObjectName is not stable")
	}
	// Multi-byte characters are never cut in half
	utf := strings.Repeat("æøå", 30)
	if n := ObjectName(utf); len(n) > maxIdentifier || !strings.HasPrefix(utf, n[:len(n)-9]) {
		t.Errorf("UTF-8 name cut badly: %q", n)
	}
	if TableName("", "Global", "k") != "Global$SIFT$k" {
		t.Errorf("global totals table name = %s", TableName("", "Global", "k"))
	}
}

// A stored definition whose totals table is gone is rebuilt; DropCompany removes all of a
// company's SIFT objects and definitions so a new company with the same name builds again.
func TestMissingTotalsRebuiltAndDropCompany(t *testing.T) {
	for _, d := range databases(t) {
		d.exec(t, d.db, `INSERT INTO `+q(d.entry())+` (no, acct, open, amount, qty) VALUES (?, ?, ?, ?, ?)`, 1, "A", true, "7", 1)
		syncSpec(t, d, testSpec)
		totals := q(TableName(testCompany, testTable, testSpec.Name))

		// Totals table dropped behind SIFT's back → next Sync rebuilds it
		for _, stmt := range BuildKey(d.dbType, testCompany, testTable, d.entry(), testSpec).Drop {
			d.exec(t, d.db, stmt)
		}
		syncSpec(t, d, testSpec)
		assertConsistent(t, d, sumNames(testSpec))

		// Company deleted: objects and definitions gone
		if err := DropCompany(d.db, d.dbType, testCompany); err != nil {
			t.Fatal(err)
		}
		if _, err := d.db.Exec(`SELECT 1 FROM ` + totals); err == nil {
			t.Fatalf("%s: totals table survived DropCompany", d.dbType)
		}
		var n int
		if err := d.db.QueryRow(`SELECT COUNT(*) FROM "` + definitionTable + `" WHERE company = '` + testCompany + `'`).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s: %d definitions left (%v)", d.dbType, n, err)
		}
		// Entries still writable (triggers gone), and a new Sync builds again
		d.exec(t, d.db, `INSERT INTO `+q(d.entry())+` (no, acct, open, amount, qty) VALUES (?, ?, ?, ?, ?)`, 2, "A", true, "3", 1)
		syncSpec(t, d, testSpec)
		assertConsistent(t, d, sumNames(testSpec))
	}
}
