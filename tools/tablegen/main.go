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
	Name         string      `yaml:"name"`
	Type         string      `yaml:"type"`
	DBName       string      `yaml:"db_name"`
	PrimaryKey   bool        `yaml:"primary_key"`
	Length       int         `yaml:"length"`
	Required     bool        `yaml:"required"`
	Default      interface{} `yaml:"default"`
	AutoTimestamp bool       `yaml:"auto_timestamp"`
	Validation   *Validation `yaml:"validation"`
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
	{{ .Name }} {{ .Type }} ` + "`db:\"{{ .DBName }}{{if .PrimaryKey}},pk{{end}}\"`" + `
{{- end }}
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

{{- range .Table.Fields }}

// {{ upperFirst .Name }} returns the {{ .Name }} field
func (t *{{ $.StructName }}) {{ upperFirst .Name }}() {{ .Type }} {
	return t.{{ .Name }}
}

// Set{{ upperFirst .Name }} sets the {{ .Name }} field
func (t *{{ $.StructName }}) Set{{ upperFirst .Name }}(value {{ .Type }}) error {
{{- if .Validation }}
	if value < {{ .Validation.Min }} || value > {{ .Validation.Max }} {
		return fmt.Errorf("{{ .Name }} must be between {{ .Validation.Min }} and {{ .Validation.Max }}")
	}
{{- end }}
{{- if .Required }}
	{{- if eq .Type "types.Code" }}
	if value.IsEmpty() {
		return fmt.Errorf("{{ .Name }} cannot be empty")
	}
	{{- else if eq .Type "string" }}
	if value == "" {
		return fmt.Errorf("{{ .Name }} cannot be empty")
	}
	{{- end }}
{{- end }}
{{- if .Length }}
	if len(value) > {{ .Length }} {
		return fmt.Errorf("{{ .Name }} cannot exceed {{ .Length }} characters")
	}
{{- end }}
	t.{{ .Name }} = value
	return nil
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
	t.{{ .Name }} = time.Now()
{{- end }}
{{- end }}
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *{{ .StructName }}) OnModify() error {
{{- range .Table.Fields }}
{{- if .AutoTimestamp }}
	t.{{ .Name }} = time.Now()
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
	t.{{ .Name }} = time.Now()
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
	if t.{{ .Name }}.IsEmpty() {
		return errors.New("{{ .Name }} is required")
	}
	{{- else if eq .Type "string" }}
	if t.{{ .Name }} == "" {
		return errors.New("{{ .Name }} is required")
	}
	{{- end }}
{{- end }}
{{- if .Length }}
	if len(t.{{ .Name }}) > {{ .Length }} {
		return errors.New("{{ .Name }} cannot exceed {{ .Length }} characters")
	}
{{- end }}
{{- if .Validation }}
	if t.{{ .Name }} < {{ .Validation.Min }} || t.{{ .Name }} > {{ .Validation.Max }} {
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
