package codeunits

import "github.com/hansjlachmann/openerp/backend/foundation/database"

// Codeunit is the interface that all codeunits must implement
type Codeunit interface {
	// ID returns the unique codeunit ID (e.g., 50010)
	ID() int

	// Name returns the codeunit name (e.g., "generate-cust-ledger-entries")
	Name() string

	// SourceTable returns the name of the table this codeunit operates on (e.g., "Customer")
	SourceTable() string

	// Run executes the codeunit with the given record
	// The record should be a pointer to the appropriate table struct
	Run(record interface{}) (Result, error)
}

// Result represents the result of a codeunit execution
type Result struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// Factory is a function that creates a new codeunit instance
type Factory func(db database.Executor, company string, dbType database.DBType) Codeunit
