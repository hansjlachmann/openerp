package tables

import (
	"database/sql"
	"fmt"
	"runtime"
	"sync/atomic"

	_ "github.com/mattn/go-sqlite3" // the in-memory databases of temporary records
)

// TemporaryStore is the private in-memory database of a temporary record (Temporary): the
// record's own table lives there, so Insert/Modify/Delete/Find/filters/keys work as usual and
// nothing reaches the real database. One connection, so the in-memory database stays the
// same; it is closed when the record is no longer used.
type TemporaryStore struct {
	DB *sql.DB
}

var temporaryStores atomic.Int64

// NewTemporaryStore opens the in-memory database of a temporary record
func NewTemporaryStore() (*TemporaryStore, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:temporary%d?mode=memory", temporaryStores.Add(1)))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	store := &TemporaryStore{DB: db}
	runtime.SetFinalizer(store, func(s *TemporaryStore) { _ = s.DB.Close() })
	return store, nil
}
