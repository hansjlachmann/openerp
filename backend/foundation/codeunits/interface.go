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

	// UsesProgress returns true if this codeunit uses progress dialog updates
	// If true, the codeunit will be run async with SSE progress streaming
	// If false, the codeunit runs synchronously and returns result directly
	UsesProgress() bool
}

// Result represents the result of a codeunit execution
type Result struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Dialog  *DialogResult          `json:"dialog,omitempty"` // Optional dialog to show
}

// DialogResult represents a dialog/message box to display
type DialogResult struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Type    string `json:"type"` // "info", "success", "warning", "error"
}

// Factory is a function that creates a new codeunit instance
type Factory func(db database.Executor, company string, dbType database.DBType) Codeunit
