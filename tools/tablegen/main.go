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
	} `yaml:"table"`
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
		"upperFirst": upperFirst,
		"lowerFirst": lowerFirst,
		"sqlType":    getSQLType,
		"isLast":     isLast,
		"isLastPK":   isLastPK,
		"hasSuffix":  strings.HasSuffix,
		"join":       strings.Join,
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
{{- if .HasTimeField }}
	"time"
{{- end }}
{{- if or .HasCodeField .HasTextField }}

	"github.com/hansjlachmann/openerp/src/foundation/types"
{{- end }}
)

// {{ .StructName }} represents Table {{ .Table.ID }}: {{ .Table.Name }}
type {{ .StructName }} struct {
{{- range .Table.Fields }}
	{{ upperFirst .Name }} {{ .Type }} ` + "`db:\"{{ .DBName }}{{if .PrimaryKey}},pk{{end}}\"`" + `
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
}

const {{ .StructName }}TableID = {{ .Table.ID }}
const {{ .StructName }}TableName = "{{ .Table.Name }}"

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
		{{ $f.DBName }} {{ sqlType $f }}{{ if $f.PrimaryKey }} PRIMARY KEY{{ end }}{{ if $f.Required }}{{ if not $f.PrimaryKey }} NOT NULL{{ end }}{{ end }}{{ if $f.Validation }} CHECK ({{ $f.DBName }} >= {{ $f.Validation.Min }} AND {{ $f.DBName }} <= {{ $f.Validation.Max }}){{ end }}{{ if $f.Default }} DEFAULT {{ $f.Default }}{{ end }}{{ if $f.AutoTimestamp }} DEFAULT CURRENT_TIMESTAMP{{ end }}{{ if not (isLast $i $.Table.Fields) }},{{ end }}
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
	t.oldValues["{{ .DBName }}"] = t.{{ upperFirst .Name }}
{{- end }}
}

// Get retrieves a record from the database by primary key
func (t *{{ .StructName }}) Get({{- range $i, $f := .Table.Fields }}{{- if $f.PrimaryKey }}{{ lowerFirst $f.Name }} {{ $f.Type }}{{ if not (isLastPK $i $.Table.Fields) }}, {{ end }}{{- end }}{{- end }}) bool {
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)

	{{- range .Table.Fields }}
	{{- if eq .Type "types.Code" }}
	var {{ lowerFirst .Name }}Str string
	{{- else if eq .Type "types.Text" }}
	var {{ lowerFirst .Name }}Str string
	{{- else if eq .Type "bool" }}
	var {{ lowerFirst .Name }}Int int
	{{- else }}
	var {{ lowerFirst .Name }}Val {{ .Type }}
	{{- end }}
	{{- end }}

	err := t.db.QueryRow(
		fmt.Sprintf(` + "`SELECT {{ range $i, $f := .Table.Fields }}{{ $f.DBName }}{{ if not (isLast $i $.Table.Fields) }}, {{ end }}{{ end }} FROM \"%s\" WHERE 1=1{{ range .Table.Fields }}{{ if .PrimaryKey }} AND {{ .DBName }} = ?{{ end }}{{ end }}`" + `, tableName),
		{{- range $i, $f := .Table.Fields }}{{- if $f.PrimaryKey }}
		{{ lowerFirst $f.Name }},
		{{- end }}{{- end }}
	).Scan(
{{- range $i, $f := .Table.Fields }}
		{{- if eq $f.Type "types.Code" }}
		&{{ lowerFirst $f.Name }}Str,
		{{- else if eq $f.Type "types.Text" }}
		&{{ lowerFirst $f.Name }}Str,
		{{- else if eq $f.Type "bool" }}
		&{{ lowerFirst $f.Name }}Int,
		{{- else }}
		&{{ lowerFirst $f.Name }}Val,
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
{{- if eq .Type "types.Code" }}
	t.{{ upperFirst .Name }} = types.NewCode({{ lowerFirst .Name }}Str)
{{- else if eq .Type "types.Text" }}
	t.{{ upperFirst .Name }} = types.NewText({{ lowerFirst .Name }}Str)
{{- else if eq .Type "bool" }}
	t.{{ upperFirst .Name }} = {{ lowerFirst .Name }}Int != 0
{{- else }}
	t.{{ upperFirst .Name }} = {{ lowerFirst .Name }}Val
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
		fmt.Sprintf(` + "`INSERT INTO \"%s\" ({{ range $i, $f := .Table.Fields }}{{ $f.DBName }}{{ if not (isLast $i $.Table.Fields) }}, {{ end }}{{ end }}) VALUES ({{ range $i, $f := .Table.Fields }}?{{ if not (isLast $i $.Table.Fields) }}, {{ end }}{{ end }})`" + `, tableName),
{{- range .Table.Fields }}
		t.{{ upperFirst .Name }},
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
{{- if not .PrimaryKey }}
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
{{- if not .PrimaryKey }}
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
{{- if not .PrimaryKey }}
	case "{{ .DBName }}":
		if old, ok := oldValue.({{ .Type }}); ok {
			return t.{{ upperFirst .Name }} != old
		}
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
	query := fmt.Sprintf(` + "`SELECT {{ range $i, $f := .Table.Fields }}{{ $f.DBName }}{{ if not (isLast $i $.Table.Fields) }}, {{ end }}{{ end }} FROM \"%s\" WHERE %s ORDER BY {{ range .Table.Fields }}{{ if .PrimaryKey }}{{ .DBName }}{{ end }}{{ end }} ASC LIMIT 1`" + `, tableName, where)

{{- range .Table.Fields }}
{{- if eq .Type "types.Code" }}
	var {{ .Name }}Str string
{{- else if eq .Type "types.Text" }}
	var {{ .Name }}Str string
{{- else if eq .Type "bool" }}
	var {{ .Name }}Int int
{{- else if eq .Type "time.Time" }}
	var {{ .Name }}Time time.Time
{{- end }}
{{- end }}

	err := t.db.QueryRow(query, args...).Scan(
{{- range $i, $f := .Table.Fields }}
{{- if eq $f.Type "types.Code" }}
		&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.Text" }}
		&{{ $f.Name }}Str,
{{- else if eq $f.Type "bool" }}
		&{{ $f.Name }}Int,
{{- else if eq $f.Type "time.Time" }}
		&{{ $f.Name }}Time,
{{- else }}
		&t.{{ upperFirst $f.Name }},
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
{{- if eq .Type "types.Code" }}
	t.{{ upperFirst .Name }} = types.NewCode({{ .Name }}Str)
{{- else if eq .Type "types.Text" }}
	t.{{ upperFirst .Name }} = types.NewText({{ .Name }}Str)
{{- else if eq .Type "bool" }}
	t.{{ upperFirst .Name }} = {{ .Name }}Int != 0
{{- else if eq .Type "time.Time" }}
	t.{{ upperFirst .Name }} = {{ .Name }}Time
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
	query := fmt.Sprintf(` + "`SELECT {{ range $i, $f := .Table.Fields }}{{ $f.DBName }}{{ if not (isLast $i $.Table.Fields) }}, {{ end }}{{ end }} FROM \"%s\" WHERE %s ORDER BY {{ range .Table.Fields }}{{ if .PrimaryKey }}{{ .DBName }}{{ end }}{{ end }} DESC LIMIT 1`" + `, tableName, where)

{{- range .Table.Fields }}
{{- if eq .Type "types.Code" }}
	var {{ .Name }}Str string
{{- else if eq .Type "types.Text" }}
	var {{ .Name }}Str string
{{- else if eq .Type "bool" }}
	var {{ .Name }}Int int
{{- else if eq .Type "time.Time" }}
	var {{ .Name }}Time time.Time
{{- end }}
{{- end }}

	err := t.db.QueryRow(query, args...).Scan(
{{- range $i, $f := .Table.Fields }}
{{- if eq $f.Type "types.Code" }}
		&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.Text" }}
		&{{ $f.Name }}Str,
{{- else if eq $f.Type "bool" }}
		&{{ $f.Name }}Int,
{{- else if eq $f.Type "time.Time" }}
		&{{ $f.Name }}Time,
{{- else }}
		&t.{{ upperFirst $f.Name }},
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
{{- if eq .Type "types.Code" }}
	t.{{ upperFirst .Name }} = types.NewCode({{ .Name }}Str)
{{- else if eq .Type "types.Text" }}
	t.{{ upperFirst .Name }} = types.NewText({{ .Name }}Str)
{{- else if eq .Type "bool" }}
	t.{{ upperFirst .Name }} = {{ .Name }}Int != 0
{{- else if eq .Type "time.Time" }}
	t.{{ upperFirst .Name }} = {{ .Name }}Time
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
	query := fmt.Sprintf(` + "`SELECT {{ range $i, $f := .Table.Fields }}{{ $f.DBName }}{{ if not (isLast $i $.Table.Fields) }}, {{ end }}{{ end }} FROM \"%s\" WHERE %s ORDER BY %s`" + `, tableName, where, orderBy)

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
// Must be called after FindSet()
// Returns true if a record was loaded, false if no more records
func (t *{{ .StructName }}) Next() bool {
	if t.currentRows == nil {
		return false
	}

	// Try to advance to next row
	if !t.currentRows.Next() {
		// No more rows - close result set
		t.currentRows.Close()
		t.currentRows = nil
		return false
	}

	// Scan the row
{{- range .Table.Fields }}
{{- if eq .Type "types.Code" }}
	var {{ .Name }}Str string
{{- else if eq .Type "types.Text" }}
	var {{ .Name }}Str string
{{- else if eq .Type "bool" }}
	var {{ .Name }}Int int
{{- else if eq .Type "time.Time" }}
	var {{ .Name }}Time time.Time
{{- end }}
{{- end }}

	err := t.currentRows.Scan(
{{- range $i, $f := .Table.Fields }}
{{- if eq $f.Type "types.Code" }}
		&{{ $f.Name }}Str,
{{- else if eq $f.Type "types.Text" }}
		&{{ $f.Name }}Str,
{{- else if eq $f.Type "bool" }}
		&{{ $f.Name }}Int,
{{- else if eq $f.Type "time.Time" }}
		&{{ $f.Name }}Time,
{{- else }}
		&t.{{ upperFirst $f.Name }},
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
{{- if eq .Type "types.Code" }}
	t.{{ upperFirst .Name }} = types.NewCode({{ .Name }}Str)
{{- else if eq .Type "types.Text" }}
	t.{{ upperFirst .Name }} = types.NewText({{ .Name }}Str)
{{- else if eq .Type "bool" }}
	t.{{ upperFirst .Name }} = {{ .Name }}Int != 0
{{- else if eq .Type "time.Time" }}
	t.{{ upperFirst .Name }} = {{ .Name }}Time
{{- end }}
{{- end }}

	// Store old values for field tracking
	t.StoreOldValues()

	return true
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
{{- else if eq .Type "bool" }}
		if v, ok := value.(bool); ok {
			t.{{ upperFirst .Name }} = v
		} else {
			return fmt.Errorf("invalid type for field {{ .Name }}")
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
	}

	return fmt.Errorf("field '%s' not found", fieldName)
}

{{- range .Table.Fields }}

// OnValidate_{{ upperFirst .Name }} is the validation trigger for {{ .Name }} field (BC/NAV style)
// All validation logic is in {{ lowerFirst $.StructName }}.go - CustomValidate_{{ upperFirst .Name }}()
func (t *{{ $.StructName }}) OnValidate_{{ upperFirst .Name }}() error {
	return t.CustomValidate_{{ upperFirst .Name }}()
}
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
