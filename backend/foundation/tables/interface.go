package tables

import (
	"github.com/hansjlachmann/openerp/backend/foundation/database"
)

// Table is the interface that all generated tables must implement.
// This enables generic API handlers without hardcoded switch statements.
type Table interface {
	// Identity
	GetTableID() int
	GetTableName() string
	// IsSetupTable reports whether this is a BC-style singleton setup table
	// (a single record identified by a blank primary key)
	IsSetupTable() bool

	// Initialization - must be called before any operations
	Init(db database.Executor, company string)
	// InitWithDBType initializes with explicit database type (required for PostgreSQL)
	InitWithDBType(db database.Executor, company string, dbType database.DBType)
	// InitRecord initializes a new, not yet inserted record (BC/NAV OnNewRecord)
	InitRecord()

	// CRUD operations (BC/NAV style)
	// Get retrieves a record by primary key, returns true if found
	Get(primaryKey interface{}) bool
	// Insert creates a new record, runTrigger controls OnInsert execution
	Insert(runTrigger bool) bool
	// Modify updates the current record, runTrigger controls OnModify execution
	Modify(runTrigger bool) bool
	// Delete removes the current record, runTrigger controls OnDelete execution
	Delete(runTrigger bool) bool
	// TriggerError returns the trigger error that made the last Insert/Modify/Delete
	// fail (a business-rule message for the user), or nil
	TriggerError() error

	// Query operations (BC/NAV style)
	// FindSet prepares iteration over filtered records
	FindSet() bool
	// Next moves to the next record, returns false when exhausted
	// Optional steps parameter allows skipping records (positive) or moving backward (negative)
	Next(steps ...int) bool
	// SetFilter applies a BC-style filter expression to a field
	SetFilter(field, expression string)
	// ClearFilters removes all filters
	ClearFilters()
	// SetCurrentKey sets the sort order by field(s)
	SetCurrentKey(fields ...string)
	// SetAscending sets the sort direction (default ascending). The primary key always
	// follows the current key, so the order is unique and pages never overlap.
	SetAscending(ascending bool)
	// SetSearch keeps the records where any of the fields contains text (case-insensitive)
	SetSearch(fields []string, text string)
	// HasColumn reports whether a field name is a stored column (allowed in filters/sort)
	HasColumn(field string) bool
	// SetPage sets a pagination window for FindSet: at most limit rows, skipping
	// the first offset rows. A limit of 0 returns all matching rows.
	SetPage(limit, offset int)
	// Count returns the number of records matching current filters (ignores pagination)
	Count() int

	// Field operations
	// CalcFields calculates FlowFields (computed fields)
	CalcFields(fields ...string)
	// CalcFieldsForRecords calculates FlowFields for many records at once (list pages):
	// one grouped query per FlowField. records are ToMap() results and get the values
	// set under the field names. With no fields, all FlowFields are calculated.
	CalcFieldsForRecords(records []map[string]interface{}, fields ...string)
	// SetFlowFilter sets a FlowFilter field's filter expression (e.g. Date Filter); FlowFields
	// applying it use it. "" clears it. Errors for unknown fields and invalid expressions.
	SetFlowFilter(field, expr string) error
	// GetFlowFilterFields returns the table's FlowFilter fields (not stored)
	GetFlowFilterFields() []FlowFilterFieldInfo
	// ValidateField validates a single field value
	ValidateField(field string, value interface{}) error

	// Serialization - for API JSON conversion
	// ToMap converts the current record to a map for JSON serialization
	ToMap() map[string]interface{}
	// FromMap populates the record fields from a map
	FromMap(data map[string]interface{})
	// UpdateFromMap updates only the provided fields (for PATCH-style updates)
	UpdateFromMap(data map[string]interface{})

	// Metadata
	// GetPrimaryKeyField returns the name of the primary key field
	GetPrimaryKeyField() string
	// GetPrimaryKeyValue returns the current primary key value as a string
	GetPrimaryKeyValue() string
	// GetFields returns metadata about all fields
	GetFields() []FieldInfo
	// GetFlowFields returns names of FlowFields that need CalcFields
	GetFlowFields() []string
	// GetOptionFields returns Option field names mapped to their option values
	// Returns map[fieldName][]string where each string is an option value (index = stored int value)
	GetOptionFields() map[string][]string
	// GetTableRelationFields returns fields that have table relations (foreign keys)
	// Returns map[fieldName]TableRelationInfo
	GetTableRelationFields() map[string]TableRelationInfo
}

// FlowFilterFieldInfo describes a FlowFilter field (NAV FieldClass FlowFilter): its name and
// value kind ("date", "bool", "int", "text", "code")
type FlowFilterFieldInfo struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// FieldInfo contains metadata about a table field
type FieldInfo struct {
	Name       string    // Field name (snake_case)
	Type       FieldType // Field type
	Length     int       // Max length for Code/Text fields
	Required   bool      // Whether the field is required
	Editable   bool      // Whether the field can be edited
	PrimaryKey bool      // Whether this is the primary key
	FlowField  bool      // Whether this is a computed FlowField
	Sensitive  bool      // Never sent by the API, not filterable/sortable/searchable (YAML sensitive: true)
}

// LookupColumnInfo defines a column to display in the lookup dropdown
type LookupColumnInfo struct {
	Source string // Field name to display
	Width  int    // Column width in pixels (0 = auto)
}

// TableRelationInfo contains metadata about a field's table relation
type TableRelationInfo struct {
	Table         string             // Related table name (e.g., "PaymentTerms")
	Field         string             // Field in related table to match (usually primary key)
	DisplayField  string             // Field to display in dropdown (simple mode)
	LookupColumns []LookupColumnInfo // Columns to show in dropdown (advanced mode)
	SearchTimeout int                // Auto-clear search after N milliseconds (0 = use default 1500)
}

// FieldType represents the BC/NAV field types
type FieldType string

const (
	FieldTypeCode     FieldType = "Code"
	FieldTypeText     FieldType = "Text"
	FieldTypeInteger  FieldType = "Integer"
	FieldTypeDecimal  FieldType = "Decimal"
	FieldTypeBoolean  FieldType = "Boolean"
	FieldTypeDate     FieldType = "Date"
	FieldTypeDateTime FieldType = "DateTime"
	FieldTypeOption   FieldType = "Option"
	FieldTypeBlob     FieldType = "Blob"
)

// TableFactory is a function that creates a new instance of a table
type TableFactory func() Table
