package genjnl

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/hansjlachmann/openerp/backend/business-logic/demodata"
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/company"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// demoCompanyPostgres is a demo company on PostgreSQL (TEST_POSTGRES_DSN), dropped afterwards
func demoCompanyPostgres(t *testing.T) ftables.Source {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	name := fmt.Sprintf("jnltest%d", time.Now().UnixNano()%1000000)
	t.Cleanup(func() {
		tx, err := db.Begin()
		if err == nil {
			_ = company.DropObjects(tx, database.DBTypePostgres, name)
			_ = tx.Commit()
		}
	})
	must(t, tables.CreateCompanyTables(db, name, database.DBTypePostgres))
	_, err = demodata.Create(db, name, database.DBTypePostgres, demodata.Options{Today: time.Date(2026, 10, 6, 0, 0, 0, 0, time.Local)})
	must(t, err)
	return ftables.DBSource(db, name, database.DBTypePostgres)
}

// Posting on PostgreSQL: the posting reads and writes in one transaction (one connection) —
// two batches posted at the same time get distinct entry and register numbers
func TestPostConcurrentPostgres(t *testing.T) {
	src := demoCompanyPostgres(t)
	for _, batchName := range []string{"PARALLEL1", "PARALLEL2"} {
		batch := gtables.NewGenJournalBatch(src)
		batch.Journal_template_name = "GENERAL"
		batch.Name = types.Code(batchName)
		must(t, batch.InsertOrError(true))
		for i := 1; i <= 5; i++ {
			addLine(t, src, "GENERAL", batchName, lineSpec{lineNo: i * 10000, docNo: fmt.Sprintf("%s-%d", batchName, i), accNo: "1210", amount: "10", balNo: "1230"})
		}
	}
	gl0, _, _, reg0 := countEntries(src)
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, batchName := range []string{"PARALLEL1", "PARALLEL2"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = postBatch(t, src, "GENERAL", batchName)
		}()
	}
	wg.Wait()
	for _, err := range errs {
		must(t, err)
	}
	gl1, _, _, reg1 := countEntries(src)
	if gl1-gl0 != 20 || reg1-reg0 != 2 {
		t.Errorf("posted %d G/L entries and %d registers, want 20 and 2", gl1-gl0, reg1-reg0)
	}
	glReg := gtables.NewGLRegister(src)
	glReg.SetFilter("no", fmt.Sprintf(">%d", reg0))
	var ranges [][2]int
	for found := glReg.FindSet(); found; found = glReg.Next() {
		ranges = append(ranges, [2]int{glReg.From_entry_no, glReg.To_entry_no})
	}
	if len(ranges) == 2 && ranges[0][1] >= ranges[1][0] && ranges[1][1] >= ranges[0][0] {
		t.Errorf("the registers overlap: %v", ranges)
	}
}
