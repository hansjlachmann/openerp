package database

import "database/sql"

// Executor is an interface that both *sql.DB and *sql.Tx implement
// This allows table operations to work with either direct DB connections or transactions
type Executor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Prepare(query string) (*sql.Stmt, error)
}

// Verify at compile time that *sql.DB implements Executor
var _ Executor = (*sql.DB)(nil)

// Verify at compile time that *sql.Tx implements Executor
var _ Executor = (*sql.Tx)(nil)
