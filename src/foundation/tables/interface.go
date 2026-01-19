package tables

import (
	"github.com/hansjlachmann/openerp/src/foundation/database"
)

// Table is the interface that all generated tables must implement.
// This enables generic API handlers without hardcoded switch statements.
type Table interface {
	// Identity
	GetTableID() int
	GetTableName() string

	// Initialization - must be called before any operations
	Init(db database.Executor, company string)

	// CRUD operations (BC/NAV style)
	// Get retrieves a record by primary key, returns true if found
	Get(primaryKey interface{}) bool
	// Insert creates a new record, runTrigger controls OnInsert execution
	Insert(runTrigger bool) bool
	// Modify updates the current record, runTrigger controls OnModify execution
	Modify(runTrigger bool) bool
	// Delete removes the current record, runTrigger controls OnDelete execution
	Delete(runTrigger bool) bool

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

	// Field operations
	// CalcFields calculates FlowFields (computed fields)
	CalcFields(fields ...string)
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

// FieldInfo contains metadata about a table field
type FieldInfo struct {
	Name       string    // Field name (snake_case)
	Type       FieldType // Field type
	Length     int       // Max length for Code/Text fields
	Required   bool      // Whether the field is required
	Editable   bool      // Whether the field can be edited
	PrimaryKey bool      // Whether this is the primary key
	FlowField  bool      // Whether this is a computed FlowField
}

// TableRelationInfo contains metadata about a field's table relation
type TableRelationInfo struct {
	Table        string // Related table name (e.g., "PaymentTerms")
	Field        string // Field in related table to match (usually primary key)
	DisplayField string // Field to display in dropdown (e.g., "description"), empty means use Field
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
