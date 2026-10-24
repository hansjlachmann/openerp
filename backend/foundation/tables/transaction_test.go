package tables

import (
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
)

func TestBeginTransaction(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tx, err := BeginTransaction(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BeginTransaction(tx); err == nil {
		t.Error("a transaction inside a transaction must be refused")
	}
	SetConsistent(tx, "G_L_Entry", false)
	if err := Commit(tx); err == nil {
		t.Error("commit with an inconsistent table must be refused")
	}
	// LockTable outside Postgres or outside a transaction does nothing
	LockTable(db, database.DBTypeSQLite, "c", "G_L_Entry")
}

// On Postgres LockTable makes a second transaction wait until the first ends (TEST_POSTGRES_DSN)
func TestLockTablePostgres(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	first, err := BeginTransaction(db)
	if err != nil {
		t.Fatal(err)
	}
	LockTable(first, database.DBTypePostgres, "locktest", "G_L_Entry")

	var mu sync.Mutex
	var order []string
	done := make(chan struct{})
	go func() {
		defer close(done)
		second, err := BeginTransaction(db)
		if err != nil {
			t.Error(err)
			return
		}
		LockTable(second, database.DBTypePostgres, "locktest", "G_L_Entry")
		mu.Lock()
		order = append(order, "second")
		mu.Unlock()
		_ = Rollback(second)
	}()
	time.Sleep(300 * time.Millisecond)
	mu.Lock()
	order = append(order, "first")
	mu.Unlock()
	_ = Rollback(first)
	<-done
	if len(order) != 2 || order[0] != "first" {
		t.Errorf("the second lock did not wait: %v", order)
	}
}
