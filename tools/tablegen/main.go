package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// TableDef represents a table definition from YAML
type TableDef struct {
	Table struct {
		ID     int     `yaml:"id"`
		Name   string  `yaml:"name"`
		Global bool    `yaml:"global"` // If true, table is global (no company prefix)
		Fields []Field `yaml:"fields"`
		Keys   []Key   `yaml:"keys"`
	} `yaml:"table"`
}

// Key represents an index/key on a table (BC/NAV style)
type Key struct {
	Name      string   `yaml:"name"`       // Key name (e.g., "customer_open")
	Fields    []string `yaml:"fields"`     // Fields in the key (e.g., ["customer_no", "open"])
	Unique    bool     `yaml:"unique"`     // Whether this is a UNIQUE index
	Clustered bool     `yaml:"clustered"`  // Primary key-like behavior (BC/NAV concept)
}

// Field represents a single field in a table
type Field struct {
	Name          string         `yaml:"name"`
	Type          string         `yaml:"type"`
	DBName        string         `yaml:"db_name"`
	PrimaryKey    bool           `yaml:"primary_key"`
	Length        int            `yaml:"length"`
	Required      bool           `yaml:"required"`
	Default       interface{}    `yaml:"default"`
	AutoTimestamp bool           `yaml:"auto_timestamp"`
	Validation    *Validation    `yaml:"validation"`
	TableRelation *TableRelation `yaml:"table_relation"`
	Options       []string       `yaml:"options"`   // For Option type fields (enum values)
	Precision     int            `yaml:"precision"` // For Decimal type (total digits)
	Scale         int            `yaml:"scale"`     // For Decimal type (decimal places)
	FlowField     bool           `yaml:"flow_field"` // For FlowFields (calculated fields)
	CalcFormula   string         `yaml:"calc_formula"` // Sum, Count, Lookup, Exist, Average, Min, Max
	SourceTable   string         `yaml:"source_table"` // Table to calculate from
	SourceField   string         `yaml:"source_field"` // Field to aggregate
	FlowFilters   []FlowFilter   `yaml:"flow_filters"` // Filter conditions
}

// FlowFilter represents a filter condition for FlowField calculation
type FlowFilter struct {
	Field     string `yaml:"field"`      // Field name in source table
	Type      string `yaml:"type"`       // "const" or "field"
	Value     string `yaml:"value"`      // Constant value or field name from current table
}

// LookupColumn defines a column to display in the lookup dropdown
type LookupColumn struct {
	Source string `yaml:"source"` // Field name to display
	Width  int    `yaml:"width"`  // Column width in pixels (optional)
}

// TableRelation represents a foreign key relationship to another table
type TableRelation struct {
	Table         string         `yaml:"table"`
	Field         string         `yaml:"field"`
	DisplayField  string         `yaml:"display_field"`  // Field to show in dropdown (e.g., "description") - simple mode
	LookupColumns []LookupColumn `yaml:"lookup_columns"` // Columns to show in dropdown - advanced mode
	SearchTimeout int            `yaml:"search_timeout"` // Auto-clear search after N milliseconds (default: 1500)
	Validate      *bool          `yaml:"validate"`       // Whether to validate the relation (default: true)
}

// ShouldValidate returns whether this table relation should be validated
func (tr *TableRelation) ShouldValidate() bool {
	if tr.Validate == nil {
		return true // default is true
	}
	return *tr.Validate
}

// Validation represents field validation rules
type Validation struct {
	Min interface{} `yaml:"min"`
	Max interface{} `yaml:"max"`
}

// TemplateData is the data passed to templates
type TemplateData struct {
	TableDef
	StructName       string
	BaseStructName   string // For generated base struct (StructName + "Base")
	PackageName      string
	GeneratedPkg     string // Import alias for generated package (e.g., "gtables")
	HasTimeField     bool
	HasCodeField     bool
	HasTextField     bool
	HasOptionField   bool
	HasDecimalField  bool
	HasDateField     bool
	HasDateTimeField bool
	HasFlowField     bool
	HasBlobField     bool
	HasIntField      bool
	FirstPrimaryKey  *Field // First primary key field (for GetPrimaryKeyField/Value)
}

func main() {
	// Get current working directory (should be business-logic/tables when run via go generate)
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// The definitions directory is in the same directory as where go generate is run
	defsDir := filepath.Join(cwd, "definitions")

	// Output directory for generated base files (backend/generated/tables/)
	// Go up from business-logic/tables to backend, then into generated/tables
	generatedDir := filepath.Join(cwd, "..", "..", "generated", "tables")

	// Ensure directories exist
	if err := os.MkdirAll(defsDir, 0755); err != nil {
		fmt.Printf("Error creating definitions directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(generatedDir, 0755); err != nil {
		fmt.Printf("Error creating generated directory: %v\n", err)
		os.Exit(1)
	}

	// Find all YAML definition files
	yamlFiles, err := filepath.Glob(filepath.Join(defsDir, "*.yaml"))
	if err != nil {
		fmt.Printf("Error finding YAML files: %v\n", err)
		os.Exit(1)
	}

	if len(yamlFiles) == 0 {
		fmt.Println("No YAML definition files found in", defsDir)
		return
	}

	fmt.Printf("Found %d table definition(s)\n", len(yamlFiles))

	for _, yamlFile := range yamlFiles {
		fmt.Printf("\nProcessing: %s\n", filepath.Base(yamlFile))

		// Parse YAML
		tableDef, err := parseYAML(yamlFile)
		if err != nil {
			fmt.Printf("  ✗ Error parsing YAML: %v\n", err)
			continue
		}

		// Prepare template data
		data := prepareTemplateData(tableDef)

		// Generate *_base.go to generated directory (always regenerate)
		baseFile := filepath.Join(generatedDir, strings.ToLower(data.StructName)+"_base.go")
		if err := generateBoilerplate(baseFile, data); err != nil {
			fmt.Printf("  ✗ Error generating base: %v\n", err)
			continue
		}
		fmt.Printf("  ✓ Generated: %s\n", filepath.Base(baseFile))

		// Generate *.go wrapper skeleton in business-logic/tables (only if doesn't exist)
		wrapperFile := filepath.Join(cwd, strings.ToLower(data.StructName)+".go")
		if !fileExists(wrapperFile) {
			if err := generateBusinessLogicSkeleton(wrapperFile, data); err != nil {
				fmt.Printf("  ✗ Error generating wrapper: %v\n", err)
				continue
			}
			fmt.Printf("  ✓ Created wrapper: %s\n", filepath.Base(wrapperFile))
		} else {
			fmt.Printf("  ⊙ Skipped (exists): %s\n", filepath.Base(wrapperFile))
		}
	}

	fmt.Println("\n✓ Code generation complete!")
}

// parseYAML reads and parses a YAML definition file
func parseYAML(filename string) (*TableDef, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var def TableDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, err
	}

	// Auto-fill db_name if not specified
	for i := range def.Table.Fields {
		if def.Table.Fields[i].DBName == "" {
			def.Table.Fields[i].DBName = toSnakeCase(def.Table.Fields[i].Name)
		}
	}

	return &def, nil
}

// prepareTemplateData creates template data from table definition
func prepareTemplateData(def *TableDef) TemplateData {
	structName := toPascalCase(def.Table.Name)
	data := TemplateData{
		TableDef:       *def,
		StructName:     structName,
		BaseStructName: structName + "Base",
		PackageName:    "tables",
		GeneratedPkg:   "gtables",
	}

	// Check which imports are needed and find first primary key
	for i := range def.Table.Fields {
		field := &def.Table.Fields[i]
		if field.Type == "time.Time" {
			data.HasTimeField = true
		}
		if field.Type == "types.Code" {
			data.HasCodeField = true
		}
		if field.Type == "types.Text" {
			data.HasTextField = true
		}
		if field.Type == "Option" {
			data.HasOptionField = true
		}
		if field.Type == "types.Decimal" {
			data.HasDecimalField = true
		}
		if field.Type == "types.Date" {
			data.HasDateField = true
		}
		if field.Type == "types.DateTime" {
			data.HasDateTimeField = true
		}
		if field.FlowField {
			data.HasFlowField = true
		}
		if field.Type == "[]byte" || field.Type == "BLOB" {
			data.HasBlobField = true
		}
		if field.Type == "int" && !field.FlowField {
			data.HasIntField = true
		}
		// Track first primary key field (for tables with composite keys)
		if field.PrimaryKey && data.FirstPrimaryKey == nil {
			data.FirstPrimaryKey = field
		}
	}

	return data
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// generateBoilerplate generates the *_gen.go file
func generateBoilerplate(filename string, data TemplateData) error {
	tmpl, err := template.New("gen").Funcs(templateFuncs()).Parse(boilerplateTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

// generateBusinessLogicSkeleton generates the *.go skeleton file
func generateBusinessLogicSkeleton(filename string, data TemplateData) error {
	tmpl, err := template.New("business").Funcs(templateFuncs()).Parse(businessTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

// getTableNameExpr returns the Go code expression for getting the table name
// For global tables: just the table name constant
// For company tables: company prefix + table name
func getTableNameExpr(isGlobal bool, structName, companyVar string) string {
	if isGlobal {
		return fmt.Sprintf("%sTableName", structName)
	}
	return fmt.Sprintf(`fmt.Sprintf("%%s$%%s", %s, %sTableName)`, companyVar, structName)
}

// templateFuncs returns template helper functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"upperFirst":         upperFirst,
		"lowerFirst":         lowerFirst,
		"sqlType":            getSQLType,
		"postgresSqlType":    getPostgresSQLType,
		"tableName":          getTableNameExpr,
		"isLast":             isLast,
		"isLastPK":           isLastPK,
		"isLastDBField":      isLastDBField,
		"hasSuffix":          strings.HasSuffix,
		"join":               strings.Join,
		"sub":                func(a, b int) int { return a - b },
		"sanitizeIdentifier": sanitizeIdentifier,
		"pkCount":            countPrimaryKeys,
		"firstPK":            getFirstPK,
		"toPascalCase":       toPascalCase,
		"tableNameVar":       getTableNameVarCode,
	}
}

// getTableNameVarCode returns the Go code to set the tableName variable
// For global tables: tableName := StructNameTableName
// For company tables: tableName := fmt.Sprintf("%s$%s", t.company, StructNameTableName)
func getTableNameVarCode(isGlobal bool, structName, companyVar string) string {
	if isGlobal {
		return fmt.Sprintf("tableName := %sTableName", structName)
	}
	return fmt.Sprintf(`tableName := fmt.Sprintf("%%s$%%s", %s, %sTableName)`, companyVar, structName)
}

// countPrimaryKeys counts the number of primary key fields
func countPrimaryKeys(fields []Field) int {
	count := 0
	for _, f := range fields {
		if f.PrimaryKey {
			count++
		}
	}
	return count
}

// getFirstPK returns the first primary key field (or nil if none)
func getFirstPK(fields []Field) *Field {
	for i := range fields {
		if fields[i].PrimaryKey {
			return &fields[i]
		}
	}
	return nil
}

// Helper functions

func toPascalCase(s string) string {
	// Remove special characters and split by spaces
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	words := strings.Fields(s)

	for i, word := range words {
		words[i] = upperFirst(word)
	}

	return strings.Join(words, "")
}

func sanitizeIdentifier(s string) string {
	// Convert option values like "Credit Memo", "G/L Account", etc. into valid Go identifiers
	// Remove or replace special characters
	s = strings.ReplaceAll(s, "/", "")  // "G/L Account" -> "GL Account"
	s = strings.ReplaceAll(s, "-", " ") // Hyphens to spaces
	s = strings.ReplaceAll(s, "_", " ") // Underscores to spaces

	// Split by spaces and capitalize each word
	words := strings.Fields(s)
	for i, word := range words {
		words[i] = upperFirst(word)
	}

	result := strings.Join(words, "")

	// Handle blank option (empty string or single space)
	if result == "" || s == " " {
		return "Blank"
	}

	return result
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func upperFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func lowerFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func isLast(index int, slice []Field) bool {
	return index == len(slice)-1
}

func isLastPK(index int, slice []Field) bool {
	// Check if this field is a PK and if there are any more PK fields after it
	if !slice[index].PrimaryKey {
		return false
	}

	// Look for any PK fields after this index
	for i := index + 1; i < len(slice); i++ {
		if slice[i].PrimaryKey {
			return false // Found another PK after this one
		}
	}

	return true // This is the last PK field
}

func isLastDBField(index int, slice []Field) bool {
	// Check if this is the last non-FlowField
	// Look for any more non-FlowFields after this index
	for i := index + 1; i < len(slice); i++ {
		if !slice[i].FlowField {
			return false // Found another DB field after this one
		}
	}

	return true // This is the last DB field
}

func getSQLType(f Field) string {
	switch f.Type {
	case "types.Code", "types.Text", "string":
		if f.Length > 0 {
			return fmt.Sprintf("TEXT(%d)", f.Length)
		}
		return "TEXT"
	case "int", "int64":
		return "INTEGER"
	case "float64":
		return "REAL"
	case "bool":
		return "INTEGER"
	case "time.Time":
		return "TEXT"
	case "Option":
		return "INTEGER"
	case "types.Decimal":
		return "TEXT" // Store as TEXT for exact decimal representation
	case "types.Date":
		return "TEXT" // Store as TEXT in "YYYY-MM-DD" format
	case "types.DateTime":
		return "TEXT" // Store as TEXT in ISO 8601 format
	case "BLOB", "[]byte":
		return "BLOB"
	default:
		return "TEXT"
	}
}

func getPostgresSQLType(f Field) string {
	switch f.Type {
	case "types.Code", "types.Text", "string":
		if f.Length > 0 {
			return fmt.Sprintf("VARCHAR(%d)", f.Length)
		}
		return "TEXT"
	case "int", "int64":
		return "INTEGER"
	case "float64":
		return "DOUBLE PRECISION"
	case "bool":
		return "BOOLEAN"
	case "time.Time":
		return "TIMESTAMP"
	case "Option":
		return "INTEGER"
	case "types.Decimal":
		return "NUMERIC" // PostgreSQL NUMERIC for exact decimal representation
	case "types.Date":
		return "DATE"
	case "types.DateTime":
		return "TIMESTAMP"
	case "BLOB", "[]byte":
		return "BYTEA"
	default:
		return "TEXT"
	}
}

func getSQLConstraints(f Field) string {
	var constraints []string

	if f.PrimaryKey {
		constraints = append(constraints, "PRIMARY KEY")
	}

	if f.Required && !f.PrimaryKey {
		constraints = append(constraints, "NOT NULL")
	}

	if f.Validation != nil {
		if f.Validation.Min != nil && f.Validation.Max != nil {
			constraints = append(constraints, fmt.Sprintf("CHECK (%s >= %v AND %s <= %v)",
				f.DBName, f.Validation.Min, f.DBName, f.Validation.Max))
		}
	}

	// Option fields need CHECK constraint for valid range
	if f.Type == "Option" && len(f.Options) > 0 {
		maxValue := len(f.Options) - 1
		constraints = append(constraints, fmt.Sprintf("CHECK (%s >= 0 AND %s <= %d)",
			f.DBName, f.DBName, maxValue))
	}

	if f.Default != nil {
		constraints = append(constraints, fmt.Sprintf("DEFAULT %v", f.Default))
	}

	if f.AutoTimestamp {
		constraints = append(constraints, "DEFAULT CURRENT_TIMESTAMP")
	}

	if len(constraints) > 0 {
		return " " + strings.Join(constraints, " ")
	}
	return ""
}

// Templates

const boilerplateTemplate = `// Code generated by tablegen. DO NOT EDIT.

package {{ .PackageName }}

import (
	"database/sql"
{{- if .HasBlobField }}
	"encoding/base64"
{{- end }}
	"fmt"
{{- if .HasIntField }}
	"strconv"
{{- end }}
	"strings"
{{- if or .HasTimeField .HasDateField .HasDateTimeField }}
	"time"
{{- end }}

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/i18n"
	"github.com/hansjlachmann/openerp/backend/foundation/tables"
{{- if or .HasCodeField .HasTextField .HasDecimalField .HasDateField .HasDateTimeField }}
	"github.com/hansjlachmann/openerp/backend/foundation/types"
{{- end }}
)

{{- if .HasOptionField }}

// ========================================
// Option Field Type Definitions (BC/NAV style)
// ========================================
{{- range .Table.Fields }}
{{- if eq .Type "Option" }}

// {{ $.StructName }}{{ upperFirst .Name }} represents the {{ .Name }} option field
type {{ $.StructName }}{{ upperFirst .Name }} int

// String returns the text representation of {{ $.StructName }}{{ upperFirst .Name }}
func (o {{ $.StructName }}{{ upperFirst .Name }}) String() string {
	options := []string{ {{- range $i, $opt := .Options }}{{- if $i }}, {{ end }}"{{ $opt }}"{{- end }} }
	if o >= 0 && int(o) < len(options) {
		return options[o]
	}
	return ""
}

// IsValid checks if the {{ $.StructName }}{{ upperFirst .Name }} value is within valid range
func (o {{ $.StructName }}{{ upperFirst .Name }}) IsValid() bool {
	return o >= 0 && o < {{ len .Options }}
}
{{- end }}
{{- end }}
{{- end }}

// {{ .BaseStructName }} represents Table {{ .Table.ID }}: {{ .Table.Name }}
// This is the generated base struct - embed in your wrapper struct and override Init
type {{ .BaseStructName }} struct {
{{- range .Table.Fields }}
{{- if .FlowField }}
	// FlowField: {{ .CalcFormula }}({{ .SourceTable }}.{{ .SourceField }})
	{{ upperFirst .Name }} {{ .Type }}
{{- else if eq .Type "Option" }}
	{{ upperFirst .Name }} {{ $.StructName }}{{ upperFirst .Name }} ` + "`db:\"{{ .DBName }}{{if .PrimaryKey}},pk{{end}}\"`" + `
{{- else }}
	{{ upperFirst .Name }} {{ .Type }} ` + "`db:\"{{ .DBName }}{{if .PrimaryKey}},pk{{end}}\"`" + `
{{- end }}
{{- end }}

	// Internal context (set by Init)
	db      database.Executor
	company string
	dbType  database.DBType

	// Field tracking for optimal Modify() operations
	oldValues map[string]interface{} // Stores original values from Get()

	// Filter state for SetRange/FindFirst/FindLast (BC/NAV style)
	filters map[string]*{{ lowerFirst .BaseStructName }}FilterCondition

	// Iteration state for FindSet/Next (BC/NAV style)
	currentRows *sql.Rows
	orderByFields []string

	// Buffered recordset for bidirectional navigation (BC/NAV style)
	bufferedRecords []*{{ .BaseStructName }}
	currentBufferPos int

	// Trigger function references (set by wrapper struct via SetTriggers)
	onInsertFn func() error
	onModifyFn func() error
	onDeleteFn func(database.Executor, string) error
}

const {{ .StructName }}TableID = {{ .Table.ID }}
const {{ .StructName }}TableName = "{{ .Table.Name }}"

{{- if .HasOptionField }}

// ========================================
// Option Field Namespaces (BC/NAV style)
// ========================================
{{- range .Table.Fields }}
{{- if eq .Type "Option" }}

// {{ $.StructName }}_{{ upperFirst .Name }} provides named constants for the {{ .Name }} option field (FieldName.OptionValue syntax)
var {{ $.StructName }}_{{ upperFirst .Name }} = struct {
{{- $fieldName := .Name }}
{{- range $i, $opt := .Options }}
	{{ sanitizeIdentifier $opt }}    {{ $.StructName }}{{ upperFirst $fieldName }}
{{- end }}
}{
{{- range $i, $opt := .Options }}
	{{ sanitizeIdentifier $opt }}:    {{ $i }},
{{- end }}
}
{{- end }}
{{- end }}
{{- end }}

// GetTableID returns the table ID (for Object Registry)
func (t *{{ .BaseStructName }}) GetTableID() int {
	return {{ .StructName }}TableID
}

// GetTableName returns the table name
func (t *{{ .BaseStructName }}) GetTableName() string {
	return {{ .StructName }}TableName
}

// GetTableSchema returns the CREATE TABLE schema (SQLite)
func (t *{{ .BaseStructName }}) GetTableSchema() string {
	return Get{{ .StructName }}TableSchema()
}

// GetPostgresTableSchema returns the CREATE TABLE schema (PostgreSQL)
func (t *{{ .BaseStructName }}) GetPostgresTableSchema() string {
	return Get{{ .StructName }}PostgresTableSchema()
}

// SetTriggers sets the trigger function references (called by wrapper Init)
func (t *{{ .BaseStructName }}) SetTriggers(onInsert, onModify func() error, onDelete func(database.Executor, string) error) {
	t.onInsertFn = onInsert
	t.onModifyFn = onModify
	t.onDeleteFn = onDelete
}

// GetDB returns the database executor (for wrapper access)
func (t *{{ .BaseStructName }}) GetDB() database.Executor {
	return t.db
}

// GetCompany returns the company name (for wrapper access)
func (t *{{ .BaseStructName }}) GetCompany() string {
	return t.company
}

// Get{{ .StructName }}TableSchema returns the SQLite schema
func Get{{ .StructName }}TableSchema() string {
	return ` + "`" + `
{{- range $i, $f := .Table.Fields }}
		{{ $f.DBName }} {{ sqlType $f }}{{ if and $f.PrimaryKey (eq (pkCount $.Table.Fields) 1) }} PRIMARY KEY{{ end }}{{ if $f.Required }}{{ if not $f.PrimaryKey }} NOT NULL{{ end }}{{ end }}{{ if $f.Validation }} CHECK ({{ $f.DBName }} >= {{ $f.Validation.Min }} AND {{ $f.DBName }} <= {{ $f.Validation.Max }}){{ end }}{{ if eq $f.Type "Option" }} CHECK ({{ $f.DBName }} >= 0 AND {{ $f.DBName }} <= {{ sub (len $f.Options) 1 }}){{ end }}{{ if $f.Default }} DEFAULT {{ $f.Default }}{{ end }}{{ if $f.AutoTimestamp }} DEFAULT CURRENT_TIMESTAMP{{ end }}{{ if or (not (isLast $i $.Table.Fields)) (gt (pkCount $.Table.Fields) 1) }},{{ end }}
{{- end }}
{{- if gt (pkCount .Table.Fields) 1 }}
		PRIMARY KEY ({{ range $i, $f := .Table.Fields }}{{ if $f.PrimaryKey }}{{ $f.DBName }}{{ if not (isLastPK $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }})
{{- end }}
	` + "`" + `
}

// Get{{ .StructName }}PostgresTableSchema returns the PostgreSQL schema
func Get{{ .StructName }}PostgresTableSchema() string {
	return ` + "`" + `
{{- range $i, $f := .Table.Fields }}
		{{ $f.DBName }} {{ postgresSqlType $f }}{{ if and $f.PrimaryKey (eq (pkCount $.Table.Fields) 1) }} PRIMARY KEY{{ end }}{{ if $f.Required }}{{ if not $f.PrimaryKey }} NOT NULL{{ end }}{{ end }}{{ if $f.Validation }} CHECK ({{ $f.DBName }} >= {{ $f.Validation.Min }} AND {{ $f.DBName }} <= {{ $f.Validation.Max }}){{ end }}{{ if eq $f.Type "Option" }} CHECK ({{ $f.DBName }} >= 0 AND {{ $f.DBName }} <= {{ sub (len $f.Options) 1 }}){{ end }}{{ if $f.Default }} DEFAULT {{ $f.Default }}{{ end }}{{ if $f.AutoTimestamp }} DEFAULT CURRENT_TIMESTAMP{{ end }}{{ if or (not (isLast $i $.Table.Fields)) (gt (pkCount $.Table.Fields) 1) }},{{ end }}
{{- end }}
{{- if gt (pkCount .Table.Fields) 1 }}
		PRIMARY KEY ({{ range $i, $f := .Table.Fields }}{{ if $f.PrimaryKey }}{{ $f.DBName }}{{ if not (isLastPK $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }})
{{- end }}
	` + "`" + `
}

// ========================================
// Translation Support (BC/NAV CaptionML)
// ========================================

// GetCaption returns the table caption in the specified language
func (t *{{ .BaseStructName }}) GetCaption(language string) string {
	ts := i18n.GetInstance()
	return ts.TableCaption("{{ .Table.Name }}", language)
}

// GetFieldCaption returns the field caption in the specified language
func (t *{{ .BaseStructName }}) GetFieldCaption(fieldName, language string) string {
	ts := i18n.GetInstance()
	return ts.FieldCaption("{{ .Table.Name }}", fieldName, language)
}
{{- if .HasOptionField }}

// GetOptionCaption returns the option field value caption in the specified language
func (t *{{ .BaseStructName }}) GetOptionCaption(fieldName, optionValue, language string) string {
	ts := i18n.GetInstance()
	return ts.OptionCaption("{{ .Table.Name }}", fieldName, optionValue, language)
}
{{- end }}

// CreateTable creates the {{ .Table.Name }} table for the specified company (SQLite)
// The db parameter can be either *sql.DB or *sql.Tx
func (t *{{ .BaseStructName }}) CreateTable(db database.Executor, company string) error {
	return t.CreateTableWithDBType(db, company, database.DBTypeSQLite)
}

// CreateTableWithDBType creates the {{ .Table.Name }} table for the specified company with the given database type
// The db parameter can be either *sql.DB or *sql.Tx
func (t *{{ .BaseStructName }}) CreateTableWithDBType(db database.Executor, company string, dbType database.DBType) error {
{{- if .Table.Global }}
	tableName := {{ .StructName }}TableName
{{- else }}
	tableName := fmt.Sprintf("%s$%s", company, {{ .StructName }}TableName)
{{- end }}
	var schema string
	if dbType == database.DBTypePostgres {
		schema = Get{{ .StructName }}PostgresTableSchema()
	} else {
		schema = Get{{ .StructName }}TableSchema()
	}

	createSQL := fmt.Sprintf(` + "`CREATE TABLE IF NOT EXISTS \"%s\" (%s)`" + `, tableName, schema)
	_, err := db.Exec(createSQL)
	if err != nil {
		return fmt.Errorf("failed to create {{ .Table.Name }} table: %w", err)
	}

	// Create indexes (BC/NAV Keys)
{{- if .Table.Keys }}
	var indexName, indexSQL string
{{- range .Table.Keys }}
	indexName = fmt.Sprintf("%s${{ $.Table.Name }}${{ .Name }}", company)
	indexSQL = fmt.Sprintf(` + "`CREATE INDEX IF NOT EXISTS \"%s\" ON \"%s\" ({{ join .Fields \", \" }})`" + `,
		indexName, tableName)
	_, err = db.Exec(indexSQL)
	if err != nil {
		return fmt.Errorf("failed to create index {{ .Name }}: %w", err)
	}
{{- end }}
{{- end }}

	return nil
}

// ========================================
// BC/NAV-style Record Methods
// ========================================

// Init initializes a new {{ .StructName }} record with database context
// The db parameter can be either *sql.DB or *sql.Tx, allowing operations
// to work seamlessly with or without explicit transactions
func (t *{{ .BaseStructName }}) Init(db database.Executor, company string) {
	t.InitWithDBType(db, company, database.DBTypeSQLite)
}

// InitWithDBType initializes a new {{ .StructName }} record with database context and type
func (t *{{ .BaseStructName }}) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.db = db
	t.company = company
	t.dbType = dbType
	t.oldValues = nil // Fresh record, no old values

{{- range .Table.Fields }}
{{- if .AutoTimestamp }}
	t.{{ upperFirst .Name }} = time.Now()
{{- else if .Default }}
	t.{{ upperFirst .Name }} = {{ .Default }}
{{- end }}
{{- end }}
}

// StoreOldValues stores current field values for change detection
// Call this after loading a record from the database
func (t *{{ .BaseStructName }}) StoreOldValues() {
	t.oldValues = make(map[string]interface{})
{{- range .Table.Fields }}
{{- if not .FlowField }}
	t.oldValues["{{ .DBName }}"] = t.{{ upperFirst .Name }}
{{- end }}
{{- end }}
}

// convertPlaceholders converts SQLite-style ? placeholders to PostgreSQL-style $1, $2, etc.
// when running on PostgreSQL
func (t *{{ .BaseStructName }}) convertPlaceholders(sql string, count int) string {
	if t.dbType != database.DBTypePostgres {
		return sql
	}
	result := sql
	for i := 1; i <= count; i++ {
		result = strings.Replace(result, "?", fmt.Sprintf("$%d", i), 1)
	}
	return result
}

// Get retrieves a record from the database by primary key (interface{} for generic API)
// For single primary key: pass the value directly (string, int, etc.)
// For composite keys: pass a map[string]interface{} with field names as keys
func (t *{{ .BaseStructName }}) Get(primaryKey interface{}) bool {
	// Handle composite primary key (map[string]interface{})
	if pkMap, ok := primaryKey.(map[string]interface{}); ok {
{{- range $i, $f := .Table.Fields }}{{- if $f.PrimaryKey }}
		var {{ lowerFirst $f.Name }}Val {{ $f.Type }}
		if v, exists := pkMap["{{ $f.DBName }}"]; exists {
{{- if eq $f.Type "types.Code" }}
			switch val := v.(type) {
			case types.Code:
				{{ lowerFirst $f.Name }}Val = val
			case string:
				{{ lowerFirst $f.Name }}Val = types.NewCode(val)
			}
{{- else if eq $f.Type "types.Text" }}
			switch val := v.(type) {
			case types.Text:
				{{ lowerFirst $f.Name }}Val = val
			case string:
				{{ lowerFirst $f.Name }}Val = types.NewText(val)
			}
{{- else if eq $f.Type "int" }}
			switch val := v.(type) {
			case int:
				{{ lowerFirst $f.Name }}Val = val
			case float64:
				{{ lowerFirst $f.Name }}Val = int(val)
			}
{{- else if eq $f.Type "int64" }}
			switch val := v.(type) {
			case int64:
				{{ lowerFirst $f.Name }}Val = val
			case int:
				{{ lowerFirst $f.Name }}Val = int64(val)
			case float64:
				{{ lowerFirst $f.Name }}Val = int64(val)
			}
{{- else if eq $f.Type "string" }}
			if val, ok := v.(string); ok {
				{{ lowerFirst $f.Name }}Val = val
			}
{{- end }}
		}
{{- end }}{{- end }}
		return t.GetByPK({{- range $i, $f := .Table.Fields }}{{- if $f.PrimaryKey }}{{ lowerFirst $f.Name }}Val{{ if not (isLastPK $i $.Table.Fields) }}, {{ end }}{{- end }}{{- end }})
	}

{{- if eq (pkCount .Table.Fields) 1 }}
	// Handle single primary key (for tables with only one PK field)
{{- $pk := firstPK .Table.Fields }}
{{- if eq $pk.Type "types.Code" }}
	switch pk := primaryKey.(type) {
	case types.Code:
		return t.GetByPK(pk)
	case string:
		return t.GetByPK(types.NewCode(pk))
	}
{{- else if eq $pk.Type "types.Text" }}
	switch pk := primaryKey.(type) {
	case types.Text:
		return t.GetByPK(pk)
	case string:
		return t.GetByPK(types.NewText(pk))
	}
{{- else if eq $pk.Type "int" }}
	switch pk := primaryKey.(type) {
	case int:
		return t.GetByPK(pk)
	case float64:
		return t.GetByPK(int(pk))
	case string:
		var intVal int
		if _, err := fmt.Sscanf(pk, "%d", &intVal); err == nil {
			return t.GetByPK(intVal)
		}
	}
{{- else if eq $pk.Type "int64" }}
	switch pk := primaryKey.(type) {
	case int64:
		return t.GetByPK(pk)
	case int:
		return t.GetByPK(int64(pk))
	case float64:
		return t.GetByPK(int64(pk))
	case string:
		var intVal int64
		if _, err := fmt.Sscanf(pk, "%d", &intVal); err == nil {
			return t.GetByPK(intVal)
		}
	}
{{- else if eq $pk.Type "string" }}
	if pk, ok := primaryKey.(string); ok {
		return t.GetByPK(pk)
	}
{{- end }}
{{- else }}
	// For tables with composite keys, direct value is not supported
	// Use a map[string]interface{} with field names as keys
{{- end }}

	fmt.Printf("Error: Invalid primary key type for {{ .Table.Name }}.Get: %T (use map for composite keys)\n", primaryKey)
	return false
}

// GetByPK retrieves a record by its typed primary key(s) - for direct typed access
func (t *{{ .BaseStructName }}) GetByPK({{- range $i, $f := .Table.Fields }}{{- if $f.PrimaryKey }}{{ lowerFirst $f.Name }} {{ $f.Type }}{{ if not (isLastPK $i $.Table.Fields) }}, {{ end }}{{- end }}{{- end }}) bool {
{{- if .Table.Global }}
	tableName := {{ .StructName }}TableName
{{- else }}
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
{{- end }}

	{{- range .Table.Fields }}
	{{- if not .FlowField }}
	{{- if eq .Type "types.Code" }}
	var {{ lowerFirst .Name }}Null sql.NullString
	{{- else if eq .Type "types.Text" }}
	var {{ lowerFirst .Name }}Null sql.NullString
	{{- else if eq .Type "types.Decimal" }}
	var {{ lowerFirst .Name }}Null sql.NullString
	{{- else if eq .Type "types.Date" }}
	var {{ lowerFirst .Name }}Null sql.NullString
	{{- else if eq .Type "types.DateTime" }}
	var {{ lowerFirst .Name }}Null sql.NullString
	{{- else if eq .Type "bool" }}
	var {{ lowerFirst .Name }}Bool sql.NullBool
	{{- else if eq .Type "Option" }}
	var {{ lowerFirst .Name }}Int int
	{{- else }}
	var {{ lowerFirst .Name }}Val {{ .Type }}
	{{- end }}
	{{- end }}
	{{- end }}

	// Collect arguments for query
	args := []interface{}{
		{{- range $i, $f := .Table.Fields }}{{- if $f.PrimaryKey }}
		{{ lowerFirst $f.Name }},
		{{- end }}{{- end }}
	}

	// Build SQL with placeholders
	sqlStr := fmt.Sprintf(` + "`SELECT {{ range $i, $f := .Table.Fields }}{{ if not $f.FlowField }}{{ $f.DBName }}{{ if not (isLastDBField $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }} FROM \"%s\" WHERE 1=1{{ range .Table.Fields }}{{ if .PrimaryKey }} AND {{ .DBName }} = ?{{ end }}{{ end }}`" + `, tableName)

	// Convert placeholders for PostgreSQL
	sqlStr = t.convertPlaceholders(sqlStr, len(args))

	err := t.db.QueryRow(sqlStr, args...).Scan(
{{- range $i, $f := .Table.Fields }}
		{{- if not $f.FlowField }}
		{{- if eq $f.Type "types.Code" }}
		&{{ lowerFirst $f.Name }}Null,
		{{- else if eq $f.Type "types.Text" }}
		&{{ lowerFirst $f.Name }}Null,
		{{- else if eq $f.Type "types.Decimal" }}
		&{{ lowerFirst $f.Name }}Null,
		{{- else if eq $f.Type "types.Date" }}
		&{{ lowerFirst $f.Name }}Null,
		{{- else if eq $f.Type "types.DateTime" }}
		&{{ lowerFirst $f.Name }}Null,
		{{- else if eq $f.Type "bool" }}
		&{{ lowerFirst $f.Name }}Bool,
		{{- else if eq $f.Type "Option" }}
		&{{ lowerFirst $f.Name }}Int,
		{{- else }}
		&{{ lowerFirst $f.Name }}Val,
		{{- end }}
		{{- end }}
{{- end }}
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Record not found - this is not an error, just return false
			return false
		}
		// Actual database error
		fmt.Printf("Error: Failed to get {{ .Table.Name }}: %v\n", err)
		return false
	}

	// Populate fields
{{- range .Table.Fields }}
{{- if not .FlowField }}
{{- if eq .Type "types.Code" }}
	t.{{ upperFirst .Name }} = types.NewCode({{ lowerFirst .Name }}Null.String)
{{- else if eq .Type "types.Text" }}
	t.{{ upperFirst .Name }} = types.NewText({{ lowerFirst .Name }}Null.String)
{{- else if eq .Type "types.Decimal" }}
	t.{{ upperFirst .Name }}, _ = types.NewDecimalFromString({{ lowerFirst .Name }}Null.String)
{{- else if eq .Type "types.Date" }}
	t.{{ upperFirst .Name }}, _ = types.NewDateFromString({{ lowerFirst .Name }}Null.String)
{{- else if eq .Type "types.DateTime" }}
	t.{{ upperFirst .Name }}, _ = types.NewDateTimeFromString({{ lowerFirst .Name }}Null.String)
{{- else if eq .Type "bool" }}
	t.{{ upperFirst .Name }} = {{ lowerFirst .Name }}Bool.Bool
{{- else if eq .Type "Option" }}
	t.{{ upperFirst .Name }} = {{ $.StructName }}{{ upperFirst .Name }}({{ lowerFirst .Name }}Int)
{{- else }}
	t.{{ upperFirst .Name }} = {{ lowerFirst .Name }}Val
{{- end }}
{{- end }}
{{- end }}

	// Store old values for field tracking
	t.StoreOldValues()

	return true
}

// Insert inserts the record into the database
func (t *{{ .BaseStructName }}) Insert(runTrigger bool) bool {
	// Call OnInsert trigger if requested (via function reference set by wrapper)
	if runTrigger && t.onInsertFn != nil {
		if err := t.onInsertFn(); err != nil {
			fmt.Printf("Error: OnInsert trigger failed: %v\n", err)
			return false
		}
	}

{{- if .Table.Global }}
	tableName := {{ .StructName }}TableName
{{- else }}
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
{{- end }}

	// Collect arguments for INSERT
	args := []interface{}{
{{- range .Table.Fields }}
{{- if not .FlowField }}
		t.{{ upperFirst .Name }},
{{- end }}
{{- end }}
	}

	// Build SQL with placeholders
	sqlStr := fmt.Sprintf(` + "`INSERT INTO \"%s\" ({{ range $i, $f := .Table.Fields }}{{ if not $f.FlowField }}{{ $f.DBName }}{{ if not (isLastDBField $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }}) VALUES ({{ range $i, $f := .Table.Fields }}{{ if not $f.FlowField }}?{{ if not (isLastDBField $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }})`" + `, tableName)

	// Convert placeholders for PostgreSQL
	sqlStr = t.convertPlaceholders(sqlStr, len(args))

	_, err := t.db.Exec(sqlStr, args...)
	if err != nil {
		fmt.Printf("Error: Failed to insert {{ .Table.Name }}: %v\n", err)
		return false
	}
	return true
}

// Modify updates the record in the database
func (t *{{ .BaseStructName }}) Modify(runTrigger bool) bool {
	// Call OnModify trigger if requested (via function reference set by wrapper)
	if runTrigger && t.onModifyFn != nil {
		if err := t.onModifyFn(); err != nil {
			fmt.Printf("Error: OnModify trigger failed: %v\n", err)
			return false
		}
	}

{{- if .Table.Global }}
	tableName := {{ .StructName }}TableName
{{- else }}
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
{{- end }}

	// Build dynamic SQL based on field tracking
	var setClauses []string
	var values []interface{}

	// If we have old values (loaded from Get), only update changed fields
	if t.oldValues != nil {
{{- range .Table.Fields }}
{{- if and (not .PrimaryKey) (not .FlowField) }}
		if t.hasFieldChanged("{{ .DBName }}") {
			setClauses = append(setClauses, "{{ .DBName }} = ?")
			values = append(values, t.{{ upperFirst .Name }})
		}
{{- end }}
{{- end }}

		// If nothing changed, skip update
		if len(setClauses) == 0 {
			return true // No changes, success
		}
	} else {
		// No old values (fresh record), update all fields
{{- range .Table.Fields }}
{{- if and (not .PrimaryKey) (not .FlowField) }}
		setClauses = append(setClauses, "{{ .DBName }} = ?")
		values = append(values, t.{{ upperFirst .Name }})
{{- end }}
{{- end }}
	}

	// Add WHERE clause value (primary key)
{{- range .Table.Fields }}
{{- if .PrimaryKey }}
	values = append(values, t.{{ upperFirst .Name }})
{{- end }}
{{- end }}

	// Build and execute SQL
	sqlStr := fmt.Sprintf(` + "`UPDATE \"%s\" SET %s WHERE 1=1{{ range .Table.Fields }}{{ if .PrimaryKey }} AND {{ .DBName }} = ?{{ end }}{{ end }}`" + `,
		tableName,
		strings.Join(setClauses, ", "),
	)

	// Convert placeholders for PostgreSQL
	sqlStr = t.convertPlaceholders(sqlStr, len(values))

	_, err := t.db.Exec(sqlStr, values...)
	if err != nil {
		fmt.Printf("Error: Failed to modify {{ .Table.Name }}: %v\n", err)
		return false
	}
	return true
}

// hasFieldChanged checks if a field value has changed from oldValues
func (t *{{ .BaseStructName }}) hasFieldChanged(fieldName string) bool {
	if t.oldValues == nil {
		return true // No old values, assume changed
	}

	oldValue, exists := t.oldValues[fieldName]
	if !exists {
		return true // Field not in old values, assume changed
	}
	_ = oldValue // Suppress unused variable when all fields are PKs

	// Compare old vs new value based on field name (with type assertion)
	switch fieldName {
{{- range .Table.Fields }}
{{- if and (not .PrimaryKey) (not .FlowField) }}
	case "{{ .DBName }}":
{{- if eq .Type "Option" }}
		if old, ok := oldValue.({{ $.StructName }}{{ upperFirst .Name }}); ok {
			return t.{{ upperFirst .Name }} != old
		}
{{- else if eq .Type "types.Code" }}
		if old, ok := oldValue.(types.Code); ok {
			return !t.{{ upperFirst .Name }}.Equal(old)
		}
{{- else if eq .Type "types.Text" }}
		if old, ok := oldValue.(types.Text); ok {
			return !t.{{ upperFirst .Name }}.Equal(old)
		}
{{- else if eq .Type "types.Decimal" }}
		if old, ok := oldValue.(types.Decimal); ok {
			return !t.{{ upperFirst .Name }}.Equal(old)
		}
{{- else if eq .Type "types.Date" }}
		if old, ok := oldValue.(types.Date); ok {
			return !t.{{ upperFirst .Name }}.Equal(old)
		}
{{- else if eq .Type "types.DateTime" }}
		if old, ok := oldValue.(types.DateTime); ok {
			return !t.{{ upperFirst .Name }}.Equal(old)
		}
{{- else if eq .Type "[]byte" }}
		// Skip comparison for BLOB fields (too large, use always modified)
		return true
{{- else }}
		if old, ok := oldValue.({{ .Type }}); ok {
			return t.{{ upperFirst .Name }} != old
		}
		return true // Type mismatch, assume changed
{{- end }}
{{- end }}
{{- end }}
	}

	return false
}

// Delete removes the record from the database
func (t *{{ .BaseStructName }}) Delete(runTrigger bool) bool {
	// Call OnDelete trigger if requested (via function reference set by wrapper)
	if runTrigger && t.onDeleteFn != nil {
		if err := t.onDeleteFn(t.db, t.company); err != nil {
			fmt.Printf("Error: OnDelete trigger failed: %v\n", err)
			return false
		}
	}

{{- if .Table.Global }}
	tableName := {{ .StructName }}TableName
{{- else }}
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
{{- end }}

	// Collect arguments for DELETE
	args := []interface{}{
{{- range .Table.Fields }}
{{- if .PrimaryKey }}
		t.{{ upperFirst .Name }},
{{- end }}
{{- end }}
	}

	// Build SQL with placeholders
	sqlStr := fmt.Sprintf(` + "`DELETE FROM \"%s\" WHERE 1=1{{ range .Table.Fields }}{{ if .PrimaryKey }} AND {{ .DBName }} = ?{{ end }}{{ end }}`" + `, tableName)

	// Convert placeholders for PostgreSQL
	sqlStr = t.convertPlaceholders(sqlStr, len(args))

	_, err := t.db.Exec(sqlStr, args...)
	if err != nil {
		fmt.Printf("Error: Failed to delete {{ .Table.Name }}: %v\n", err)
		return false
	}
	return true
}

{{- if .HasFlowField }}

// ========================================
// FlowField Calculations (BC/NAV style)
// ========================================

// CalcFields calculates FlowField values (BC/NAV style)
// Usage:
//   customer.CalcFields("balance", "balance_lcy") - Calculate specific fields
//   customer.CalcFields() - Calculate all FlowFields
func (t *{{ .BaseStructName }}) CalcFields(fieldNames ...string) {
	// If no field names specified, calculate all FlowFields
	if len(fieldNames) == 0 {
		{{- range .Table.Fields }}
		{{- if .FlowField }}
		t.calcFlowField_{{ .Name }}()
		{{- end }}
		{{- end }}
		return
	}

	// Calculate only specified fields
	for _, fieldName := range fieldNames {
		switch fieldName {
		{{- range .Table.Fields }}
		{{- if .FlowField }}
		case "{{ .Name }}":
			t.calcFlowField_{{ .Name }}()
		{{- end }}
		{{- end }}
		}
	}
}

{{- range .Table.Fields }}
{{- if .FlowField }}

// calcFlowField_{{ .Name }} calculates the {{ .Name }} FlowField
// CalcFormula: {{ .CalcFormula }}({{ .SourceTable }}.{{ .SourceField }})
func (t *{{ $.BaseStructName }}) calcFlowField_{{ .Name }}() {
	{{- if eq .CalcFormula "Sum" }}
	t.{{ upperFirst .Name }} = t.calcSum{{ upperFirst .SourceTable }}{{ upperFirst .SourceField }}()
	{{- else if eq .CalcFormula "Count" }}
	t.{{ upperFirst .Name }} = t.calcCount{{ upperFirst .SourceTable }}()
	{{- else if eq .CalcFormula "Average" }}
	t.{{ upperFirst .Name }} = t.calcAverage{{ upperFirst .SourceTable }}{{ upperFirst .SourceField }}()
	{{- else if eq .CalcFormula "Min" }}
	t.{{ upperFirst .Name }} = t.calcMin{{ upperFirst .SourceTable }}{{ upperFirst .SourceField }}()
	{{- else if eq .CalcFormula "Max" }}
	t.{{ upperFirst .Name }} = t.calcMax{{ upperFirst .SourceTable }}{{ upperFirst .SourceField }}()
	{{- else if eq .CalcFormula "Lookup" }}
	t.{{ upperFirst .Name }} = t.calcLookup{{ upperFirst .SourceTable }}{{ upperFirst .SourceField }}()
	{{- else if eq .CalcFormula "Exist" }}
	t.{{ upperFirst .Name }} = t.calcExist{{ upperFirst .SourceTable }}()
	{{- end }}
}
{{- end }}
{{- end }}

// Helper methods for FlowField calculations
{{- range .Table.Fields }}
{{- if and .FlowField (eq .CalcFormula "Sum") }}

func (t *{{ $.BaseStructName }}) calcSum{{ upperFirst .SourceTable }}{{ upperFirst .SourceField }}() {{ .Type }} {
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .SourceTable }}TableName)

	// Build WHERE clause from FlowFilters
	var whereClauses []string
	var args []interface{}

	{{- range .FlowFilters }}
	{{- if eq .Type "const" }}
	whereClauses = append(whereClauses, "{{ .Field }} = ?")
	args = append(args, {{ .Value }})
	{{- else if eq .Type "field" }}
	whereClauses = append(whereClauses, "{{ .Field }} = ?")
	args = append(args, t.{{ upperFirst .Value }})
	{{- end }}
	{{- end }}

	whereClause := "1=1"
	if len(whereClauses) > 0 {
		whereClause = strings.Join(whereClauses, " AND ")
	}

	query := fmt.Sprintf(` + "`SELECT COALESCE(SUM({{ .SourceField }}), 0) FROM \"%s\" WHERE %s`" + `, tableName, whereClause)

	// Convert placeholders for PostgreSQL
	query = t.convertPlaceholders(query, len(args))

	var sumStr string
	err := t.db.QueryRow(query, args...).Scan(&sumStr)
	if err != nil {
		fmt.Printf("Error: Failed to calculate sum for {{ .Name }}: %v\n", err)
		return types.ZeroDecimal()
	}

	sum, _ := types.NewDecimalFromString(sumStr)
	return sum
}
{{- end }}
{{- if and .FlowField (eq .CalcFormula "Count") }}

func (t *{{ $.BaseStructName }}) calcCount{{ upperFirst .SourceTable }}() int {
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .SourceTable }}TableName)

	// Build WHERE clause from FlowFilters
	var whereClauses []string
	var args []interface{}

	{{- range .FlowFilters }}
	{{- if eq .Type "const" }}
	whereClauses = append(whereClauses, "{{ .Field }} = ?")
	args = append(args, {{ .Value }})
	{{- else if eq .Type "field" }}
	whereClauses = append(whereClauses, "{{ .Field }} = ?")
	args = append(args, t.{{ upperFirst .Value }})
	{{- end }}
	{{- end }}

	whereClause := "1=1"
	if len(whereClauses) > 0 {
		whereClause = strings.Join(whereClauses, " AND ")
	}

	query := fmt.Sprintf(` + "`SELECT COUNT(*) FROM \"%s\" WHERE %s`" + `, tableName, whereClause)

	// Convert placeholders for PostgreSQL
	query = t.convertPlaceholders(query, len(args))

	var count int
	err := t.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		fmt.Printf("Error: Failed to calculate count for {{ .Name }}: %v\n", err)
		return 0
	}

	return count
}
{{- end }}
{{- end }}

{{- else }}

// CalcFields is a no-op for tables without FlowFields
// Implemented for tables.Table interface compliance
func (t *{{ .BaseStructName }}) CalcFields(fieldNames ...string) {
	// This table has no FlowFields to calculate
}

{{- end }}

// ========================================
// BC/NAV-style Filtering and Search
// ========================================

// {{ lowerFirst .BaseStructName }}FilterCondition represents a filter on a field
type {{ lowerFirst .BaseStructName }}FilterCondition struct {
	fieldName    string
	minValue     interface{}
	maxValue     interface{}
	filterExpr   string        // For complex SetFilter expressions
	isExpression bool          // True if using filterExpr instead of min/max
}

// SetRange sets a range filter on a field (BC/NAV style)
// Usage:
//   SetRange("No", "10000") - exact match (No = "10000")
//   SetRange("No", "10000", "20000") - range (No between "10000" and "20000")
func (t *{{ .BaseStructName }}) SetRange(fieldName string, values ...interface{}) {
	if t.filters == nil {
		t.filters = make(map[string]*{{ lowerFirst .BaseStructName }}FilterCondition)
	}

	var minValue, maxValue interface{}

	switch len(values) {
	case 1:
		// Exact match: SetRange("No", "10000")
		minValue = values[0]
		maxValue = values[0]
	case 2:
		// Range: SetRange("No", "10000", "20000")
		minValue = values[0]
		maxValue = values[1]
	default:
		fmt.Printf("Error: SetRange requires 1 or 2 values, got %d\n", len(values))
		return
	}

	t.filters[fieldName] = &{{ lowerFirst .BaseStructName }}FilterCondition{
		fieldName: fieldName,
		minValue:  minValue,
		maxValue:  maxValue,
	}
}

// SetFilter sets a complex filter expression on a field (BC/NAV style)
// Supports BC/NAV filter syntax: "100..200|500" (range OR exact value)
// Operators: .. (range), | (OR), & (AND), * (wildcard), <> (not equal)
// Example: customer.SetFilter("No", "001..003|005")
func (t *{{ .BaseStructName }}) SetFilter(fieldName, filterExpr string) {
	if t.filters == nil {
		t.filters = make(map[string]*{{ lowerFirst .BaseStructName }}FilterCondition)
	}
	t.filters[fieldName] = &{{ lowerFirst .BaseStructName }}FilterCondition{
		fieldName:    fieldName,
		filterExpr:   filterExpr,
		isExpression: true,
	}
}

// SetCurrentKey sets the sort order for queries (BC/NAV style)
// Example: customer.SetCurrentKey("City", "Name")
func (t *{{ .BaseStructName }}) SetCurrentKey(fields ...string) {
	t.orderByFields = fields
}

// Reset clears all filters (BC/NAV style)
func (t *{{ .BaseStructName }}) Reset() {
	t.filters = nil
	t.oldValues = nil
	t.orderByFields = nil
	if t.currentRows != nil {
		t.currentRows.Close()
		t.currentRows = nil
	}
}

// buildWhereClause builds WHERE clause from current filters
func (t *{{ .BaseStructName }}) buildWhereClause() (string, []interface{}) {
	if len(t.filters) == 0 {
		return "1=1", nil
	}

	var conditions []string
	var args []interface{}

	for _, filter := range t.filters {
		if filter.isExpression {
			// Parse BC/NAV filter expression
			clause, exprArgs := t.parseFilterExpression(filter.fieldName, filter.filterExpr)
			conditions = append(conditions, clause)
			args = append(args, exprArgs...)
		} else {
			// Simple range filter
			if filter.minValue != nil && filter.maxValue != nil {
				conditions = append(conditions, fmt.Sprintf("%s BETWEEN ? AND ?", filter.fieldName))
				args = append(args, filter.minValue, filter.maxValue)
			} else if filter.minValue != nil {
				conditions = append(conditions, fmt.Sprintf("%s >= ?", filter.fieldName))
				args = append(args, filter.minValue)
			} else if filter.maxValue != nil {
				conditions = append(conditions, fmt.Sprintf("%s <= ?", filter.fieldName))
				args = append(args, filter.maxValue)
			}
		}
	}

	where := strings.Join(conditions, " AND ")
	if where == "" {
		where = "1=1"
	}

	return where, args
}

// parseFilterExpression parses BC/NAV filter expressions into SQL
// Supports: "100..200" (range), "100|200|300" (OR), "100..200|500" (combined)
func (t *{{ .BaseStructName }}) parseFilterExpression(fieldName, expr string) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	// Split by | (OR operator)
	orParts := strings.Split(expr, "|")

	for _, part := range orParts {
		part = strings.TrimSpace(part)

		// Check for range (..)
		if strings.Contains(part, "..") {
			rangeParts := strings.Split(part, "..")
			if len(rangeParts) == 2 {
				min := strings.TrimSpace(rangeParts[0])
				max := strings.TrimSpace(rangeParts[1])
				conditions = append(conditions, fmt.Sprintf("%s BETWEEN ? AND ?", fieldName))
				args = append(args, min, max)
			}
		} else if strings.Contains(part, "*") {
			// Wildcard support: convert * to %
			likePattern := strings.ReplaceAll(part, "*", "%")
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", fieldName))
			args = append(args, likePattern)
		} else if strings.HasPrefix(part, "<>") {
			// Not equal
			value := strings.TrimSpace(strings.TrimPrefix(part, "<>"))
			conditions = append(conditions, fmt.Sprintf("%s <> ?", fieldName))
			args = append(args, value)
		} else {
			// Exact match
			conditions = append(conditions, fmt.Sprintf("%s = ?", fieldName))
			args = append(args, part)
		}
	}

	// Join with OR
	whereClause := "(" + strings.Join(conditions, " OR ") + ")"
	return whereClause, args
}

// getOrderByClause builds ORDER BY clause from current key
func (t *{{ .BaseStructName }}) getOrderByClause() string {
	if len(t.orderByFields) > 0 {
		return strings.Join(t.orderByFields, ", ")
	}
	// Default: order by primary key
	return "{{ range $i, $f := .Table.Fields }}{{ if $f.PrimaryKey }}{{ $f.DBName }}{{ if not (isLastPK $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }}"
}

// FindFirst finds the first record matching current filters (BC/NAV style)
// Returns true if found, false if not found
func (t *{{ .BaseStructName }}) FindFirst() bool {
{{- if .Table.Global }}
	tableName := {{ .StructName }}TableName
{{- else }}
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
{{- end }}
	where, args := t.buildWhereClause()

	// Build SELECT with all fields
	query := fmt.Sprintf(` + "`SELECT {{ range $i, $f := .Table.Fields }}{{ if not $f.FlowField }}{{ $f.DBName }}{{ if not (isLastDBField $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }} FROM \"%s\" WHERE %s ORDER BY {{ range $i, $f := .Table.Fields }}{{ if $f.PrimaryKey }}{{ $f.DBName }}{{ if not (isLastPK $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }} ASC LIMIT 1`" + `, tableName, where)

	// Convert placeholders for PostgreSQL
	query = t.convertPlaceholders(query, len(args))

{{- range .Table.Fields }}
{{- if not .FlowField }}
{{- if eq .Type "types.Code" }}
	var {{ .Name }}Null sql.NullString
{{- else if eq .Type "types.Text" }}
	var {{ .Name }}Null sql.NullString
{{- else if eq .Type "types.Decimal" }}
	var {{ .Name }}Null sql.NullString
{{- else if eq .Type "types.Date" }}
	var {{ .Name }}Null sql.NullString
{{- else if eq .Type "types.DateTime" }}
	var {{ .Name }}Null sql.NullString
{{- else if eq .Type "bool" }}
	var {{ .Name }}Bool sql.NullBool
{{- else if eq .Type "Option" }}
	var {{ .Name }}Int int
{{- else if eq .Type "time.Time" }}
	var {{ .Name }}Time time.Time
{{- end }}
{{- end }}
{{- end }}

	err := t.db.QueryRow(query, args...).Scan(
{{- range $i, $f := .Table.Fields }}
{{- if not $f.FlowField }}
{{- if eq $f.Type "types.Code" }}
		&{{ $f.Name }}Null,
{{- else if eq $f.Type "types.Text" }}
		&{{ $f.Name }}Null,
{{- else if eq $f.Type "types.Decimal" }}
		&{{ $f.Name }}Null,
{{- else if eq $f.Type "types.Date" }}
		&{{ $f.Name }}Null,
{{- else if eq $f.Type "types.DateTime" }}
		&{{ $f.Name }}Null,
{{- else if eq $f.Type "bool" }}
		&{{ $f.Name }}Bool,
{{- else if eq $f.Type "Option" }}
		&{{ $f.Name }}Int,
{{- else if eq $f.Type "time.Time" }}
		&{{ $f.Name }}Time,
{{- else }}
		&t.{{ upperFirst $f.Name }},
{{- end }}
{{- end }}
{{- end }}
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		fmt.Printf("Error: Failed to find first {{ .Table.Name }}: %v\n", err)
		return false
	}

	// Populate fields
{{- range .Table.Fields }}
{{- if not .FlowField }}
{{- if eq .Type "types.Code" }}
	t.{{ upperFirst .Name }} = types.NewCode({{ .Name }}Null.String)
{{- else if eq .Type "types.Text" }}
	t.{{ upperFirst .Name }} = types.NewText({{ .Name }}Null.String)
{{- else if eq .Type "types.Decimal" }}
	t.{{ upperFirst .Name }}, _ = types.NewDecimalFromString({{ .Name }}Null.String)
{{- else if eq .Type "types.Date" }}
	t.{{ upperFirst .Name }}, _ = types.NewDateFromString({{ .Name }}Null.String)
{{- else if eq .Type "types.DateTime" }}
	t.{{ upperFirst .Name }}, _ = types.NewDateTimeFromString({{ .Name }}Null.String)
{{- else if eq .Type "bool" }}
	t.{{ upperFirst .Name }} = {{ .Name }}Bool.Bool
{{- else if eq .Type "Option" }}
	t.{{ upperFirst .Name }} = {{ $.StructName }}{{ upperFirst .Name }}({{ .Name }}Int)
{{- else if eq .Type "time.Time" }}
	t.{{ upperFirst .Name }} = {{ .Name }}Time
{{- end }}
{{- end }}
{{- end }}

	// Store old values for field tracking
	t.StoreOldValues()

	return true
}

// FindLast finds the last record matching current filters (BC/NAV style)
// Returns true if found, false if not found
func (t *{{ .BaseStructName }}) FindLast() bool {
{{- if .Table.Global }}
	tableName := {{ .StructName }}TableName
{{- else }}
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
{{- end }}
	where, args := t.buildWhereClause()

	// Build SELECT with all fields
	query := fmt.Sprintf(` + "`SELECT {{ range $i, $f := .Table.Fields }}{{ if not $f.FlowField }}{{ $f.DBName }}{{ if not (isLastDBField $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }} FROM \"%s\" WHERE %s ORDER BY {{ range $i, $f := .Table.Fields }}{{ if $f.PrimaryKey }}{{ $f.DBName }}{{ if not (isLastPK $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }} DESC LIMIT 1`" + `, tableName, where)

	// Convert placeholders for PostgreSQL
	query = t.convertPlaceholders(query, len(args))

{{- range .Table.Fields }}
{{- if not .FlowField }}
{{- if eq .Type "types.Code" }}
	var {{ .Name }}Null sql.NullString
{{- else if eq .Type "types.Text" }}
	var {{ .Name }}Null sql.NullString
{{- else if eq .Type "types.Decimal" }}
	var {{ .Name }}Null sql.NullString
{{- else if eq .Type "types.Date" }}
	var {{ .Name }}Null sql.NullString
{{- else if eq .Type "types.DateTime" }}
	var {{ .Name }}Null sql.NullString
{{- else if eq .Type "bool" }}
	var {{ .Name }}Bool sql.NullBool
{{- else if eq .Type "Option" }}
	var {{ .Name }}Int int
{{- else if eq .Type "time.Time" }}
	var {{ .Name }}Time time.Time
{{- end }}
{{- end }}
{{- end }}

	err := t.db.QueryRow(query, args...).Scan(
{{- range $i, $f := .Table.Fields }}
{{- if not $f.FlowField }}
{{- if eq $f.Type "types.Code" }}
		&{{ $f.Name }}Null,
{{- else if eq $f.Type "types.Text" }}
		&{{ $f.Name }}Null,
{{- else if eq $f.Type "types.Decimal" }}
		&{{ $f.Name }}Null,
{{- else if eq $f.Type "types.Date" }}
		&{{ $f.Name }}Null,
{{- else if eq $f.Type "types.DateTime" }}
		&{{ $f.Name }}Null,
{{- else if eq $f.Type "bool" }}
		&{{ $f.Name }}Bool,
{{- else if eq $f.Type "Option" }}
		&{{ $f.Name }}Int,
{{- else if eq $f.Type "time.Time" }}
		&{{ $f.Name }}Time,
{{- else }}
		&t.{{ upperFirst $f.Name }},
{{- end }}
{{- end }}
{{- end }}
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		fmt.Printf("Error: Failed to find last {{ .Table.Name }}: %v\n", err)
		return false
	}

	// Populate fields
{{- range .Table.Fields }}
{{- if not .FlowField }}
{{- if eq .Type "types.Code" }}
	t.{{ upperFirst .Name }} = types.NewCode({{ .Name }}Null.String)
{{- else if eq .Type "types.Text" }}
	t.{{ upperFirst .Name }} = types.NewText({{ .Name }}Null.String)
{{- else if eq .Type "types.Decimal" }}
	t.{{ upperFirst .Name }}, _ = types.NewDecimalFromString({{ .Name }}Null.String)
{{- else if eq .Type "types.Date" }}
	t.{{ upperFirst .Name }}, _ = types.NewDateFromString({{ .Name }}Null.String)
{{- else if eq .Type "types.DateTime" }}
	t.{{ upperFirst .Name }}, _ = types.NewDateTimeFromString({{ .Name }}Null.String)
{{- else if eq .Type "bool" }}
	t.{{ upperFirst .Name }} = {{ .Name }}Bool.Bool
{{- else if eq .Type "Option" }}
	t.{{ upperFirst .Name }} = {{ $.StructName }}{{ upperFirst .Name }}({{ .Name }}Int)
{{- else if eq .Type "time.Time" }}
	t.{{ upperFirst .Name }} = {{ .Name }}Time
{{- end }}
{{- end }}
{{- end }}

	// Store old values for field tracking
	t.StoreOldValues()

	return true
}

// Count returns the number of records matching current filters (BC/NAV style)
func (t *{{ .BaseStructName }}) Count() int {
{{- if .Table.Global }}
	tableName := {{ .StructName }}TableName
{{- else }}
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
{{- end }}
	where, args := t.buildWhereClause()

	query := fmt.Sprintf(` + "`SELECT COUNT(*) FROM \"%s\" WHERE %s`" + `, tableName, where)

	// Convert placeholders for PostgreSQL
	query = t.convertPlaceholders(query, len(args))

	var count int
	err := t.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		fmt.Printf("Error: Failed to count {{ .Table.Name }}: %v\n", err)
		return 0
	}

	return count
}

// FindSet opens a result set matching current filters (BC/NAV style)
// Call Next() to iterate through the results
// Returns true if at least one record found, false otherwise
func (t *{{ .BaseStructName }}) FindSet() bool {
	// Close any existing result set
	if t.currentRows != nil {
		t.currentRows.Close()
		t.currentRows = nil
	}

{{- if .Table.Global }}
	tableName := {{ .StructName }}TableName
{{- else }}
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
{{- end }}
	where, args := t.buildWhereClause()
	orderBy := t.getOrderByClause()

	// Build SELECT with all fields
	query := fmt.Sprintf(` + "`SELECT {{ range $i, $f := .Table.Fields }}{{ if not $f.FlowField }}{{ $f.DBName }}{{ if not (isLastDBField $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }} FROM \"%s\" WHERE %s ORDER BY %s`" + `, tableName, where, orderBy)

	// Convert placeholders for PostgreSQL
	query = t.convertPlaceholders(query, len(args))

	rows, err := t.db.Query(query, args...)
	if err != nil {
		fmt.Printf("Error: Failed to execute FindSet for {{ .Table.Name }}: %v\n", err)
		return false
	}

	t.currentRows = rows

	// Load first record
	return t.Next()
}

// Next advances to the next record in the result set (BC/NAV style)
// Must be called after FindSet() or FindSetBuffered()
// Optional steps parameter:
//   - Next() or Next(1): Move forward 1 record (default)
//   - Next(5): Skip forward 5 records
//   - Next(-1): Move backward 1 record (only with FindSetBuffered)
//   - Next(-3): Skip backward 3 records (only with FindSetBuffered)
// Returns true if a record was loaded, false if no more records or out of bounds
func (t *{{ .BaseStructName }}) Next(steps ...int) bool {
	// Default to 1 step forward
	step := 1
	if len(steps) > 0 {
		step = steps[0]
	}

	// BUFFERED MODE: Bidirectional navigation with in-memory records
	if t.bufferedRecords != nil {
		// Calculate new position
		newPos := t.currentBufferPos + step

		// Check bounds
		if newPos < 0 || newPos >= len(t.bufferedRecords) {
			return false // Out of bounds
		}

		// Move to new position
		t.currentBufferPos = newPos
		t.copyFromBuffered(t.bufferedRecords[t.currentBufferPos])
		return true
	}

	// FORWARD-ONLY MODE: Streaming with sql.Rows (only positive steps allowed)
	if t.currentRows != nil {
		// Validate: only forward movement allowed
		if step < 1 {
			fmt.Printf("Error: Backward navigation (Next(%d)) requires FindSetBuffered()\n", step)
			return false
		}

		// Advance 'step' times (1 = next record, 2 = skip 1 record, etc.)
		for i := 0; i < step; i++ {
			if !t.currentRows.Next() {
				// No more rows - close result set
				t.currentRows.Close()
				t.currentRows = nil
				return false
			}
		}

		// Scan the row
{{- range .Table.Fields }}
{{- if not .FlowField }}
{{- if eq .Type "types.Code" }}
		var {{ .Name }}Null sql.NullString
{{- else if eq .Type "types.Text" }}
		var {{ .Name }}Null sql.NullString
{{- else if eq .Type "types.Decimal" }}
		var {{ .Name }}Null sql.NullString
{{- else if eq .Type "types.Date" }}
		var {{ .Name }}Null sql.NullString
{{- else if eq .Type "types.DateTime" }}
		var {{ .Name }}Null sql.NullString
{{- else if eq .Type "bool" }}
		var {{ .Name }}Bool sql.NullBool
{{- else if eq .Type "Option" }}
		var {{ .Name }}Int int
{{- else if eq .Type "time.Time" }}
		var {{ .Name }}Time time.Time
{{- end }}
{{- end }}
{{- end }}

		err := t.currentRows.Scan(
{{- range $i, $f := .Table.Fields }}
{{- if not $f.FlowField }}
{{- if eq $f.Type "types.Code" }}
			&{{ $f.Name }}Null,
{{- else if eq $f.Type "types.Text" }}
			&{{ $f.Name }}Null,
{{- else if eq $f.Type "types.Decimal" }}
			&{{ $f.Name }}Null,
{{- else if eq $f.Type "types.Date" }}
			&{{ $f.Name }}Null,
{{- else if eq $f.Type "types.DateTime" }}
			&{{ $f.Name }}Null,
{{- else if eq $f.Type "bool" }}
			&{{ $f.Name }}Bool,
{{- else if eq $f.Type "Option" }}
			&{{ $f.Name }}Int,
{{- else if eq $f.Type "time.Time" }}
			&{{ $f.Name }}Time,
{{- else }}
			&t.{{ upperFirst $f.Name }},
{{- end }}
{{- end }}
{{- end }}
		)

		if err != nil {
			fmt.Printf("Error: Failed to scan {{ .Table.Name }} record: %v\n", err)
			t.currentRows.Close()
			t.currentRows = nil
			return false
		}

		// Populate fields
{{- range .Table.Fields }}
{{- if not .FlowField }}
{{- if eq .Type "types.Code" }}
		t.{{ upperFirst .Name }} = types.NewCode({{ .Name }}Null.String)
{{- else if eq .Type "types.Text" }}
		t.{{ upperFirst .Name }} = types.NewText({{ .Name }}Null.String)
{{- else if eq .Type "types.Decimal" }}
		t.{{ upperFirst .Name }}, _ = types.NewDecimalFromString({{ .Name }}Null.String)
{{- else if eq .Type "types.Date" }}
		t.{{ upperFirst .Name }}, _ = types.NewDateFromString({{ .Name }}Null.String)
{{- else if eq .Type "types.DateTime" }}
		t.{{ upperFirst .Name }}, _ = types.NewDateTimeFromString({{ .Name }}Null.String)
{{- else if eq .Type "bool" }}
		t.{{ upperFirst .Name }} = {{ .Name }}Bool.Bool
{{- else if eq .Type "Option" }}
		t.{{ upperFirst .Name }} = {{ $.StructName }}{{ upperFirst .Name }}({{ .Name }}Int)
{{- else if eq .Type "time.Time" }}
		t.{{ upperFirst .Name }} = {{ .Name }}Time
{{- end }}
{{- end }}
{{- end }}

		// Store old values for field tracking
		t.StoreOldValues()

		return true
	}

	// No active recordset
	return false
}

// FindSetBuffered loads all filtered records into memory for bidirectional navigation (BC/NAV style)
// Use this when you need to move backward/forward with Next(steps)
// Filters (SetRange/SetFilter) are applied in SQL before buffering to minimize memory usage
// Returns true if at least one record found, false otherwise
func (t *{{ .BaseStructName }}) FindSetBuffered() bool {
	// Close any existing forward-only result set
	if t.currentRows != nil {
		t.currentRows.Close()
		t.currentRows = nil
	}

	// Clear any existing buffer
	t.bufferedRecords = nil
	t.currentBufferPos = -1

{{- if .Table.Global }}
	tableName := {{ .StructName }}TableName
{{- else }}
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
{{- end }}
	where, args := t.buildWhereClause()
	orderBy := t.getOrderByClause()

	// Build SELECT with all fields
	query := fmt.Sprintf(` + "`SELECT {{ range $i, $f := .Table.Fields }}{{ if not $f.FlowField }}{{ $f.DBName }}{{ if not (isLastDBField $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }} FROM \"%s\" WHERE %s ORDER BY %s`" + `, tableName, where, orderBy)

	// Convert placeholders for PostgreSQL
	query = t.convertPlaceholders(query, len(args))

	rows, err := t.db.Query(query, args...)
	if err != nil {
		fmt.Printf("Error: Failed to execute FindSetBuffered for {{ .Table.Name }}: %v\n", err)
		return false
	}
	defer rows.Close()

	// Load all records into memory
	for rows.Next() {
		// Create a new record instance
		record := &{{ .BaseStructName }}{}
		record.db = t.db
		record.company = t.company
		record.dbType = t.dbType

		// Scan the row
{{- range .Table.Fields }}
{{- if not .FlowField }}
{{- if eq .Type "types.Code" }}
		var {{ .Name }}Null sql.NullString
{{- else if eq .Type "types.Text" }}
		var {{ .Name }}Null sql.NullString
{{- else if eq .Type "types.Decimal" }}
		var {{ .Name }}Null sql.NullString
{{- else if eq .Type "types.Date" }}
		var {{ .Name }}Null sql.NullString
{{- else if eq .Type "types.DateTime" }}
		var {{ .Name }}Null sql.NullString
{{- else if eq .Type "bool" }}
		var {{ .Name }}Bool sql.NullBool
{{- else if eq .Type "Option" }}
		var {{ .Name }}Int int
{{- else if eq .Type "time.Time" }}
		var {{ .Name }}Time time.Time
{{- end }}
{{- end }}
{{- end }}

		err := rows.Scan(
{{- range $i, $f := .Table.Fields }}
{{- if not $f.FlowField }}
{{- if eq $f.Type "types.Code" }}
			&{{ $f.Name }}Null,
{{- else if eq $f.Type "types.Text" }}
			&{{ $f.Name }}Null,
{{- else if eq $f.Type "types.Decimal" }}
			&{{ $f.Name }}Null,
{{- else if eq $f.Type "types.Date" }}
			&{{ $f.Name }}Null,
{{- else if eq $f.Type "types.DateTime" }}
			&{{ $f.Name }}Null,
{{- else if eq $f.Type "bool" }}
			&{{ $f.Name }}Bool,
{{- else if eq $f.Type "Option" }}
			&{{ $f.Name }}Int,
{{- else if eq $f.Type "time.Time" }}
			&{{ $f.Name }}Time,
{{- else }}
			&record.{{ upperFirst $f.Name }},
{{- end }}
{{- end }}
{{- end }}
		)

		if err != nil {
			fmt.Printf("Error: Failed to scan {{ .Table.Name }} record: %v\n", err)
			return false
		}

		// Populate special type fields
{{- range .Table.Fields }}
{{- if not .FlowField }}
{{- if eq .Type "types.Code" }}
		record.{{ upperFirst .Name }} = types.NewCode({{ .Name }}Null.String)
{{- else if eq .Type "types.Text" }}
		record.{{ upperFirst .Name }} = types.NewText({{ .Name }}Null.String)
{{- else if eq .Type "types.Decimal" }}
		record.{{ upperFirst .Name }}, _ = types.NewDecimalFromString({{ .Name }}Null.String)
{{- else if eq .Type "types.Date" }}
		record.{{ upperFirst .Name }}, _ = types.NewDateFromString({{ .Name }}Null.String)
{{- else if eq .Type "types.DateTime" }}
		record.{{ upperFirst .Name }}, _ = types.NewDateTimeFromString({{ .Name }}Null.String)
{{- else if eq .Type "bool" }}
		record.{{ upperFirst .Name }} = {{ .Name }}Bool.Bool
{{- else if eq .Type "Option" }}
		record.{{ upperFirst .Name }} = {{ $.StructName }}{{ upperFirst .Name }}({{ .Name }}Int)
{{- else if eq .Type "time.Time" }}
		record.{{ upperFirst .Name }} = {{ .Name }}Time
{{- end }}
{{- end }}
{{- end }}

		// Store old values
		record.StoreOldValues()

		// Add to buffer
		t.bufferedRecords = append(t.bufferedRecords, record)
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		fmt.Printf("Error: Failed to iterate {{ .Table.Name }} records: %v\n", err)
		return false
	}

	// If no records found, return false
	if len(t.bufferedRecords) == 0 {
		return false
	}

	// Load first record into current instance
	t.currentBufferPos = 0
	t.copyFromBuffered(t.bufferedRecords[0])

	return true
}

// copyFromBuffered copies field values from a buffered record to the current instance
func (t *{{ .BaseStructName }}) copyFromBuffered(record *{{ .BaseStructName }}) {
{{- range .Table.Fields }}
	t.{{ upperFirst .Name }} = record.{{ upperFirst .Name }}
{{- end }}
	t.StoreOldValues()
}

// ========================================
// Phase 3: Advanced BC/NAV Methods
// ========================================

// IsEmpty returns true if no records match current filters (BC/NAV style)
func (t *{{ .BaseStructName }}) IsEmpty() bool {
	return t.Count() == 0
}

// ModifyAll updates a field for all records matching current filters (BC/NAV style)
// Returns the number of records modified
func (t *{{ .BaseStructName }}) ModifyAll(fieldName string, newValue interface{}) int {
{{- if .Table.Global }}
	tableName := {{ .StructName }}TableName
{{- else }}
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
{{- end }}
	where, args := t.buildWhereClause()

	// Build UPDATE SQL
	updateSQL := fmt.Sprintf(` + "`UPDATE \"%s\" SET %s = ? WHERE %s`" + `, tableName, fieldName, where)

	// Prepend newValue to args
	allArgs := append([]interface{}{newValue}, args...)

	// Convert placeholders for PostgreSQL
	updateSQL = t.convertPlaceholders(updateSQL, len(allArgs))

	result, err := t.db.Exec(updateSQL, allArgs...)
	if err != nil {
		fmt.Printf("Error: Failed to modify all {{ .Table.Name }}: %v\n", err)
		return 0
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected)
}

// DeleteAll deletes all records matching current filters (BC/NAV style)
// Returns the number of records deleted
func (t *{{ .BaseStructName }}) DeleteAll() int {
{{- if .Table.Global }}
	tableName := {{ .StructName }}TableName
{{- else }}
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
{{- end }}
	where, args := t.buildWhereClause()

	// Build DELETE SQL
	deleteSQL := fmt.Sprintf(` + "`DELETE FROM \"%s\" WHERE %s`" + `, tableName, where)

	// Convert placeholders for PostgreSQL
	deleteSQL = t.convertPlaceholders(deleteSQL, len(args))

	result, err := t.db.Exec(deleteSQL, args...)
	if err != nil {
		fmt.Printf("Error: Failed to delete all {{ .Table.Name }}: %v\n", err)
		return 0
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected)
}

// CopyFilters copies filters from another record variable (BC/NAV style)
func (t *{{ .BaseStructName }}) CopyFilters(from *{{ .BaseStructName }}) {
	if from.filters == nil {
		t.filters = nil
		return
	}

	// Deep copy filters
	t.filters = make(map[string]*{{ lowerFirst .BaseStructName }}FilterCondition)
	for key, filter := range from.filters {
		t.filters[key] = &{{ lowerFirst .BaseStructName }}FilterCondition{
			fieldName:    filter.fieldName,
			minValue:     filter.minValue,
			maxValue:     filter.maxValue,
			filterExpr:   filter.filterExpr,
			isExpression: filter.isExpression,
		}
	}

	// Also copy order by fields
	if len(from.orderByFields) > 0 {
		t.orderByFields = make([]string, len(from.orderByFields))
		copy(t.orderByFields, from.orderByFields)
	} else {
		t.orderByFields = nil
	}
}

// GetFilters returns a string representation of current filters (BC/NAV style)
// Useful for debugging and logging
func (t *{{ .BaseStructName }}) GetFilters() string {
	if len(t.filters) == 0 {
		return ""
	}

	var parts []string
	for _, filter := range t.filters {
		if filter.isExpression {
			parts = append(parts, fmt.Sprintf("%s: %s", filter.fieldName, filter.filterExpr))
		} else if filter.minValue != nil && filter.maxValue != nil {
			parts = append(parts, fmt.Sprintf("%s: %v..%v", filter.fieldName, filter.minValue, filter.maxValue))
		} else if filter.minValue != nil {
			parts = append(parts, fmt.Sprintf("%s: >=%v", filter.fieldName, filter.minValue))
		} else if filter.maxValue != nil {
			parts = append(parts, fmt.Sprintf("%s: <=%v", filter.fieldName, filter.maxValue))
		}
	}

	return strings.Join(parts, ", ")
}

// ========================================
// BC/NAV-style Field Validation
// ========================================

// ValidateField validates a field and calls its OnValidate trigger (BC/NAV style)
// This is equivalent to the BC/NAV VALIDATE function
// Usage: customer.ValidateField("Payment_terms_code", types.NewCode("30DAYS"))
func (t *{{ .BaseStructName }}) ValidateField(fieldName string, value interface{}) error {
	fieldNameLower := strings.ToLower(fieldName)

	switch fieldNameLower {
{{- range .Table.Fields }}
{{- if not .FlowField }}
	case "{{ .DBName }}":
		// Set field value
{{- if eq .Type "types.Code" }}
		if v, ok := value.(types.Code); ok {
			t.{{ upperFirst .Name }} = v
		} else if v, ok := value.(string); ok {
			t.{{ upperFirst .Name }} = types.NewCode(v)
		} else {
			return fmt.Errorf("invalid type for field {{ .Name }}")
		}
{{- else if eq .Type "types.Text" }}
		if v, ok := value.(types.Text); ok {
			t.{{ upperFirst .Name }} = v
		} else if v, ok := value.(string); ok {
			t.{{ upperFirst .Name }} = types.NewText(v)
		} else {
			return fmt.Errorf("invalid type for field {{ .Name }}")
		}
{{- else if eq .Type "types.Decimal" }}
		if v, ok := value.(types.Decimal); ok {
			t.{{ upperFirst .Name }} = v
		} else if v, ok := value.(string); ok {
			var err error
			t.{{ upperFirst .Name }}, err = types.NewDecimalFromString(v)
			if err != nil {
				return fmt.Errorf("invalid decimal value for field {{ .Name }}: %w", err)
			}
		} else if v, ok := value.(float64); ok {
			t.{{ upperFirst .Name }} = types.NewDecimal(v)
		} else if v, ok := value.(int); ok {
			t.{{ upperFirst .Name }} = types.NewDecimalFromInt(int64(v))
		} else if v, ok := value.(int64); ok {
			t.{{ upperFirst .Name }} = types.NewDecimalFromInt(v)
		} else {
			return fmt.Errorf("invalid type for field {{ .Name }} (expected Decimal, string, float64, int, or int64)")
		}
{{- else if eq .Type "types.Date" }}
		if v, ok := value.(types.Date); ok {
			t.{{ upperFirst .Name }} = v
		} else if v, ok := value.(string); ok {
			var err error
			t.{{ upperFirst .Name }}, err = types.NewDateFromString(v)
			if err != nil {
				return fmt.Errorf("invalid date value for field {{ .Name }}: %w", err)
			}
		} else if v, ok := value.(time.Time); ok {
			t.{{ upperFirst .Name }} = types.NewDateFromTime(v)
		} else {
			return fmt.Errorf("invalid type for field {{ .Name }} (expected Date, string, or time.Time)")
		}
{{- else if eq .Type "types.DateTime" }}
		if v, ok := value.(types.DateTime); ok {
			t.{{ upperFirst .Name }} = v
		} else if v, ok := value.(string); ok {
			var err error
			t.{{ upperFirst .Name }}, err = types.NewDateTimeFromString(v)
			if err != nil {
				return fmt.Errorf("invalid datetime value for field {{ .Name }}: %w", err)
			}
		} else if v, ok := value.(time.Time); ok {
			t.{{ upperFirst .Name }} = types.NewDateTimeFromTime(v)
		} else {
			return fmt.Errorf("invalid type for field {{ .Name }} (expected DateTime, string, or time.Time)")
		}
{{- else if eq .Type "[]byte" }}
		// Accept nil, []byte, or base64-encoded string
		if value == nil {
			t.{{ upperFirst .Name }} = nil
		} else if v, ok := value.([]byte); ok {
			t.{{ upperFirst .Name }} = v
		} else if s, ok := value.(string); ok {
			if s == "" {
				t.{{ upperFirst .Name }} = nil
			} else {
				// Try base64 decode
				decoded, err := base64.StdEncoding.DecodeString(s)
				if err != nil {
					return fmt.Errorf("invalid base64 value for field {{ .Name }}: %w", err)
				}
				t.{{ upperFirst .Name }} = decoded
			}
		} else {
			return fmt.Errorf("invalid type for field {{ .Name }} (expected []byte, string, or nil)")
		}
{{- else if eq .Type "bool" }}
		if v, ok := value.(bool); ok {
			t.{{ upperFirst .Name }} = v
		} else if v, ok := value.(string); ok {
			// Handle string boolean values from JSON/frontend
			t.{{ upperFirst .Name }} = v == "true" || v == "1"
		} else {
			return fmt.Errorf("invalid type for field {{ .Name }}")
		}
{{- else if eq .Type "Option" }}
		// Accept enum type directly
		if v, ok := value.({{ $.StructName }}{{ upperFirst .Name }}); ok {
			t.{{ upperFirst .Name }} = v
		// Accept int (convert to enum)
		} else if v, ok := value.(int); ok {
			if v < 0 || v >= {{ len .Options }} {
				return fmt.Errorf("invalid option value %d for field {{ .Name }} (valid range: 0-%d)", v, {{ len .Options }}-1)
			}
			t.{{ upperFirst .Name }} = {{ $.StructName }}{{ upperFirst .Name }}(v)
		// Accept float64 (JSON numbers decode as float64)
		} else if v, ok := value.(float64); ok {
			intVal := int(v)
			if intVal < 0 || intVal >= {{ len .Options }} {
				return fmt.Errorf("invalid option value %d for field {{ .Name }} (valid range: 0-%d)", intVal, {{ len .Options }}-1)
			}
			t.{{ upperFirst .Name }} = {{ $.StructName }}{{ upperFirst .Name }}(intVal)
		// Accept string (lookup in options and convert)
		} else if v, ok := value.(string); ok {
			if v == "" {
				// Empty string defaults to first option (NAV/BC behavior)
				t.{{ upperFirst .Name }} = 0
			} else {
				options := []string{ {{- range $i, $opt := .Options }}{{- if $i }}, {{ end }}"{{ $opt }}"{{- end }} }
				found := false
				for i, opt := range options {
					if opt == v {
						t.{{ upperFirst .Name }} = {{ $.StructName }}{{ upperFirst .Name }}(i)
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("invalid option '%s' for field {{ .Name }} (valid options: %v)", v, options)
				}
			}
		} else {
			return fmt.Errorf("invalid type for field {{ .Name }} (expected {{ $.StructName }}{{ upperFirst .Name }}, int, or string)")
		}
{{- else if eq .Type "int" }}
		switch v := value.(type) {
		case int:
			t.{{ upperFirst .Name }} = v
		case float64:
			t.{{ upperFirst .Name }} = int(v)
		case string:
			if v == "" {
				t.{{ upperFirst .Name }} = 0
			} else if i, err := strconv.Atoi(v); err == nil {
				t.{{ upperFirst .Name }} = i
			} else {
				return fmt.Errorf("invalid integer value for field {{ .Name }}: %s", v)
			}
		default:
			return fmt.Errorf("invalid type for field {{ .Name }}")
		}
{{- else if eq .Type "time.Time" }}
		if v, ok := value.(time.Time); ok {
			t.{{ upperFirst .Name }} = v
		} else {
			return fmt.Errorf("invalid type for field {{ .Name }}")
		}
{{- end }}
		// Call OnValidate trigger
		return t.OnValidate_{{ upperFirst .Name }}()
{{- end }}
{{- end }}
	}

	return fmt.Errorf("field '%s' not found", fieldName)
}

{{- range .Table.Fields }}
{{- if not .FlowField }}

// OnValidate_{{ upperFirst .Name }} is the validation trigger for {{ .Name }} field (BC/NAV style)
// Override this in the wrapper struct to add custom validation
func (t *{{ $.BaseStructName }}) OnValidate_{{ upperFirst .Name }}() error {
	return nil
}
{{- end }}
{{- end }}

// ========================================
// Interface Implementation (tables.Table)
// ========================================

// ClearFilters removes all filters (BC/NAV style, alias for Reset)
func (t *{{ .BaseStructName }}) ClearFilters() {
	t.filters = nil
	t.orderByFields = nil
	// Note: Don't clear oldValues or iteration state here
}

// ToMap converts the current record to a map for JSON serialization
func (t *{{ .BaseStructName }}) ToMap() map[string]interface{} {
	return map[string]interface{}{
{{- range .Table.Fields }}
{{- if not .FlowField }}
{{- if eq .Type "types.Code" }}
		"{{ .DBName }}": t.{{ upperFirst .Name }}.String(),
{{- else if eq .Type "types.Text" }}
		"{{ .DBName }}": t.{{ upperFirst .Name }}.String(),
{{- else if eq .Type "types.Decimal" }}
		"{{ .DBName }}": t.{{ upperFirst .Name }}.String(),
{{- else if eq .Type "types.Date" }}
		"{{ .DBName }}": t.{{ upperFirst .Name }}.String(),
{{- else if eq .Type "types.DateTime" }}
		"{{ .DBName }}": t.{{ upperFirst .Name }}.String(),
{{- else if eq .Type "Option" }}
		"{{ .DBName }}": int(t.{{ upperFirst .Name }}),
{{- else }}
		"{{ .DBName }}": t.{{ upperFirst .Name }},
{{- end }}
{{- else }}
		// FlowField: {{ .DBName }}
{{- if eq .Type "types.Decimal" }}
		"{{ .DBName }}": t.{{ upperFirst .Name }}.String(),
{{- else }}
		"{{ .DBName }}": t.{{ upperFirst .Name }},
{{- end }}
{{- end }}
{{- end }}
	}
}

// FromMap populates the record fields from a map (for API POST/PUT)
func (t *{{ .BaseStructName }}) FromMap(data map[string]interface{}) {
{{- range .Table.Fields }}
{{- if not .FlowField }}
	if v, ok := data["{{ .DBName }}"]; ok && v != nil {
{{- if eq .Type "types.Code" }}
		if s, ok := v.(string); ok {
			t.{{ upperFirst .Name }} = types.NewCode(s)
		}
{{- else if eq .Type "types.Text" }}
		if s, ok := v.(string); ok {
			t.{{ upperFirst .Name }} = types.NewText(s)
		}
{{- else if eq .Type "types.Decimal" }}
		switch val := v.(type) {
		case string:
			t.{{ upperFirst .Name }}, _ = types.NewDecimalFromString(val)
		case float64:
			t.{{ upperFirst .Name }} = types.NewDecimal(val)
		}
{{- else if eq .Type "types.Date" }}
		if s, ok := v.(string); ok {
			t.{{ upperFirst .Name }}, _ = types.NewDateFromString(s)
		}
{{- else if eq .Type "types.DateTime" }}
		if s, ok := v.(string); ok {
			t.{{ upperFirst .Name }}, _ = types.NewDateTimeFromString(s)
		}
{{- else if eq .Type "Option" }}
		switch val := v.(type) {
		case float64:
			t.{{ upperFirst .Name }} = {{ $.StructName }}{{ upperFirst .Name }}(int(val))
		case int:
			t.{{ upperFirst .Name }} = {{ $.StructName }}{{ upperFirst .Name }}(val)
		case string:
			// Lookup string value in options
			options := []string{ {{- range $i, $opt := .Options }}{{- if $i }}, {{ end }}"{{ $opt }}"{{- end }} }
			for i, opt := range options {
				if opt == val {
					t.{{ upperFirst .Name }} = {{ $.StructName }}{{ upperFirst .Name }}(i)
					break
				}
			}
		}
{{- else if eq .Type "bool" }}
		if b, ok := v.(bool); ok {
			t.{{ upperFirst .Name }} = b
		}
{{- else if eq .Type "int" }}
		switch val := v.(type) {
		case float64:
			t.{{ upperFirst .Name }} = int(val)
		case int:
			t.{{ upperFirst .Name }} = val
		}
{{- else if eq .Type "int64" }}
		switch val := v.(type) {
		case float64:
			t.{{ upperFirst .Name }} = int64(val)
		case int64:
			t.{{ upperFirst .Name }} = val
		case int:
			t.{{ upperFirst .Name }} = int64(val)
		}
{{- else if eq .Type "float64" }}
		if f, ok := v.(float64); ok {
			t.{{ upperFirst .Name }} = f
		}
{{- else if eq .Type "string" }}
		if s, ok := v.(string); ok {
			t.{{ upperFirst .Name }} = s
		}
{{- end }}
	}
{{- end }}
{{- end }}
}

// UpdateFromMap updates only the provided fields (for PATCH-style updates)
func (t *{{ .BaseStructName }}) UpdateFromMap(data map[string]interface{}) {
	// Same as FromMap - only updates fields present in the map
	t.FromMap(data)
}

// GetPrimaryKeyField returns the name of the primary key field
func (t *{{ .BaseStructName }}) GetPrimaryKeyField() string {
{{- if .FirstPrimaryKey }}
	return "{{ .FirstPrimaryKey.DBName }}"
{{- else }}
	return ""
{{- end }}
}

// GetPrimaryKeyValue returns the current primary key value as a string
func (t *{{ .BaseStructName }}) GetPrimaryKeyValue() string {
{{- if .FirstPrimaryKey }}
{{- if eq .FirstPrimaryKey.Type "types.Code" }}
	return t.{{ upperFirst .FirstPrimaryKey.Name }}.String()
{{- else if eq .FirstPrimaryKey.Type "types.Text" }}
	return t.{{ upperFirst .FirstPrimaryKey.Name }}.String()
{{- else if eq .FirstPrimaryKey.Type "int" }}
	return fmt.Sprintf("%d", t.{{ upperFirst .FirstPrimaryKey.Name }})
{{- else if eq .FirstPrimaryKey.Type "int64" }}
	return fmt.Sprintf("%d", t.{{ upperFirst .FirstPrimaryKey.Name }})
{{- else if eq .FirstPrimaryKey.Type "string" }}
	return t.{{ upperFirst .FirstPrimaryKey.Name }}
{{- else }}
	return fmt.Sprintf("%v", t.{{ upperFirst .FirstPrimaryKey.Name }})
{{- end }}
{{- else }}
	return ""
{{- end }}
}

// GetFields returns metadata about all fields
func (t *{{ .BaseStructName }}) GetFields() []tables.FieldInfo {
	return []tables.FieldInfo{
{{- range .Table.Fields }}
		{
			Name:       "{{ .DBName }}",
{{- if eq .Type "types.Code" }}
			Type:       tables.FieldTypeCode,
{{- else if eq .Type "types.Text" }}
			Type:       tables.FieldTypeText,
{{- else if eq .Type "int" }}
			Type:       tables.FieldTypeInteger,
{{- else if eq .Type "int64" }}
			Type:       tables.FieldTypeInteger,
{{- else if eq .Type "types.Decimal" }}
			Type:       tables.FieldTypeDecimal,
{{- else if eq .Type "float64" }}
			Type:       tables.FieldTypeDecimal,
{{- else if eq .Type "bool" }}
			Type:       tables.FieldTypeBoolean,
{{- else if eq .Type "types.Date" }}
			Type:       tables.FieldTypeDate,
{{- else if eq .Type "types.DateTime" }}
			Type:       tables.FieldTypeDateTime,
{{- else if eq .Type "Option" }}
			Type:       tables.FieldTypeOption,
{{- else if or (eq .Type "[]byte") (eq .Type "BLOB") }}
			Type:       tables.FieldTypeBlob,
{{- else }}
			Type:       tables.FieldTypeText,
{{- end }}
			Length:     {{ .Length }},
			Required:   {{ .Required }},
			Editable:   {{ not .PrimaryKey }},
			PrimaryKey: {{ .PrimaryKey }},
			FlowField:  {{ .FlowField }},
		},
{{- end }}
	}
}

// GetFlowFields returns names of FlowFields that need CalcFields
func (t *{{ .BaseStructName }}) GetFlowFields() []string {
	return []string{
{{- range .Table.Fields }}
{{- if .FlowField }}
		"{{ .DBName }}",
{{- end }}
{{- end }}
	}
}

// GetOptionFields returns Option field names mapped to their option values
func (t *{{ .BaseStructName }}) GetOptionFields() map[string][]string {
	return map[string][]string{
{{- range .Table.Fields }}
{{- if eq .Type "Option" }}
		"{{ .DBName }}": { {{- range $i, $opt := .Options }}{{ if $i }}, {{ end }}"{{ $opt }}"{{- end }} },
{{- end }}
{{- end }}
	}
}

// GetTableRelationFields returns fields that have table relations (foreign keys)
func (t *{{ .BaseStructName }}) GetTableRelationFields() map[string]tables.TableRelationInfo {
	return map[string]tables.TableRelationInfo{
{{- range .Table.Fields }}
{{- if .TableRelation }}
		"{{ .DBName }}": {
			Table:        "{{ .TableRelation.Table }}",
			Field:        "{{ .TableRelation.Field }}",
			DisplayField: "{{ .TableRelation.DisplayField }}",
{{- if .TableRelation.LookupColumns }}
			LookupColumns: []tables.LookupColumnInfo{
{{- range .TableRelation.LookupColumns }}
				{Source: "{{ .Source }}", Width: {{ .Width }}},
{{- end }}
			},
{{- end }}
			SearchTimeout: {{ .TableRelation.SearchTimeout }},
		},
{{- end }}
{{- end }}
	}
}
`

const businessTemplate = `package {{ .PackageName }}

import (
	"errors"
{{- if .HasTimeField }}
	"time"
{{- end }}

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// {{ .StructName }} wraps {{ .BaseStructName }} and adds trigger methods
type {{ .StructName }} struct {
	gtables.{{ .BaseStructName }}
}

// New{{ .StructName }} creates a new {{ .StructName }} instance
func New{{ .StructName }}() *{{ .StructName }} {
	return &{{ .StructName }}{}
}

// Init initializes the record with database context and sets up triggers
func (t *{{ .StructName }}) Init(db database.Executor, company string) {
	t.{{ .BaseStructName }}.Init(db, company)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *{{ .StructName }}) OnInsert() error {
{{- range .Table.Fields }}
{{- if .AutoTimestamp }}
	t.{{ upperFirst .Name }} = time.Now()
{{- end }}
{{- end }}
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *{{ .StructName }}) OnModify() error {
{{- range .Table.Fields }}
{{- if .AutoTimestamp }}
	t.{{ upperFirst .Name }} = time.Now()
{{- end }}
{{- end }}
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *{{ .StructName }}) OnDelete(db database.Executor, company string) error {
	// TODO: Add checks for related records (if any)
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *{{ .StructName }}) OnRename() error {
{{- range .Table.Fields }}
{{- if .AutoTimestamp }}
	t.{{ upperFirst .Name }} = time.Now()
{{- end }}
{{- end }}
	// TODO: Update related records if needed
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *{{ .StructName }}) Validate() error {
{{- range .Table.Fields }}
{{- if .Required }}
	{{- if eq .Type "types.Code" }}
	if t.{{ upperFirst .Name }}.IsEmpty() {
		return errors.New("{{ .Name }} is required")
	}
	{{- else if eq .Type "string" }}
	if t.{{ upperFirst .Name }} == "" {
		return errors.New("{{ .Name }} is required")
	}
	{{- end }}
{{- end }}
{{- if .Length }}
	if len(t.{{ upperFirst .Name }}) > {{ .Length }} {
		return errors.New("{{ .Name }} cannot exceed {{ .Length }} characters")
	}
{{- end }}
{{- if .Validation }}
	if t.{{ upperFirst .Name }} < {{ .Validation.Min }} || t.{{ upperFirst .Name }} > {{ .Validation.Max }} {
		return errors.New("{{ .Name }} must be between {{ .Validation.Min }} and {{ .Validation.Max }}")
	}
{{- end }}
{{- end }}

	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
// Example:
// func (t *{{ .StructName }}) CalculateSomething() error {
//     // Your logic here
//     return nil
// }
`
