package tables

import (
	"database/sql"
	"sort"
	"strings"
	"sync"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
)

// Consistent: a table can be marked inconsistent within a transaction (record Consistent
// (false), e.g. G/L entries while a posting is not balanced yet); Commit refuses to commit the
// transaction while any table is inconsistent and rolls it back.
var (
	inconsistentMu sync.Mutex
	inconsistent   = map[database.Executor]map[string]bool{}
)

// SetConsistent marks a table consistent or inconsistent in the transaction (or database) exec
func SetConsistent(exec database.Executor, table string, ok bool) {
	inconsistentMu.Lock()
	defer inconsistentMu.Unlock()
	if ok {
		delete(inconsistent[exec], table)
		if len(inconsistent[exec]) == 0 {
			delete(inconsistent, exec)
		}
		return
	}
	if inconsistent[exec] == nil {
		inconsistent[exec] = map[string]bool{}
	}
	inconsistent[exec][table] = true
}

// Commit commits a transaction unless a table was left inconsistent in it (Consistent(false)):
// then it rolls the transaction back and returns an error naming the tables
func Commit(tx *sql.Tx) error {
	tables := clearConsistency(tx)
	if len(tables) > 0 {
		_ = tx.Rollback()
		return apperrors.Inconsistent(strings.Join(tables, ", "))
	}
	return tx.Commit()
}

// Rollback rolls a transaction back and forgets its inconsistent tables
func Rollback(tx *sql.Tx) error {
	clearConsistency(tx)
	return tx.Rollback()
}

// clearConsistency returns the inconsistent tables of exec (sorted) and forgets them
func clearConsistency(exec database.Executor) []string {
	inconsistentMu.Lock()
	defer inconsistentMu.Unlock()
	var tables []string
	for table := range inconsistent[exec] {
		tables = append(tables, table)
	}
	delete(inconsistent, exec)
	sort.Strings(tables)
	return tables
}
