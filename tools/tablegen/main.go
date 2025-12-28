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

// TableRelation represents a foreign key relationship to another table
type TableRelation struct {
	Table string `yaml:"table"`
	Field string `yaml:"field"`
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
	PackageName      string
	HasTimeField     bool
	HasCodeField     bool
	HasTextField     bool
	HasOptionField   bool
	HasDecimalField  bool
	HasDateField     bool
	HasDateTimeField bool
	HasFlowField     bool
}

func main() {
	// Get current working directory (should be business-logic/tables when run via go generate)
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// The definitions directory is in the same directory as where go generate is run
	tablesDir := cwd
	defsDir := filepath.Join(tablesDir, "definitions")

	// Ensure definitions directory exists
	if err := os.MkdirAll(defsDir, 0755); err != nil {
		fmt.Printf("Error creating definitions directory: %v\n", err)
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

		// Generate *_gen.go (always regenerate)
		genFile := filepath.Join(tablesDir, strings.ToLower(data.StructName)+"_gen.go")
		if err := generateBoilerplate(genFile, data); err != nil {
			fmt.Printf("  ✗ Error generating boilerplate: %v\n", err)
			continue
		}
		fmt.Printf("  ✓ Generated: %s\n", filepath.Base(genFile))

		// Generate *.go skeleton (only if doesn't exist)
		businessFile := filepath.Join(tablesDir, strings.ToLower(data.StructName)+".go")
		if !fileExists(businessFile) {
			if err := generateBusinessLogicSkeleton(businessFile, data); err != nil {
				fmt.Printf("  ✗ Error generating skeleton: %v\n", err)
				continue
			}
			fmt.Printf("  ✓ Created skeleton: %s\n", filepath.Base(businessFile))
		} else {
			fmt.Printf("  ⊙ Skipped (exists): %s\n", filepath.Base(businessFile))
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
	data := TemplateData{
		TableDef:    *def,
		StructName:  toPascalCase(def.Table.Name),
		PackageName: "tables",
	}

	// Check which imports are needed
	for _, field := range def.Table.Fields {
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

// templateFuncs returns template helper functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"upperFirst":         upperFirst,
		"lowerFirst":         lowerFirst,
		"sqlType":            getSQLType,
		"isLast":             isLast,
		"isLastPK":           isLastPK,
		"isLastDBField":      isLastDBField,
		"hasSuffix":          strings.HasSuffix,
		"join":               strings.Join,
		"sub":                func(a, b int) int { return a - b },
		"sanitizeIdentifier": sanitizeIdentifier,
	}
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
	"fmt"
	"strings"
{{- if or .HasTimeField .HasDateField .HasDateTimeField }}
	"time"
{{- end }}
{{- if or .HasCodeField .HasTextField .HasDecimalField .HasDateField .HasDateTimeField }}

	"github.com/hansjlachmann/openerp/src/foundation/types"
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

// {{ .StructName }} represents Table {{ .Table.ID }}: {{ .Table.Name }}
type {{ .StructName }} struct {
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
	db      *sql.DB
	company string

	// Field tracking for optimal Modify() operations
	oldValues map[string]interface{} // Stores original values from Get()

	// Filter state for SetRange/FindFirst/FindLast (BC/NAV style)
	filters map[string]*{{ lowerFirst .StructName }}FilterCondition

	// Iteration state for FindSet/Next (BC/NAV style)
	currentRows *sql.Rows
	orderByFields []string

	// Buffered recordset for bidirectional navigation (BC/NAV style)
	bufferedRecords []*{{ .StructName }}
	currentBufferPos int
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
func (t *{{ .StructName }}) GetTableID() int {
	return {{ .StructName }}TableID
}

// GetTableName returns the table name
func (t *{{ .StructName }}) GetTableName() string {
	return {{ .StructName }}TableName
}

// GetTableSchema returns the CREATE TABLE schema
func (t *{{ .StructName }}) GetTableSchema() string {
	return Get{{ .StructName }}TableSchema()
}

// Get{{ .StructName }}TableSchema returns the SQLite schema
func Get{{ .StructName }}TableSchema() string {
	return ` + "`" + `
{{- range $i, $f := .Table.Fields }}
		{{ $f.DBName }} {{ sqlType $f }}{{ if $f.PrimaryKey }} PRIMARY KEY{{ end }}{{ if $f.Required }}{{ if not $f.PrimaryKey }} NOT NULL{{ end }}{{ end }}{{ if $f.Validation }} CHECK ({{ $f.DBName }} >= {{ $f.Validation.Min }} AND {{ $f.DBName }} <= {{ $f.Validation.Max }}){{ end }}{{ if eq $f.Type "Option" }} CHECK ({{ $f.DBName }} >= 0 AND {{ $f.DBName }} <= {{ sub (len $f.Options) 1 }}){{ end }}{{ if $f.Default }} DEFAULT {{ $f.Default }}{{ end }}{{ if $f.AutoTimestamp }} DEFAULT CURRENT_TIMESTAMP{{ end }}{{ if not (isLast $i $.Table.Fields) }},{{ end }}
{{- end }}
	` + "`" + `
}

// CreateTable creates the {{ .Table.Name }} table for the specified company
func (t *{{ .StructName }}) CreateTable(db *sql.DB, company string) error {
	tableName := fmt.Sprintf("%s$%s", company, {{ .StructName }}TableName)
	schema := Get{{ .StructName }}TableSchema()

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
func (t *{{ .StructName }}) Init(db *sql.DB, company string) {
	t.db = db
	t.company = company
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
func (t *{{ .StructName }}) StoreOldValues() {
	t.oldValues = make(map[string]interface{})
{{- range .Table.Fields }}
{{- if not .FlowField }}
	t.oldValues["{{ .DBName }}"] = t.{{ upperFirst .Name }}
{{- end }}
{{- end }}
}

// Get retrieves a record from the database by primary key
func (t *{{ .StructName }}) Get({{- range $i, $f := .Table.Fields }}{{- if $f.PrimaryKey }}{{ lowerFirst $f.Name }} {{ $f.Type }}{{ if not (isLastPK $i $.Table.Fields) }}, {{ end }}{{- end }}{{- end }}) bool {
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)

	{{- range .Table.Fields }}
	{{- if not .FlowField }}
	{{- if eq .Type "types.Code" }}
	var {{ lowerFirst .Name }}Str string
	{{- else if eq .Type "types.Text" }}
	var {{ lowerFirst .Name }}Str string
	{{- else if eq .Type "types.Decimal" }}
	var {{ lowerFirst .Name }}Str string
	{{- else if eq .Type "types.Date" }}
	var {{ lowerFirst .Name }}Str string
	{{- else if eq .Type "types.DateTime" }}
	var {{ lowerFirst .Name }}Str string
	{{- else if eq .Type "bool" }}
	var {{ lowerFirst .Name }}Int int
	{{- else if eq .Type "Option" }}
	var {{ lowerFirst .Name }}Int int
	{{- else }}
	var {{ lowerFirst .Name }}Val {{ .Type }}
	{{- end }}
	{{- end }}
	{{- end }}

	err := t.db.QueryRow(
		fmt.Sprintf(` + "`SELECT {{ range $i, $f := .Table.Fields }}{{ if not $f.FlowField }}{{ $f.DBName }}{{ if not (isLastDBField $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }} FROM \"%s\" WHERE 1=1{{ range .Table.Fields }}{{ if .PrimaryKey }} AND {{ .DBName }} = ?{{ end }}{{ end }}`" + `, tableName),
		{{- range $i, $f := .Table.Fields }}{{- if $f.PrimaryKey }}
		{{ lowerFirst $f.Name }},
		{{- end }}{{- end }}
	).Scan(
{{- range $i, $f := .Table.Fields }}
		{{- if not $f.FlowField }}
		{{- if eq $f.Type "types.Code" }}
		&{{ lowerFirst $f.Name }}Str,
		{{- else if eq $f.Type "types.Text" }}
		&{{ lowerFirst $f.Name }}Str,
		{{- else if eq $f.Type "types.Decimal" }}
		&{{ lowerFirst $f.Name }}Str,
		{{- else if eq $f.Type "types.Date" }}
		&{{ lowerFirst $f.Name }}Str,
		{{- else if eq $f.Type "types.DateTime" }}
		&{{ lowerFirst $f.Name }}Str,
		{{- else if eq $f.Type "bool" }}
		&{{ lowerFirst $f.Name }}Int,
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
	t.{{ upperFirst .Name }} = types.NewCode({{ lowerFirst .Name }}Str)
{{- else if eq .Type "types.Text" }}
	t.{{ upperFirst .Name }} = types.NewText({{ lowerFirst .Name }}Str)
{{- else if eq .Type "types.Decimal" }}
	t.{{ upperFirst .Name }}, _ = types.NewDecimalFromString({{ lowerFirst .Name }}Str)
{{- else if eq .Type "types.Date" }}
	t.{{ upperFirst .Name }}, _ = types.NewDateFromString({{ lowerFirst .Name }}Str)
{{- else if eq .Type "types.DateTime" }}
	t.{{ upperFirst .Name }}, _ = types.NewDateTimeFromString({{ lowerFirst .Name }}Str)
{{- else if eq .Type "bool" }}
	t.{{ upperFirst .Name }} = {{ lowerFirst .Name }}Int != 0
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
func (t *{{ .StructName }}) Insert(runTrigger bool) bool {
	// Call OnInsert trigger if requested
	if runTrigger {
		if err := t.OnInsert(); err != nil {
			fmt.Printf("Error: OnInsert trigger failed: %v\n", err)
			return false
		}
	}

	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
	_, err := t.db.Exec(
		fmt.Sprintf(` + "`INSERT INTO \"%s\" ({{ range $i, $f := .Table.Fields }}{{ if not $f.FlowField }}{{ $f.DBName }}{{ if not (isLastDBField $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }}) VALUES ({{ range $i, $f := .Table.Fields }}{{ if not $f.FlowField }}?{{ if not (isLastDBField $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }})`" + `, tableName),
{{- range .Table.Fields }}
{{- if not .FlowField }}
		t.{{ upperFirst .Name }},
{{- end }}
{{- end }}
	)
	if err != nil {
		fmt.Printf("Error: Failed to insert {{ .Table.Name }}: %v\n", err)
		return false
	}
	return true
}

// Modify updates the record in the database
func (t *{{ .StructName }}) Modify(runTrigger bool) bool {
	// Call OnModify trigger if requested
	if runTrigger {
		if err := t.OnModify(); err != nil {
			fmt.Printf("Error: OnModify trigger failed: %v\n", err)
			return false
		}
	}

	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)

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
	sql := fmt.Sprintf(` + "`UPDATE \"%s\" SET %s WHERE {{ range .Table.Fields }}{{ if .PrimaryKey }}{{ .DBName }} = ?{{ end }}{{ end }}`" + `,
		tableName,
		strings.Join(setClauses, ", "),
	)

	_, err := t.db.Exec(sql, values...)
	if err != nil {
		fmt.Printf("Error: Failed to modify {{ .Table.Name }}: %v\n", err)
		return false
	}
	return true
}

// hasFieldChanged checks if a field value has changed from oldValues
func (t *{{ .StructName }}) hasFieldChanged(fieldName string) bool {
	if t.oldValues == nil {
		return true // No old values, assume changed
	}

	oldValue, exists := t.oldValues[fieldName]
	if !exists {
		return true // Field not in old values, assume changed
	}

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
{{- end }}
		return true // Type mismatch, assume changed
{{- end }}
{{- end }}
	}

	return false
}

// Delete removes the record from the database
func (t *{{ .StructName }}) Delete(runTrigger bool) bool {
	// Call OnDelete trigger if requested
	if runTrigger {
		if err := t.OnDelete(t.db, t.company); err != nil {
			fmt.Printf("Error: OnDelete trigger failed: %v\n", err)
			return false
		}
	}

	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
	_, err := t.db.Exec(
		fmt.Sprintf(` + "`DELETE FROM \"%s\" WHERE {{ range .Table.Fields }}{{ if .PrimaryKey }}{{ .DBName }} = ?{{ end }}{{ end }}`" + `, tableName),
{{- range .Table.Fields }}
{{- if .PrimaryKey }}
		t.{{ upperFirst .Name }},
{{- end }}
{{- end }}
	)
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
func (t *{{ .StructName }}) CalcFields(fieldNames ...string) {
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
func (t *{{ $.StructName }}) calcFlowField_{{ .Name }}() {
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

func (t *{{ $.StructName }}) calcSum{{ upperFirst .SourceTable }}{{ upperFirst .SourceField }}() {{ .Type }} {
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

func (t *{{ $.StructName }}) calcCount{{ upperFirst .SourceTable }}() int {
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

{{- end }}

// ========================================
// BC/NAV-style Filtering and Search
// ========================================

// {{ lowerFirst .StructName }}FilterCondition represents a filter on a field
type {{ lowerFirst .StructName }}FilterCondition struct {
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
func (t *{{ .StructName }}) SetRange(fieldName string, values ...interface{}) {
	if t.filters == nil {
		t.filters = make(map[string]*{{ lowerFirst .StructName }}FilterCondition)
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

	t.filters[fieldName] = &{{ lowerFirst .StructName }}FilterCondition{
		fieldName: fieldName,
		minValue:  minValue,
		maxValue:  maxValue,
	}
}

// SetFilter sets a complex filter expression on a field (BC/NAV style)
// Supports BC/NAV filter syntax: "100..200|500" (range OR exact value)
// Operators: .. (range), | (OR), & (AND), * (wildcard), <> (not equal)
// Example: customer.SetFilter("No", "001..003|005")
func (t *{{ .StructName }}) SetFilter(fieldName, filterExpr string) {
	if t.filters == nil {
		t.filters = make(map[string]*{{ lowerFirst .StructName }}FilterCondition)
	}
	t.filters[fieldName] = &{{ lowerFirst .StructName }}FilterCondition{
		fieldName:    fieldName,
		filterExpr:   filterExpr,
		isExpression: true,
	}
}

// SetCurrentKey sets the sort order for queries (BC/NAV style)
// Example: customer.SetCurrentKey("City", "Name")
func (t *{{ .StructName }}) SetCurrentKey(fields ...string) {
	t.orderByFields = fields
}

// Reset clears all filters (BC/NAV style)
func (t *{{ .StructName }}) Reset() {
	t.filters = nil
	t.oldValues = nil
	t.orderByFields = nil
	if t.currentRows != nil {
		t.currentRows.Close()
		t.currentRows = nil
	}
}

// buildWhereClause builds WHERE clause from current filters
func (t *{{ .StructName }}) buildWhereClause() (string, []interface{}) {
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
func (t *{{ .StructName }}) parseFilterExpression(fieldName, expr string) (string, []interface{}) {
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
func (t *{{ .StructName }}) getOrderByClause() string {
	if len(t.orderByFields) > 0 {
		return strings.Join(t.orderByFields, ", ")
	}
	// Default: order by primary key
	return "{{ range .Table.Fields }}{{ if .PrimaryKey }}{{ .DBName }}{{ end }}{{ end }}"
}

// FindFirst finds the first record matching current filters (BC/NAV style)
// Returns true if found, false if not found
func (t *{{ .StructName }}) FindFirst() bool {
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
	where, args := t.buildWhereClause()

	// Build SELECT with all fields
	query := fmt.Sprintf(` + "`SELECT {{ range $i, $f := .Table.Fields }}{{ if not $f.FlowField }}{{ $f.DBName }}{{ if not (isLastDBField $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }} FROM \"%s\" WHERE %s ORDER BY {{ range .Table.Fields }}{{ if .PrimaryKey }}{{ .DBName }}{{ end }}{{ end }} ASC LIMIT 1`" + `, tableName, where)

{{- range .Table.Fields }}
{{- if not .FlowField }}
{{- if eq .Type "types.Code" }}
	var {{ .Name }}Str string
{{- else if eq .Type "types.Text" }}
	var {{ .Name }}Str string
{{- else if eq .Type "types.Decimal" }}
	var {{ .Name }}Str string
{{- else if eq .Type "types.Date" }}
	var {{ .Name }}Str string
{{- else if eq .Type "types.DateTime" }}
	var {{ .Name }}Str string
{{- else if eq .Type "bool" }}
	var {{ .Name }}Int int
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
		&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.Text" }}
		&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.Decimal" }}
		&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.Date" }}
		&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.DateTime" }}
		&{{ $f.Name }}Str,
{{- else if eq $f.Type "bool" }}
		&{{ $f.Name }}Int,
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
	t.{{ upperFirst .Name }} = types.NewCode({{ .Name }}Str)
{{- else if eq .Type "types.Text" }}
	t.{{ upperFirst .Name }} = types.NewText({{ .Name }}Str)
{{- else if eq .Type "types.Decimal" }}
	t.{{ upperFirst .Name }}, _ = types.NewDecimalFromString({{ .Name }}Str)
{{- else if eq .Type "types.Date" }}
	t.{{ upperFirst .Name }}, _ = types.NewDateFromString({{ .Name }}Str)
{{- else if eq .Type "types.DateTime" }}
	t.{{ upperFirst .Name }}, _ = types.NewDateTimeFromString({{ .Name }}Str)
{{- else if eq .Type "bool" }}
	t.{{ upperFirst .Name }} = {{ .Name }}Int != 0
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
func (t *{{ .StructName }}) FindLast() bool {
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
	where, args := t.buildWhereClause()

	// Build SELECT with all fields
	query := fmt.Sprintf(` + "`SELECT {{ range $i, $f := .Table.Fields }}{{ if not $f.FlowField }}{{ $f.DBName }}{{ if not (isLastDBField $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }} FROM \"%s\" WHERE %s ORDER BY {{ range .Table.Fields }}{{ if .PrimaryKey }}{{ .DBName }}{{ end }}{{ end }} DESC LIMIT 1`" + `, tableName, where)

{{- range .Table.Fields }}
{{- if not .FlowField }}
{{- if eq .Type "types.Code" }}
	var {{ .Name }}Str string
{{- else if eq .Type "types.Text" }}
	var {{ .Name }}Str string
{{- else if eq .Type "types.Decimal" }}
	var {{ .Name }}Str string
{{- else if eq .Type "types.Date" }}
	var {{ .Name }}Str string
{{- else if eq .Type "types.DateTime" }}
	var {{ .Name }}Str string
{{- else if eq .Type "bool" }}
	var {{ .Name }}Int int
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
		&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.Text" }}
		&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.Decimal" }}
		&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.Date" }}
		&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.DateTime" }}
		&{{ $f.Name }}Str,
{{- else if eq $f.Type "bool" }}
		&{{ $f.Name }}Int,
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
	t.{{ upperFirst .Name }} = types.NewCode({{ .Name }}Str)
{{- else if eq .Type "types.Text" }}
	t.{{ upperFirst .Name }} = types.NewText({{ .Name }}Str)
{{- else if eq .Type "types.Decimal" }}
	t.{{ upperFirst .Name }}, _ = types.NewDecimalFromString({{ .Name }}Str)
{{- else if eq .Type "types.Date" }}
	t.{{ upperFirst .Name }}, _ = types.NewDateFromString({{ .Name }}Str)
{{- else if eq .Type "types.DateTime" }}
	t.{{ upperFirst .Name }}, _ = types.NewDateTimeFromString({{ .Name }}Str)
{{- else if eq .Type "bool" }}
	t.{{ upperFirst .Name }} = {{ .Name }}Int != 0
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
func (t *{{ .StructName }}) Count() int {
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
	where, args := t.buildWhereClause()

	query := fmt.Sprintf(` + "`SELECT COUNT(*) FROM \"%s\" WHERE %s`" + `, tableName, where)

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
func (t *{{ .StructName }}) FindSet() bool {
	// Close any existing result set
	if t.currentRows != nil {
		t.currentRows.Close()
		t.currentRows = nil
	}

	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
	where, args := t.buildWhereClause()
	orderBy := t.getOrderByClause()

	// Build SELECT with all fields
	query := fmt.Sprintf(` + "`SELECT {{ range $i, $f := .Table.Fields }}{{ if not $f.FlowField }}{{ $f.DBName }}{{ if not (isLastDBField $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }} FROM \"%s\" WHERE %s ORDER BY %s`" + `, tableName, where, orderBy)

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
func (t *{{ .StructName }}) Next(steps ...int) bool {
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
		var {{ .Name }}Str string
{{- else if eq .Type "types.Text" }}
		var {{ .Name }}Str string
{{- else if eq .Type "types.Decimal" }}
		var {{ .Name }}Str string
{{- else if eq .Type "types.Date" }}
		var {{ .Name }}Str string
{{- else if eq .Type "types.DateTime" }}
		var {{ .Name }}Str string
{{- else if eq .Type "bool" }}
		var {{ .Name }}Int int
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
			&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.Text" }}
			&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.Decimal" }}
			&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.Date" }}
			&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.DateTime" }}
			&{{ $f.Name }}Str,
{{- else if eq $f.Type "bool" }}
			&{{ $f.Name }}Int,
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
		t.{{ upperFirst .Name }} = types.NewCode({{ .Name }}Str)
{{- else if eq .Type "types.Text" }}
		t.{{ upperFirst .Name }} = types.NewText({{ .Name }}Str)
{{- else if eq .Type "types.Decimal" }}
		t.{{ upperFirst .Name }}, _ = types.NewDecimalFromString({{ .Name }}Str)
{{- else if eq .Type "types.Date" }}
		t.{{ upperFirst .Name }}, _ = types.NewDateFromString({{ .Name }}Str)
{{- else if eq .Type "types.DateTime" }}
		t.{{ upperFirst .Name }}, _ = types.NewDateTimeFromString({{ .Name }}Str)
{{- else if eq .Type "bool" }}
		t.{{ upperFirst .Name }} = {{ .Name }}Int != 0
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
func (t *{{ .StructName }}) FindSetBuffered() bool {
	// Close any existing forward-only result set
	if t.currentRows != nil {
		t.currentRows.Close()
		t.currentRows = nil
	}

	// Clear any existing buffer
	t.bufferedRecords = nil
	t.currentBufferPos = -1

	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
	where, args := t.buildWhereClause()
	orderBy := t.getOrderByClause()

	// Build SELECT with all fields
	query := fmt.Sprintf(` + "`SELECT {{ range $i, $f := .Table.Fields }}{{ if not $f.FlowField }}{{ $f.DBName }}{{ if not (isLastDBField $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }} FROM \"%s\" WHERE %s ORDER BY %s`" + `, tableName, where, orderBy)

	rows, err := t.db.Query(query, args...)
	if err != nil {
		fmt.Printf("Error: Failed to execute FindSetBuffered for {{ .Table.Name }}: %v\n", err)
		return false
	}
	defer rows.Close()

	// Load all records into memory
	for rows.Next() {
		// Create a new record instance
		record := &{{ .StructName }}{}
		record.db = t.db
		record.company = t.company

		// Scan the row
{{- range .Table.Fields }}
{{- if not .FlowField }}
{{- if eq .Type "types.Code" }}
		var {{ .Name }}Str string
{{- else if eq .Type "types.Text" }}
		var {{ .Name }}Str string
{{- else if eq .Type "types.Decimal" }}
		var {{ .Name }}Str string
{{- else if eq .Type "types.Date" }}
		var {{ .Name }}Str string
{{- else if eq .Type "types.DateTime" }}
		var {{ .Name }}Str string
{{- else if eq .Type "bool" }}
		var {{ .Name }}Int int
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
			&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.Text" }}
			&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.Decimal" }}
			&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.Date" }}
			&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.DateTime" }}
			&{{ $f.Name }}Str,
{{- else if eq $f.Type "bool" }}
			&{{ $f.Name }}Int,
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
		record.{{ upperFirst .Name }} = types.NewCode({{ .Name }}Str)
{{- else if eq .Type "types.Text" }}
		record.{{ upperFirst .Name }} = types.NewText({{ .Name }}Str)
{{- else if eq .Type "types.Decimal" }}
		record.{{ upperFirst .Name }}, _ = types.NewDecimalFromString({{ .Name }}Str)
{{- else if eq .Type "types.Date" }}
		record.{{ upperFirst .Name }}, _ = types.NewDateFromString({{ .Name }}Str)
{{- else if eq .Type "types.DateTime" }}
		record.{{ upperFirst .Name }}, _ = types.NewDateTimeFromString({{ .Name }}Str)
{{- else if eq .Type "bool" }}
		record.{{ upperFirst .Name }} = {{ .Name }}Int != 0
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
func (t *{{ .StructName }}) copyFromBuffered(record *{{ .StructName }}) {
{{- range .Table.Fields }}
	t.{{ upperFirst .Name }} = record.{{ upperFirst .Name }}
{{- end }}
	t.StoreOldValues()
}

// ========================================
// Phase 3: Advanced BC/NAV Methods
// ========================================

// IsEmpty returns true if no records match current filters (BC/NAV style)
func (t *{{ .StructName }}) IsEmpty() bool {
	return t.Count() == 0
}

// ModifyAll updates a field for all records matching current filters (BC/NAV style)
// Returns the number of records modified
func (t *{{ .StructName }}) ModifyAll(fieldName string, newValue interface{}) int {
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
	where, args := t.buildWhereClause()

	// Build UPDATE SQL
	updateSQL := fmt.Sprintf(` + "`UPDATE \"%s\" SET %s = ? WHERE %s`" + `, tableName, fieldName, where)

	// Prepend newValue to args
	allArgs := append([]interface{}{newValue}, args...)

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
func (t *{{ .StructName }}) DeleteAll() int {
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
	where, args := t.buildWhereClause()

	// Build DELETE SQL
	deleteSQL := fmt.Sprintf(` + "`DELETE FROM \"%s\" WHERE %s`" + `, tableName, where)

	result, err := t.db.Exec(deleteSQL, args...)
	if err != nil {
		fmt.Printf("Error: Failed to delete all {{ .Table.Name }}: %v\n", err)
		return 0
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected)
}

// CopyFilters copies filters from another record variable (BC/NAV style)
func (t *{{ .StructName }}) CopyFilters(from *{{ .StructName }}) {
	if from.filters == nil {
		t.filters = nil
		return
	}

	// Deep copy filters
	t.filters = make(map[string]*{{ lowerFirst .StructName }}FilterCondition)
	for key, filter := range from.filters {
		t.filters[key] = &{{ lowerFirst .StructName }}FilterCondition{
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
func (t *{{ .StructName }}) GetFilters() string {
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
func (t *{{ .StructName }}) ValidateField(fieldName string, value interface{}) error {
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
		if v, ok := value.([]byte); ok {
			t.{{ upperFirst .Name }} = v
		} else {
			return fmt.Errorf("invalid type for field {{ .Name }} (expected []byte)")
		}
{{- else if eq .Type "bool" }}
		if v, ok := value.(bool); ok {
			t.{{ upperFirst .Name }} = v
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
		// Accept string (lookup in options and convert)
		} else if v, ok := value.(string); ok {
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
		} else {
			return fmt.Errorf("invalid type for field {{ .Name }} (expected {{ $.StructName }}{{ upperFirst .Name }}, int, or string)")
		}
{{- else if eq .Type "int" }}
		if v, ok := value.(int); ok {
			t.{{ upperFirst .Name }} = v
		} else {
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
// All validation logic is in {{ lowerFirst $.StructName }}.go - CustomValidate_{{ upperFirst .Name }}()
func (t *{{ $.StructName }}) OnValidate_{{ upperFirst .Name }}() error {
	return t.CustomValidate_{{ upperFirst .Name }}()
}
{{- end }}
{{- end }}
`

const businessTemplate = `package {{ .PackageName }}

import (
	"database/sql"
	"errors"
{{- if .HasTimeField }}
	"time"
{{- end }}
)

//go:generate go run ../../../tools/tablegen/main.go

// New{{ .StructName }} creates a new {{ .StructName }} instance
func New{{ .StructName }}() *{{ .StructName }} {
	return &{{ .StructName }}{
{{- range .Table.Fields }}
{{- if .AutoTimestamp }}
		{{ .Name }}: time.Now(),
{{- end }}
{{- end }}
	}
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
func (t *{{ .StructName }}) OnDelete(db *sql.DB, company string) error {
	// TODO: Add checks for related records (if any)
	// Example:
	// var count int
	// err := db.QueryRow(
	//     fmt.Sprintf(` + "`SELECT COUNT(*) FROM \"%s$OtherTable\" WHERE {{ .StructName | lowerFirst }}_code = $1`" + `, company),
	//     t.primaryKeyValue,
	// ).Scan(&count)
	// if count > 0 {
	//     return fmt.Errorf("cannot delete: {{ .Table.Name }} is used by %d records", count)
	// }

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
// Field Validation Hooks
// ========================================
// These methods are called by auto-generated OnValidate triggers in {{ lowerFirst .StructName }}_gen.go
// Add your custom field validation logic here

{{- range .Table.Fields }}
{{- if not .FlowField }}

// CustomValidate_{{ upperFirst .Name }} - Custom validation for {{ .Name }} field
func (t *{{ $.StructName }}) CustomValidate_{{ upperFirst .Name }}() error {
{{- if .TableRelation }}
	// Table relation validation - {{ .Name }} must exist in {{ .TableRelation.Table }}
	if t.{{ upperFirst .Name }} != "" && t.{{ upperFirst .Name }} != types.{{ if eq .Type "types.Code" }}Code{{ else }}Text{{ end }}("") {
		var relatedRecord {{ .TableRelation.Table }}
		relatedRecord.Init(t.db, t.company)
		if !relatedRecord.Get(t.{{ upperFirst .Name }}) {
			return errors.New("{{ .Name }} does not exist in {{ .TableRelation.Table }} table")
		}

		// *** ADD YOUR CUSTOM LOGIC HERE ***
		// You can access the related record:
		// if !relatedRecord.Active {
		//     return errors.New("{{ .TableRelation.Table }} is inactive")
		// }
	}
{{- else }}
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for {{ .Name }}:
	// if len(t.{{ upperFirst .Name }}) < 3 {
	//     return errors.New("{{ .Name }} must be at least 3 characters")
	// }
{{- end }}

	return nil
}
{{- end }}
{{- end }}

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
