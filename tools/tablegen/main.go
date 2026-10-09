package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// TableDef represents a table definition from YAML
type TableDef struct {
	Table struct {
		ID         int     `yaml:"id"`
		Name       string  `yaml:"name"`
		Global     bool    `yaml:"global"`      // If true, table is global (no company prefix)
		SetupTable bool    `yaml:"setup_table"` // If true, BC-style singleton setup table (single blank-PK record)
		Fields     []Field `yaml:"fields"`
		Keys       []Key   `yaml:"keys"`
	} `yaml:"table"`

	// FlowFilterFields (derived): fields with flow_filter: true, taken out of Fields after
	// parsing — they are not stored columns, the record only keeps their filter expression
	FlowFilterFields []Field `yaml:"-"`

	// References (derived): per primary key field (db name), the fields of other tables
	// whose table_relation points to it — updated when the record is renamed (BC Rename)
	References map[string][]Reference `yaml:"-"`
}

// Reference is a field of a table that refers to another table's key field
type Reference struct {
	Table  string // referencing table name (e.g. "Customer Ledger Entry")
	Global bool   // referencing table has no company prefix
	Column string // referencing column (e.g. "customer_no")
}

// Key represents an index/key on a table (BC/NAV style)
type Key struct {
	Name      string   `yaml:"name"`      // Key name (e.g., "customer_open")
	Fields    []string `yaml:"fields"`    // Fields in the key (e.g., ["customer_no", "open"])
	Unique    bool     `yaml:"unique"`    // Whether this is a UNIQUE index
	Clustered bool     `yaml:"clustered"` // Primary key-like behavior (BC/NAV concept)
	// SumIndexFields (BC/NAV SIFT): the database keeps a totals table for this key with
	// these fields summed, and FlowFields over the key's fields read the totals
	SumIndexFields []string `yaml:"sum_index_fields"`
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
	Options       []string       `yaml:"options"`      // For Option type fields (enum values)
	Precision     int            `yaml:"precision"`    // For Decimal type (total digits)
	Scale         int            `yaml:"scale"`        // For Decimal type (decimal places)
	FlowField     bool           `yaml:"flow_field"`   // For FlowFields (calculated fields)
	CalcFormula   string         `yaml:"calc_formula"` // Sum, Count, Lookup, Exist, Average, Min, Max
	SourceTable   string         `yaml:"source_table"` // Table to calculate from
	SourceField   string         `yaml:"source_field"` // Field to aggregate
	FlowFilters   []FlowFilter   `yaml:"flow_filters"` // Filter conditions
	// FlowFilter (NAV FieldClass FlowFilter): not stored; holds a filter expression the
	// user sets (e.g. Date Filter), applied by FlowFields with a flow filter of type "filter"
	FlowFilter bool `yaml:"flow_filter"`
	// Sensitive: never sent by the table API (e.g. User.password_hash), and not usable in
	// filters, sorting, search or lookup columns; the API cannot set it either
	Sensitive bool `yaml:"sensitive"`
	// Masked (BC ExtendedDatatype Masked): a secret that can be set through the API but is
	// never read back — responses carry tables.MaskedValue when it has a value (e.g.
	// SMTP_Setup.password); not usable in filters, sorting, search or lookup columns
	Masked bool `yaml:"masked"`
	// Encrypted: a masked secret the server has to read back in clear (e.g. the SMTP
	// password it logs in with) is stored encrypted (backend/foundation/secrets, AES-256-GCM)
	// — Insert/Modify/InsertAll encrypt it, Plain<Field>() decrypts it. Requires masked and
	// types.Text. Not for values that are only compared (user passwords: hash them, sensitive).
	Encrypted bool `yaml:"encrypted"`

	// Derived (not in YAML): for a FlowField with exactly one "field" flow filter, the
	// source column to group by and the record map key holding its value. Set by
	// prepareTemplateData; empty means CalcFieldsForRecords calculates per record.
	BatchKeyField string `yaml:"-"`
	BatchKeyValue string `yaml:"-"`
	// SIFTKey (derived): the source table's SIFT key this FlowField reads its totals from
	// ("" = sums the entries)
	SIFTKey string `yaml:"-"`
	// SIFTKeyFiltered (derived): the SIFT key to read when a FlowFilter of this FlowField is
	// set (its key must contain the filtered fields too); "" = sums the entries then
	SIFTKeyFiltered string `yaml:"-"`
	// FilterRefs (derived): the flow filters of type "filter" (FlowFilter fields applied)
	FilterRefs []FlowFilter `yaml:"-"`
}

// FlowFilter represents a filter condition for FlowField calculation
type FlowFilter struct {
	Field string `yaml:"field"` // Field name in source table
	Type  string `yaml:"type"`  // "const", "field" or "filter" (apply a FlowFilter field of this table)
	Value string `yaml:"value"` // Constant value, or field / FlowFilter field name of the current table

	// Derived for type "filter": the FlowFilter field's struct field and flowfilter kind
	FilterField string `yaml:"-"`
	Kind        string `yaml:"-"`
	// Derived for type "field": the database column of the current table's field
	ValueColumn string `yaml:"-"`
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
	FirstPrimaryKey  *Field        // First primary key field (for GetPrimaryKeyField/Value)
	SIFTKeys         []SIFTKeyData // this table's keys with sum_index_fields
	UsesSIFT         bool          // imports the sift package (own SIFT keys or FlowFields reading totals)
	UsesFlowFilter   bool          // imports the flowfilter package (table has FlowFilter fields)
	EncryptedFields  []Field       // fields stored encrypted (encrypted: true)
	// HasFilterableFlowField: Sum/Count FlowFields lists can filter on (FlowFieldFilterKind)
	HasFilterableFlowField bool
}

// SIFTKeyData is a key with sum index fields, as the templates need it
type SIFTKeyData struct {
	Name   string
	Fields []SIFTColumn
	Sums   []SIFTColumn
}

// SIFTColumn is a key or sum column of a SIFT key (column name and sift.Kind)
type SIFTColumn struct {
	Name string
	Kind string
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

	// Pass 1: read every definition. A FlowField reads the totals of a SIFT key on its
	// source table, so FlowFields can only be resolved when all tables are known.
	type parsed struct {
		file string
		def  *TableDef
	}
	var defs []parsed
	byStruct := map[string]*TableDef{}
	for _, yamlFile := range yamlFiles {
		tableDef, err := parseYAML(yamlFile)
		if err != nil {
			fmt.Printf("✗ Error parsing %s: %v\n", filepath.Base(yamlFile), err)
			continue
		}
		splitFlowFilterFields(tableDef)
		defs = append(defs, parsed{yamlFile, tableDef})
		byStruct[toPascalCase(tableDef.Table.Name)] = tableDef
	}
	failed := false
	for _, p := range defs {
		if err := validateSIFTKeys(p.def); err != nil {
			fmt.Printf("✗ %s: %v\n", filepath.Base(p.file), err)
			failed = true
		}
		if err := validateFlowFilters(p.def, byStruct); err != nil {
			fmt.Printf("✗ %s: %v\n", filepath.Base(p.file), err)
			failed = true
		}
		if err := validateSensitiveFields(p.def, byStruct); err != nil {
			fmt.Printf("✗ %s: %v\n", filepath.Base(p.file), err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
	for _, p := range defs {
		resolveFlowFieldSIFT(p.def, byStruct)
	}
	resolveReferences(defs, func(d parsed) *TableDef { return d.def })

	// Pass 2: generate
	for _, p := range defs {
		yamlFile, tableDef := p.file, p.def
		fmt.Printf("\nProcessing: %s\n", filepath.Base(yamlFile))

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
		if field.Encrypted {
			data.EncryptedFields = append(data.EncryptedFields, *field)
		}
		// Track first primary key field (for tables with composite keys)
		if field.PrimaryKey && data.FirstPrimaryKey == nil {
			data.FirstPrimaryKey = field
		}
	}

	// FlowFields with a single "field" flow filter can be calculated for many records
	// with one grouped query (CalcFieldsForRecords)
	dbNames := make(map[string]string, len(def.Table.Fields))
	for _, f := range def.Table.Fields {
		dbNames[f.Name] = f.DBName
	}
	for i := range def.Table.Fields {
		field := &def.Table.Fields[i]
		if !field.FlowField {
			continue
		}
		var keyFilters []FlowFilter
		for _, ff := range field.FlowFilters {
			if ff.Type == "field" {
				keyFilters = append(keyFilters, ff)
			}
		}
		if len(keyFilters) == 1 {
			field.BatchKeyField = keyFilters[0].Field
			field.BatchKeyValue = dbNames[keyFilters[0].Value]
			if field.BatchKeyValue == "" {
				field.BatchKeyValue = keyFilters[0].Value
			}
		}
	}
	data.TableDef = *def

	// SIFT keys of this table, and whether the sift package is needed
	fieldsByName := map[string]Field{}
	for _, f := range def.Table.Fields {
		fieldsByName[f.Name] = f
	}
	for _, k := range def.Table.Keys {
		if len(k.SumIndexFields) == 0 {
			continue
		}
		kd := SIFTKeyData{Name: k.Name}
		for _, name := range k.Fields {
			f := fieldsByName[name]
			kd.Fields = append(kd.Fields, SIFTColumn{Name: f.DBName, Kind: siftKind(f)})
		}
		for _, name := range k.SumIndexFields {
			f := fieldsByName[name]
			kd.Sums = append(kd.Sums, SIFTColumn{Name: f.DBName, Kind: siftKind(f)})
		}
		data.SIFTKeys = append(data.SIFTKeys, kd)
	}
	data.UsesSIFT = true // SyncKeys always calls sift.Sync (drops totals of removed keys)
	for _, f := range def.Table.Fields {
		if f.FlowField && (f.CalcFormula == "Sum" || f.CalcFormula == "Count") {
			data.HasFilterableFlowField = true
		}
	}
	data.UsesFlowFilter = len(def.FlowFilterFields) > 0 || data.HasFilterableFlowField

	return data
}

// siftKind maps a field type to the sift package's column kind
func siftKind(f Field) string {
	switch f.Type {
	case "int", "int64", "Option":
		return "KindInt"
	case "bool":
		return "KindBool"
	case "types.Date":
		return "KindDate"
	case "types.DateTime", "time.Time":
		return "KindDateTime"
	case "types.Decimal", "float64":
		return "KindDecimal"
	default:
		return "KindText"
	}
}

// normalizedTableName compares table names written as registry names ("Payment_terms") and
// as display names ("Payment Terms")
func normalizedTableName(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "_"))
}

// resolveReferences records, for each table's primary key fields, the fields of other tables
// whose table_relation points to them (renaming a record updates those, BC/NAV Rename).
func resolveReferences[T any](items []T, defOf func(T) *TableDef) {
	byName := map[string]*TableDef{}
	for _, it := range items {
		d := defOf(it)
		byName[normalizedTableName(d.Table.Name)] = d
		d.References = nil
	}
	for _, it := range items {
		d := defOf(it)
		for _, f := range d.Table.Fields {
			if f.TableRelation == nil || f.FlowField {
				continue
			}
			target, ok := byName[normalizedTableName(f.TableRelation.Table)]
			if !ok {
				continue
			}
			for _, tf := range target.Table.Fields {
				if tf.PrimaryKey && (tf.Name == f.TableRelation.Field || tf.DBName == f.TableRelation.Field) {
					if target.References == nil {
						target.References = map[string][]Reference{}
					}
					target.References[tf.DBName] = append(target.References[tf.DBName], Reference{Table: d.Table.Name, Global: d.Table.Global, Column: f.DBName})
				}
			}
		}
	}
}

// splitFlowFilterFields moves FlowFilter fields out of Fields (they are not stored).
func splitFlowFilterFields(def *TableDef) {
	stored := def.Table.Fields[:0]
	for _, f := range def.Table.Fields {
		if f.FlowFilter {
			def.FlowFilterFields = append(def.FlowFilterFields, f)
		} else {
			stored = append(stored, f)
		}
	}
	def.Table.Fields = stored
}

// flowFilterKind maps a FlowFilter field's type to the flowfilter package kind
func flowFilterKind(f Field) (string, bool) {
	switch f.Type {
	case "types.Date":
		return "KindDate", true
	case "bool":
		return "KindBool", true
	case "int", "int64":
		return "KindInt", true
	case "types.Text":
		return "KindText", true
	case "types.Code":
		return "KindCode", true
	}
	return "", false
}

// validateFlowFilters checks FlowFilter field types and the "filter" flow filters of
// FlowFields (FlowFilter field of this table, stored field of the source table), and fills
// their derived struct field / kind.
func validateFlowFilters(def *TableDef, byStruct map[string]*TableDef) error {
	filters := map[string]Field{}
	for _, f := range def.FlowFilterFields {
		if _, ok := flowFilterKind(f); !ok {
			return fmt.Errorf("FlowFilter field %q has type %s; supported: types.Date, bool, int, types.Text, types.Code", f.Name, f.Type)
		}
		filters[f.Name] = f
	}
	for i := range def.Table.Fields {
		f := &def.Table.Fields[i]
		if !f.FlowField {
			continue
		}
		for j := range f.FlowFilters {
			ff := &f.FlowFilters[j]
			if ff.Type == "field" {
				for _, own := range def.Table.Fields {
					if own.Name == ff.Value && !own.FlowField && !own.FlowFilter {
						ff.ValueColumn = own.DBName
					}
				}
				if ff.ValueColumn == "" {
					return fmt.Errorf("FlowField %s: flow filter on %s uses %q, which is not a stored field of %s", f.Name, ff.Field, ff.Value, def.Table.Name)
				}
			}
			if ff.Type != "filter" {
				continue
			}
			filter, ok := filters[ff.Value]
			if !ok {
				return fmt.Errorf("FlowField %s: flow filter on %s uses %q, which is not a FlowFilter field of %s", f.Name, ff.Field, ff.Value, def.Table.Name)
			}
			if source, ok := byStruct[toPascalCase(f.SourceTable)]; ok {
				found := false
				for _, sf := range source.Table.Fields {
					if sf.Name == ff.Field && !sf.FlowField {
						found = true
					}
				}
				if !found {
					return fmt.Errorf("FlowField %s: %q is not a stored field of %s", f.Name, ff.Field, source.Table.Name)
				}
			}
			ff.FilterField = upperFirst(filter.Name)
			ff.Kind, _ = flowFilterKind(filter)
		}
	}
	return nil
}

// encryptedLength is the stored length of a secret of n bytes: "enc:v1:" + base64 without
// padding of nonce (12) + secret + GCM tag (16). Must match secrets.EncryptedLength.
func encryptedLength(n int) int {
	return len("enc:v1:") + (4*(12+n+16)+2)/3
}

// maxEncryptedPlain is the longest secret (bytes) whose encrypted form fits a column of
// the given length
func maxEncryptedPlain(length int) int {
	n := 0
	for encryptedLength(n+1) <= length {
		n++
	}
	return n
}

// validateSensitiveFields checks that sensitive and masked fields are plain stored fields
// (not a key, FlowField or FlowFilter; masked ones Text/Code) and that no table relation
// shows one of another table in its dropdown — the dropdown rows would send it to the client.
func validateSensitiveFields(def *TableDef, byStruct map[string]*TableDef) error {
	for _, f := range def.Table.Fields {
		if f.Sensitive && (f.PrimaryKey || f.FlowField || f.FlowFilter) {
			return fmt.Errorf("sensitive field %q cannot be a primary key, FlowField or FlowFilter", f.Name)
		}
		if f.Masked && (f.PrimaryKey || f.FlowField || f.FlowFilter || f.Sensitive) {
			return fmt.Errorf("masked field %q cannot be a primary key, FlowField, FlowFilter or sensitive", f.Name)
		}
		if f.Masked && f.Type != "types.Text" && f.Type != "types.Code" {
			return fmt.Errorf("masked field %q must be types.Text or types.Code, not %s", f.Name, f.Type)
		}
		if f.Encrypted && (!f.Masked || f.Type != "types.Text") {
			return fmt.Errorf("encrypted field %q must be masked and types.Text (an encrypted value is never sent by the API)", f.Name)
		}
		if f.Encrypted && maxEncryptedPlain(f.Length) < 16 {
			return fmt.Errorf("encrypted field %q: length %d is too short for an encrypted value (at least %d)", f.Name, f.Length, encryptedLength(16))
		}
	}
	for _, f := range def.Table.Fields {
		rel := f.TableRelation
		if rel == nil {
			continue
		}
		related, ok := byStruct[toPascalCase(rel.Table)]
		if !ok {
			continue
		}
		shown := []string{rel.Field, rel.DisplayField}
		for _, col := range rel.LookupColumns {
			shown = append(shown, col.Source)
		}
		for _, rf := range related.Table.Fields {
			if !rf.Sensitive && !rf.Masked {
				continue
			}
			for _, name := range shown {
				if name == rf.Name {
					return fmt.Errorf("table relation of %s shows secret (sensitive/masked) field %s.%s", f.Name, related.Table.Name, rf.Name)
				}
			}
		}
	}
	return nil
}

// validateSIFTKeys checks keys with sum_index_fields: key and sum fields must be stored
// fields of the table, sum fields numeric.
func validateSIFTKeys(def *TableDef) error {
	fields := map[string]Field{}
	for _, f := range def.Table.Fields {
		fields[f.Name] = f
	}
	for _, k := range def.Table.Keys {
		if len(k.SumIndexFields) == 0 {
			continue
		}
		for _, name := range k.Fields {
			f, ok := fields[name]
			if !ok || f.FlowField {
				return fmt.Errorf("key %s: field %q is not a stored field of %s", k.Name, name, def.Table.Name)
			}
		}
		for _, name := range k.SumIndexFields {
			f, ok := fields[name]
			if !ok || f.FlowField {
				return fmt.Errorf("key %s: sum index field %q is not a stored field of %s", k.Name, name, def.Table.Name)
			}
			switch f.Type {
			case "types.Decimal", "int", "int64", "float64":
			default:
				return fmt.Errorf("key %s: sum index field %q has type %s; only numbers can be summed", k.Name, name, f.Type)
			}
		}
	}
	return nil
}

// resolveFlowFieldSIFT picks, for each Sum/Count FlowField, a SIFT key on its source table
// whose fields contain every flow filter field and (Sum) whose sum index fields contain
// the summed field — the key with the fewest fields. Without one the FlowField sums the
// entries; when the source table has SIFT keys that is reported, as it is likely a
// removed sum index field.
func resolveFlowFieldSIFT(def *TableDef, byStruct map[string]*TableDef) {
	for i := range def.Table.Fields {
		f := &def.Table.Fields[i]
		f.SIFTKey, f.SIFTKeyFiltered, f.FilterRefs = "", "", nil
		for _, ff := range f.FlowFilters {
			if ff.Type == "filter" {
				f.FilterRefs = append(f.FilterRefs, ff)
			}
		}
		if !f.FlowField || (f.CalcFormula != "Sum" && f.CalcFormula != "Count") {
			continue
		}
		source, ok := byStruct[toPascalCase(f.SourceTable)]
		if !ok {
			continue
		}
		// Without FlowFilters set: the fixed (const/field) filters; with: all filter fields
		var fixed, all []string
		for _, ff := range f.FlowFilters {
			all = append(all, ff.Field)
			if ff.Type != "filter" {
				fixed = append(fixed, ff.Field)
			}
		}
		var hasSIFT bool
		f.SIFTKey, hasSIFT = pickSIFTKey(source, f, fixed)
		if f.SIFTKey == "" && hasSIFT {
			fmt.Printf("⚠ %s.%s: no SIFT key of %s covers it, sums the entries\n", def.Table.Name, f.Name, source.Table.Name)
		}
		if len(f.FilterRefs) > 0 {
			f.SIFTKeyFiltered, _ = pickSIFTKey(source, f, all)
			if f.SIFTKeyFiltered == "" && hasSIFT {
				fmt.Printf("⚠ %s.%s: no SIFT key of %s covers it with its FlowFilters, sums the entries when one is set\n", def.Table.Name, f.Name, source.Table.Name)
			}
		}
	}
}

// pickSIFTKey returns the SIFT key of source with the fewest fields that contains all
// filterFields and (Sum) sums f's source field; hasSIFT reports whether source has any.
func pickSIFTKey(source *TableDef, f *Field, filterFields []string) (best string, hasSIFT bool) {
	bestLen := 0
	for _, k := range source.Table.Keys {
		if len(k.SumIndexFields) == 0 {
			continue
		}
		hasSIFT = true
		if f.CalcFormula == "Sum" && !slices.Contains(k.SumIndexFields, f.SourceField) {
			continue
		}
		covers := true
		for _, field := range filterFields {
			if !slices.Contains(k.Fields, field) {
				covers = false
				break
			}
		}
		if covers && (best == "" || len(k.Fields) < bestLen) {
			best, bestLen = k.Name, len(k.Fields)
		}
	}
	return best, hasSIFT
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
		"filterSetExpr":      filterSetExpr,
		"flowFilterKind":     func(f Field) string { k, _ := flowFilterKind(f); return k },
		"maxEncryptedPlain":  maxEncryptedPlain,
	}
}

// filterSetExpr is the Go condition "one of these FlowFilters has a value"
func filterSetExpr(refs []FlowFilter) string {
	var conds []string
	for _, r := range refs {
		conds = append(conds, fmt.Sprintf("t.%s != \"\"", r.FilterField))
	}
	return strings.Join(conds, " || ")
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
{{- define "flowSource" }}
	// Source: the SIFT totals table of a key covering the filters, else the entries
	tableName, useSIFT := fmt.Sprintf("%s$%s", t.company, {{ toPascalCase .SourceTable }}TableName), false
{{- if .SIFTKey }}
	tableName, useSIFT = sift.TableName(t.company, {{ toPascalCase .SourceTable }}TableName, "{{ .SIFTKey }}"), true
{{- end }}
{{- if .FilterRefs }}
	if {{ filterSetExpr .FilterRefs }} {
		// A FlowFilter is set: the key must contain the filtered fields too
{{- if .SIFTKeyFiltered }}
		tableName, useSIFT = sift.TableName(t.company, {{ toPascalCase .SourceTable }}TableName, "{{ .SIFTKeyFiltered }}"), true
{{- else }}
		tableName, useSIFT = fmt.Sprintf("%s$%s", t.company, {{ toPascalCase .SourceTable }}TableName), false
{{- end }}
	}
{{- end }}
	_ = useSIFT
{{- end }}
{{- define "flowFilterWhere" }}
{{- range .FlowFilters }}
{{- if eq .Type "filter" }}
	if t.{{ .FilterField }} != "" {
		// FlowFilter {{ .Value }} applied to {{ .Field }} (validated by SetFlowFilter)
		if clause, filterArgs, _ := flowfilter.Clause("{{ .Field }}", flowfilter.{{ .Kind }}, t.{{ .FilterField }}); clause != "" {
			whereClauses = append(whereClauses, clause)
			args = append(args, filterArgs...)
		}
	}
{{- end }}
{{- end }}
{{- end }}

package {{ .PackageName }}

import (
	"database/sql"
{{- if .HasBlobField }}
	"encoding/base64"
{{- end }}
{{- if .EncryptedFields }}
	"errors"
{{- end }}
	"fmt"
	"slices"
{{- if .HasIntField }}
	"strconv"
{{- end }}
	"strings"
{{- if or .HasTimeField .HasDateField .HasDateTimeField }}
	"time"
{{- end }}

	"github.com/hansjlachmann/openerp/backend/foundation/database"
{{- if .EncryptedFields }}
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
{{- end }}
	"github.com/hansjlachmann/openerp/backend/foundation/i18n"
{{- if .UsesFlowFilter }}
	"github.com/hansjlachmann/openerp/backend/foundation/flowfilter"
{{- end }}
{{- if .EncryptedFields }}
	"github.com/hansjlachmann/openerp/backend/foundation/secrets"
{{- end }}
{{- if .UsesSIFT }}
	"github.com/hansjlachmann/openerp/backend/foundation/sift"
{{- end }}
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

{{- range .FlowFilterFields }}
	// FlowFilter {{ .Name }} ({{ .Type }}): the filter expression FlowFields apply — not stored
	{{ upperFirst .Name }} string
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
	descending    bool // sort order of the current key (SetAscending)

	// Free-text search (SetSearch): rows where any of the columns contains the text
	searchColumns []string
	searchText    string

	// Pagination window for FindSet/FindSetBuffered (0 limit = all rows)
	limit  int
	offset int

	// Buffered recordset for bidirectional navigation (BC/NAV style)
	bufferedRecords []*{{ .BaseStructName }}
	currentBufferPos int

	// Trigger function references (set by wrapper struct via SetTriggers)
	onInsertFn func() error
	onModifyFn func() error
	onDeleteFn func(database.Executor, string) error

	// Wrapper struct (set via SetSelf) so ValidateField can dispatch to
	// OnValidate_* overrides defined on the wrapper (Go has no virtual methods)
	self interface{}

	// Error returned by the OnInsert/OnModify/OnDelete trigger of the last
	// Insert/Modify/Delete call (nil if the trigger passed or did not run)
	triggerErr error
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

// TriggerError returns the error from the OnInsert/OnModify/OnDelete trigger that made
// the last Insert/Modify/Delete fail, or nil if it failed for another reason (database)
// or succeeded. Trigger errors are business-rule messages meant for the user.
func (t *{{ .BaseStructName }}) TriggerError() error {
	return t.triggerErr
}

// SetSelf registers the wrapper struct (called by wrapper InitWithDBType) so that
// ValidateField dispatches to OnValidate_* overrides defined on the wrapper.
func (t *{{ .BaseStructName }}) SetSelf(self interface{}) {
	t.self = self
}

// GetDB returns the database executor (for wrapper access)
func (t *{{ .BaseStructName }}) GetDB() database.Executor {
	return t.db
}

// GetCompany returns the company name (for wrapper access)
func (t *{{ .BaseStructName }}) GetCompany() string {
	return t.company
}

// GetDBType returns the database type (for wrapper access)
func (t *{{ .BaseStructName }}) GetDBType() database.DBType {
	return t.dbType
}

// IsSetupTable reports whether this is a BC-style singleton setup table
// (a single record identified by a blank primary key).
func (t *{{ .BaseStructName }}) IsSetupTable() bool {
	return {{ .Table.SetupTable }}
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

	// Indexes (BC/NAV Keys) and SIFT totals
	return t.SyncKeys(db, company, dbType)
}

// SyncKeys brings an existing table's keys up to date (table sync at startup): creates
// missing indexes and builds, rebuilds or drops the SIFT totals of keys with
// sum_index_fields (sift.Sync; unchanged keys cost one query).
func (t *{{ .BaseStructName }}) SyncKeys(db database.Executor, company string, dbType database.DBType) error {
{{- if .Table.Global }}
	tableName := {{ .StructName }}TableName
	siftCompany := ""
{{- else }}
	tableName := fmt.Sprintf("%s$%s", company, {{ .StructName }}TableName)
	siftCompany := company
{{- end }}
{{- range .Table.Keys }}
	// Index "company$Table$key"; a global table's indexes are shared by all companies:
	// "Table$key" (a company prefix created a copy of every index per company)
	if _, err := db.Exec(fmt.Sprintf(` + "`CREATE INDEX IF NOT EXISTS \"%s\" ON \"%s\" ({{ join .Fields \", \" }})`" + `,
{{- if $.Table.Global }}
		"{{ $.Table.Name }}${{ .Name }}", tableName)); err != nil {
{{- else }}
		fmt.Sprintf("%s${{ $.Table.Name }}${{ .Name }}", company), tableName)); err != nil {
{{- end }}
		return fmt.Errorf("failed to create index {{ .Name }}: %w", err)
	}
{{- end }}
{{- if not .Table.Keys }}
	_ = tableName // no keys: nothing to index
{{- end }}
	var keys []sift.Key
	for _, spec := range {{ lowerFirst .BaseStructName }}SIFTSpecs() {
		keys = append(keys, sift.BuildKey(dbType, siftCompany, {{ .StructName }}TableName, tableName, spec))
	}
	return sift.Sync(db, dbType, siftCompany, {{ .StructName }}TableName, keys)
}

// {{ lowerFirst .BaseStructName }}SIFTSpecs are the table's keys with sum_index_fields
func {{ lowerFirst .BaseStructName }}SIFTSpecs() []sift.KeySpec {
	return []sift.KeySpec{
{{- range .SIFTKeys }}
		{
			Name: "{{ .Name }}",
			Fields: []sift.Column{ {{- range .Fields }}{Name: "{{ .Name }}", Kind: sift.{{ .Kind }}}, {{ end -}} },
			Sums: []sift.Column{ {{- range .Sums }}{Name: "{{ .Name }}", Kind: sift.{{ .Kind }}}, {{ end -}} },
		},
{{- end }}
	}
}

// VerifySIFT compares the table's SIFT totals with its entries (Verify SIFT codeunit); with
// repair it rebuilds the keys whose totals differ.
func (t *{{ .BaseStructName }}) VerifySIFT(repair bool) ([]sift.VerifyResult, error) {
{{- if .Table.Global }}
	tableName, siftCompany := {{ .StructName }}TableName, ""
{{- else }}
	tableName, siftCompany := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName), t.company
{{- end }}
	var results []sift.VerifyResult
	for _, spec := range {{ lowerFirst .BaseStructName }}SIFTSpecs() {
		n, err := sift.VerifyKey(t.db, t.dbType, siftCompany, {{ .StructName }}TableName, tableName, spec)
		if err != nil {
			return results, fmt.Errorf("%s: %w", spec.Name, err)
		}
		result := sift.VerifyResult{Table: {{ .StructName }}TableName, Key: spec.Name, Differences: n}
		if n > 0 && repair {
			if err := sift.RebuildKey(t.db, t.dbType, siftCompany, {{ .StructName }}TableName, tableName, spec); err != nil {
				return results, fmt.Errorf("%s: %w", spec.Name, err)
			}
			result.Rebuilt = true
		}
		results = append(results, result)
	}
	return results, nil
}

// ========================================
// BC/NAV-style Record Methods
// ========================================

// InitWithDBType initializes a new {{ .StructName }} record with database context and type.
// db can be *sql.DB or *sql.Tx. There is no Init without the type: the type decides the SQL
// placeholders, and a default (SQLite) silently broke code on PostgreSQL.
func (t *{{ .BaseStructName }}) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.db = db
	t.company = company
	t.dbType = dbType
	t.oldValues = nil // Fresh record, no old values
	t.applyDefaults()
}

// InitRecord initializes a new, not yet inserted record (BC/NAV OnNewRecord).
// The base implementation applies the YAML default values; wrappers can override
// it to supply further defaults (call the base implementation first).
func (t *{{ .BaseStructName }}) InitRecord() {
	t.applyDefaults()
}

// applyDefaults assigns the YAML default values and auto timestamps
func (t *{{ .BaseStructName }}) applyDefaults() {
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

// OldValue returns a field's value as last read from or written to the database (BC/NAV
// xRec), e.g. the old key in OnRename; nil when the record was not loaded.
func (t *{{ .BaseStructName }}) OldValue(fieldName string) interface{} {
	return t.oldValues[fieldName]
}

// IsGlobal reports whether the table is global (no company prefix, e.g. User, Company)
func (t *{{ .BaseStructName }}) IsGlobal() bool {
	return {{ .Table.Global }}
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
	t.triggerErr = nil
	if runTrigger && t.onInsertFn != nil {
		if err := t.onInsertFn(); err != nil {
			fmt.Printf("Error: OnInsert trigger failed: %v\n", err)
			t.triggerErr = err
			return false
		}
	}
{{- if .EncryptedFields }}
	if err := t.encryptSecrets(); err != nil {
		fmt.Printf("Error: %v\n", err)
		t.triggerErr = err
		return false
	}
{{- end }}

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

// InsertAll inserts records in bulk (demo data, imports): multi-row INSERT statements, one
// round trip per batch instead of one per record. With runTrigger every record's OnInsert
// trigger runs first, as Insert(true) would; the first failure stops before anything is
// written. Records are written with the receiver's database and company. Errors are
// *tables.BatchInsertError (Index = the record in records). Use it in a transaction: a
// failed batch leaves the earlier batches written.
func (t *{{ .BaseStructName }}) InsertAll(records []*{{ .BaseStructName }}, runTrigger bool) error {
	rows := make([][]interface{}, 0, len(records))
	for i, r := range records {
		r.triggerErr = nil
		if runTrigger && r.onInsertFn != nil {
			if err := r.onInsertFn(); err != nil {
				r.triggerErr = err
				return &tables.BatchInsertError{Index: i, Trigger: true, Err: err}
			}
		}
{{- if .EncryptedFields }}
		if err := r.encryptSecrets(); err != nil {
			return &tables.BatchInsertError{Index: i, Trigger: true, Err: err}
		}
{{- end }}
		rows = append(rows, []interface{}{
{{- range .Table.Fields }}
{{- if not .FlowField }}
			r.{{ upperFirst .Name }},
{{- end }}
{{- end }}
		})
	}
{{- if .Table.Global }}
	tableName := {{ .StructName }}TableName
{{- else }}
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
{{- end }}
	columns := []string{ {{- range $i, $f := .Table.Fields }}{{ if not $f.FlowField }}"{{ $f.DBName }}"{{ if not (isLastDBField $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end -}} }
	return tables.InsertRows(t.db, t.dbType, tableName, columns, rows)
}

{{ if .EncryptedFields -}}
// ========================================
// Encrypted fields (encrypted: true)
// ========================================

// encryptSecrets stores the encrypted fields encrypted (secrets.Encrypt) when they hold a
// new value as typed. Without an encryption key (neither OPENERP_ENCRYPTION_KEY nor
// JWT_SECRET set — development) the value stays as typed; the backend warns at startup.
func (t *{{ .BaseStructName }}) encryptSecrets() error {
{{- range .EncryptedFields }}
	if v := t.{{ upperFirst .Name }}.String(); v != "" && !secrets.IsEncrypted(v) {
		// Encrypted, the value must still fit the {{ .Length }}-character column
		if len(v) > {{ maxEncryptedPlain .Length }} {
			return apperrors.FieldTooLong({{ $.StructName }}TableName, "{{ .Name }}", {{ maxEncryptedPlain .Length }})
		}
		stored, err := secrets.Encrypt(v)
		if err != nil && !errors.Is(err, secrets.ErrNoKey) {
			return err
		}
		if err == nil {
			t.{{ upperFirst .Name }} = types.NewText(stored)
		}
	}
{{- end }}
	return nil
}
{{- range .EncryptedFields }}

// Plain{{ upperFirst .Name }} returns {{ .Name }} decrypted, for the server code that uses the
// secret (never send it to the client). secrets.ErrDecrypt means it was encrypted with
// another key: it has to be entered again.
func (t *{{ $.BaseStructName }}) Plain{{ upperFirst .Name }}() (string, error) {
	return secrets.Decrypt(t.{{ upperFirst .Name }}.String())
}
{{- end }}

// EncryptStoredSecrets encrypts the values of encrypted fields that are stored as typed
// (saved before the field was encrypted, or without a key) in the table of the receiver's
// company. Run at startup; does nothing without a key. Returns how many it encrypted.
func (t *{{ .BaseStructName }}) EncryptStoredSecrets() (int, error) {
	if secrets.Source() == "" {
		return 0, nil
	}
{{- if .Table.Global }}
	tableName := {{ .StructName }}TableName
{{- else }}
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
{{- end }}
	done := 0
{{- range .EncryptedFields }}
	{
		rows, err := t.db.Query(fmt.Sprintf(` + "`SELECT {{ range $.Table.Fields }}{{ if .PrimaryKey }}{{ .DBName }}, {{ end }}{{ end }}{{ .DBName }} FROM \"%s\" WHERE {{ .DBName }} <> ''`" + `, tableName))
		if err != nil {
			return done, err
		}
		type plainRow struct {
			key   []interface{}
			value string
		}
		var plain []plainRow
		for rows.Next() {
			key := make([]interface{}, {{ pkCount $.Table.Fields }})
			dest := make([]interface{}, 0, len(key)+1)
			for i := range key {
				dest = append(dest, &key[i])
			}
			var value string
			dest = append(dest, &value)
			if err := rows.Scan(dest...); err != nil {
				_ = rows.Close()
				return done, err
			}
			if !secrets.IsEncrypted(value) {
				plain = append(plain, plainRow{key, value})
			}
		}
		if err := rows.Close(); err != nil {
			return done, err
		}
		for _, r := range plain {
			if len(r.value) > {{ maxEncryptedPlain .Length }} {
				return done, apperrors.FieldTooLong({{ $.StructName }}TableName, "{{ .Name }}", {{ maxEncryptedPlain .Length }})
			}
			stored, err := secrets.Encrypt(r.value)
			if err != nil {
				return done, err
			}
			sqlStr := t.convertPlaceholders(fmt.Sprintf(` + "`UPDATE \"%s\" SET {{ .DBName }} = ? WHERE {{ range $i, $f := $.Table.Fields }}{{ if $f.PrimaryKey }}{{ $f.DBName }} = ?{{ if not (isLastPK $i $.Table.Fields) }} AND {{ end }}{{ end }}{{ end }}`" + `, tableName), 1+len(r.key))
			if _, err := t.db.Exec(sqlStr, append([]interface{}{stored}, r.key...)...); err != nil {
				return done, err
			}
			done++
		}
	}
{{- end }}
	return done, nil
}

{{ end -}}
// Modify updates the record in the database
func (t *{{ .BaseStructName }}) Modify(runTrigger bool) bool {
	// Call OnModify trigger if requested (via function reference set by wrapper)
	t.triggerErr = nil
	if runTrigger && t.onModifyFn != nil {
		if err := t.onModifyFn(); err != nil {
			fmt.Printf("Error: OnModify trigger failed: %v\n", err)
			t.triggerErr = err
			return false
		}
	}
	// A changed primary key is a rename (BC/NAV): run the wrapper's OnRename trigger
	// before it is written; xRec values are available through OldValue
	keyChanged := false
	if t.oldValues != nil {
{{- range .Table.Fields }}
{{- if .PrimaryKey }}
		if t.hasFieldChanged("{{ .DBName }}") {
			keyChanged = true
		}
{{- end }}
{{- end }}
	}
	if runTrigger && keyChanged {
		if r, ok := t.self.(interface{ OnRename() error }); ok {
			if err := r.OnRename(); err != nil {
				fmt.Printf("Error: OnRename trigger failed: %v\n", err)
				t.triggerErr = err
				return false
			}
		}
	}
{{- if .EncryptedFields }}
	if err := t.encryptSecrets(); err != nil {
		fmt.Printf("Error: %v\n", err)
		t.triggerErr = err
		return false
	}
{{- end }}

{{- if .Table.Global }}
	tableName := {{ .StructName }}TableName
{{- else }}
	tableName := fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName)
{{- end }}

	// Build dynamic SQL based on field tracking
	var setClauses []string
	var values []interface{}

	// If we have old values (loaded from Get), only update changed fields.
	// Changed primary key fields are renamed in place (BC/NAV Rename): they are SET to the
	// new value while the WHERE clause matches the old one.
	if t.oldValues != nil {
{{- range .Table.Fields }}
{{- if not .FlowField }}
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

	// Add WHERE clause value (primary key as loaded, so a renamed key still matches)
{{- range .Table.Fields }}
{{- if .PrimaryKey }}
	if old, ok := t.oldValues["{{ .DBName }}"]; ok {
		values = append(values, old)
	} else {
		values = append(values, t.{{ upperFirst .Name }})
	}
{{- end }}
{{- end }}

{{- if .References }}
	// Renamed key fields that other tables refer to (BC/NAV Rename): carried over below
	type renamedKey struct {
		field    string
		old, new interface{}
	}
	var renamed []renamedKey
	if t.oldValues != nil {
{{- range .Table.Fields }}
{{- if and .PrimaryKey (index $.References .DBName) }}
		if t.hasFieldChanged("{{ .DBName }}") {
			renamed = append(renamed, renamedKey{"{{ .DBName }}", t.oldValues["{{ .DBName }}"], t.{{ upperFirst .Name }}})
		}
{{- end }}
{{- end }}
	}
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
{{- if .References }}
	for _, r := range renamed {
		if err := t.renameReferences(r.field, r.old, r.new); err != nil {
			fmt.Printf("Error: Failed to rename references to {{ .Table.Name }}: %v\n", err)
			t.triggerErr = err
			return false
		}
	}
{{- end }}
	// The stored record now matches the current values (including a renamed key)
	if t.oldValues != nil {
		t.StoreOldValues()
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
{{- if not .FlowField }}
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

// SetDB changes the database executor of the record without touching its values — e.g. to
// run a Modify (with its rename cascade) inside a transaction.
func (t *{{ .BaseStructName }}) SetDB(db database.Executor) {
	t.db = db
}
{{- if .References }}

// renameReferences carries a renamed key over to every field of another table that refers
// to it (BC/NAV Rename). References from company tables are updated in this company — in
// every company when this table is global. Runs on the record's executor, so inside the
// caller's transaction.
func (t *{{ .BaseStructName }}) renameReferences(field string, oldValue, newValue interface{}) error {
	type reference struct {
		table  string
		global bool
		column string
	}
	references := map[string][]reference{
{{- range $field, $refs := .References }}
		"{{ $field }}": { {{- range $refs }}{"{{ .Table }}", {{ .Global }}, "{{ .Column }}"}, {{ end -}} },
{{- end }}
	}
{{- if .Table.Global }}
	var companies []string
	rows, err := t.db.Query(` + "`SELECT name FROM \"Company\"`" + `)
	if err != nil {
		return err
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		companies = append(companies, name)
	}
	rows.Close()
{{- else }}
	companies := []string{t.company}
{{- end }}
	for _, ref := range references[field] {
		tables := []string{ref.table}
		if !ref.global {
			tables = tables[:0]
			for _, c := range companies {
				tables = append(tables, c+"$"+ref.table)
			}
		}
		for _, tbl := range tables {
			query := t.convertPlaceholders(fmt.Sprintf(` + "`UPDATE \"%s\" SET %s = ? WHERE %s = ?`" + `, tbl, ref.column, ref.column), 2)
			if _, err := t.db.Exec(query, newValue, oldValue); err != nil {
				return fmt.Errorf("%s.%s: %w", tbl, ref.column, err)
			}
		}
	}
	return nil
}
{{- end }}

// Delete removes the record from the database
func (t *{{ .BaseStructName }}) Delete(runTrigger bool) bool {
	// Call OnDelete trigger if requested (via function reference set by wrapper)
	t.triggerErr = nil
	if runTrigger && t.onDeleteFn != nil {
		if err := t.onDeleteFn(t.db, t.company); err != nil {
			fmt.Printf("Error: OnDelete trigger failed: %v\n", err)
			t.triggerErr = err
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

// CalcFieldsForRecords calculates FlowFields for many records at once (list pages):
// one grouped query per FlowField instead of one query per record. records are
// ToMap() results; each gets its FlowField values under the field name. With no
// field names, all FlowFields are calculated.
func (t *{{ .BaseStructName }}) CalcFieldsForRecords(records []map[string]interface{}, fieldNames ...string) {
	if len(records) == 0 {
		return
	}
	if len(fieldNames) == 0 {
		fieldNames = t.GetFlowFields()
	}
	for _, fieldName := range fieldNames {
		switch fieldName {
		{{- range .Table.Fields }}
		{{- if .FlowField }}
		case "{{ .DBName }}":
			t.calcForRecords_{{ .Name }}(records)
		{{- end }}
		{{- end }}
		}
	}
}

{{- range .Table.Fields }}
{{- if .FlowField }}

// calcForRecords_{{ .Name }} calculates the {{ .Name }} FlowField for a set of records
// CalcFormula: {{ .CalcFormula }}({{ .SourceTable }}.{{ .SourceField }})
func (t *{{ $.BaseStructName }}) calcForRecords_{{ .Name }}(records []map[string]interface{}) {
{{- if and .BatchKeyField (or (eq .CalcFormula "Sum") (eq .CalcFormula "Count")) }}
{{- template "flowSource" . }}

	// Distinct key values of the records
	seen := make(map[string]bool, len(records))
	var keys []interface{}
	for _, rec := range records {
		k := fmt.Sprint(rec["{{ .BatchKeyValue }}"])
		if !seen[k] {
			seen[k] = true
			keys = append(keys, rec["{{ .BatchKeyValue }}"])
		}
	}

	// One grouped query for the keys (IN list). With more keys than chunkSize (a long
	// list) one grouped query over the whole source table instead: much cheaper than
	// many IN queries, and it keeps the bind parameters bounded.
	{{- if eq .CalcFormula "Count" }}
	values := make(map[string]int, len(keys))
	{{- else }}
	values := make(map[string]{{ .Type }}, len(keys))
	{{- end }}
	const chunkSize = 500
	chunks := [][]interface{}{nil} // nil chunk: no IN filter, all keys
	if len(keys) <= chunkSize {
		chunks = [][]interface{}{keys}
	}
	for _, chunk := range chunks {

		var whereClauses []string
		var args []interface{}
		{{- range .FlowFilters }}
		{{- if eq .Type "const" }}
		whereClauses = append(whereClauses, "{{ .Field }} = ?")
		args = append(args, {{ .Value }})
		{{- end }}
		{{- end }}
		{{- template "flowFilterWhere" . }}
		if chunk != nil {
			whereClauses = append(whereClauses, "{{ .BatchKeyField }} IN ("+strings.TrimSuffix(strings.Repeat("?, ", len(chunk)), ", ")+")")
			args = append(args, chunk...)
		}
		whereClause := "1=1"
		if len(whereClauses) > 0 {
			whereClause = strings.Join(whereClauses, " AND ")
		}

		{{- if eq .CalcFormula "Count" }}
		agg := "COUNT(*)"
		if useSIFT {
			agg = "COALESCE(SUM(cnt), 0)" // the totals hold the entry count per key value
		}
		query := fmt.Sprintf(` + "`SELECT {{ .BatchKeyField }}, %s FROM \"%s\" WHERE %s GROUP BY {{ .BatchKeyField }}`" + `, agg, tableName, whereClause)
		{{- else }}
		query := fmt.Sprintf(` + "`SELECT {{ .BatchKeyField }}, COALESCE(SUM({{ .SourceField }}), 0) FROM \"%s\" WHERE %s GROUP BY {{ .BatchKeyField }}`" + `, tableName, whereClause)
		{{- end }}
		query = t.convertPlaceholders(query, len(args))

		rows, err := t.db.Query(query, args...)
		if err != nil {
			fmt.Printf("Error: Failed to calculate {{ .Name }}: %v\n", err)
			return
		}
		for rows.Next() {
			var key string
			{{- if eq .CalcFormula "Count" }}
			var count int
			if err := rows.Scan(&key, &count); err == nil {
				values[key] = count
			}
			{{- else }}
			var sumStr string
			if err := rows.Scan(&key, &sumStr); err == nil {
				values[key], _ = types.NewDecimalFromString(sumStr)
			}
			{{- end }}
		}
		_ = rows.Close()
	}

	// Records without matching entries get zero
	for _, rec := range records {
		{{- if eq .CalcFormula "Count" }}
		rec["{{ .DBName }}"] = values[fmt.Sprint(rec["{{ .BatchKeyValue }}"])]
		{{- else }}
		v, ok := values[fmt.Sprint(rec["{{ .BatchKeyValue }}"])]
		if !ok {
			v = types.ZeroDecimal()
		}
		rec["{{ .DBName }}"] = v.String()
		{{- end }}
	}
{{- else }}
	// No single key field to group by: calculate per record
	for _, rec := range records {
		r := &{{ $.BaseStructName }}{db: t.db, company: t.company, dbType: t.dbType}
		r.FromMap(rec)
		r.calcFlowField_{{ .Name }}()
		{{- if eq .Type "types.Decimal" }}
		rec["{{ .DBName }}"] = r.{{ upperFirst .Name }}.String()
		{{- else }}
		rec["{{ .DBName }}"] = r.{{ upperFirst .Name }}
		{{- end }}
	}
{{- end }}
}
{{- end }}
{{- end }}

// Helper methods for FlowField calculations
{{- range .Table.Fields }}
{{- if and .FlowField (eq .CalcFormula "Sum") }}

func (t *{{ $.BaseStructName }}) calcSum{{ upperFirst .SourceTable }}{{ upperFirst .SourceField }}() {{ .Type }} {
{{- template "flowSource" . }}

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
	{{- template "flowFilterWhere" . }}

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
{{- template "flowSource" . }}

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
	{{- template "flowFilterWhere" . }}

	whereClause := "1=1"
	if len(whereClauses) > 0 {
		whereClause = strings.Join(whereClauses, " AND ")
	}

	agg := "COUNT(*)"
	if useSIFT {
		agg = "COALESCE(SUM(cnt), 0)" // the totals hold the entry count per key value
	}
	query := fmt.Sprintf(` + "`SELECT %s FROM \"%s\" WHERE %s`" + `, agg, tableName, whereClause)

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

// CalcFieldsForRecords is a no-op for tables without FlowFields
func (t *{{ .BaseStructName }}) CalcFieldsForRecords(records []map[string]interface{}, fieldNames ...string) {
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
	invalidField bool          // Filter on an unknown field: matches no rows (fail closed)
{{- if .HasFilterableFlowField }}
	flowField    string        // Filter on a FlowField (filterExpr applied to its subquery)
{{- end }}
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

	column, ok := t.columnName(fieldName)
	if !ok {
		// Unknown field: fail closed (no rows) rather than drop the filter or put
		// the name into SQL
		fmt.Printf("Error: SetRange on unknown field %q of {{ .Table.Name }}\n", fieldName)
		t.filters[fieldName] = &{{ lowerFirst .BaseStructName }}FilterCondition{invalidField: true}
		return
	}

	t.filters[column] = &{{ lowerFirst .BaseStructName }}FilterCondition{
		fieldName: column,
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
{{- if .HasFilterableFlowField }}
	if _, ok := t.FlowFieldFilterKind(fieldName); ok {
		// A FlowField is not a column: the filter applies to its subquery (flowFieldExpr)
		name := strings.ToLower(fieldName)
		t.filters["flowfield:"+name] = &{{ lowerFirst .BaseStructName }}FilterCondition{
			flowField:    name,
			filterExpr:   filterExpr,
			isExpression: true,
		}
		return
	}
{{- end }}
	column, ok := t.columnName(fieldName)
	if !ok {
		// Unknown field: fail closed (no rows) rather than drop the filter or put
		// the name into SQL
		fmt.Printf("Error: SetFilter on unknown field %q of {{ .Table.Name }}\n", fieldName)
		t.filters[fieldName] = &{{ lowerFirst .BaseStructName }}FilterCondition{invalidField: true}
		return
	}
	t.filters[column] = &{{ lowerFirst .BaseStructName }}FilterCondition{
		fieldName:    column,
		filterExpr:   filterExpr,
		isExpression: true,
	}
}

{{ if .HasFilterableFlowField -}}
// FlowFieldFilterKind reports whether a list can be filtered on FlowField field (Sum and
// Count FlowFields, BC: SETFILTER on a FlowField) and the kind of its values.
func (t *{{ .BaseStructName }}) FlowFieldFilterKind(field string) (flowfilter.Kind, bool) {
	switch strings.ToLower(field) {
{{- range .Table.Fields }}
{{- if and .FlowField (or (eq .CalcFormula "Sum") (eq .CalcFormula "Count")) }}
	case "{{ .DBName }}":
{{- if eq .CalcFormula "Count" }}
		return flowfilter.KindInt, true
{{- else }}
		return flowfilter.KindDecimal, true
{{- end }}
{{- end }}
{{- end }}
	}
	return "", false
}

// flowFieldExpr is the SQL expression computing FlowField field for each row of a query
// on this table: a correlated subquery reading the source like CalcFieldsForRecords (the
// SIFT totals of a covering key, else the entries), with the FlowFilters set on t.
func (t *{{ .BaseStructName }}) flowFieldExpr(field string) (string, []interface{}) {
{{- if .Table.Global }}
	outer := "\"" + {{ .StructName }}TableName + "\""
{{- else }}
	outer := "\"" + fmt.Sprintf("%s$%s", t.company, {{ .StructName }}TableName) + "\""
{{- end }}
	_ = outer
	switch strings.ToLower(field) {
{{- range .Table.Fields }}
{{- if and .FlowField (or (eq .CalcFormula "Sum") (eq .CalcFormula "Count")) }}
	case "{{ .DBName }}":
{{- template "flowSource" . }}
		var whereClauses []string
		var args []interface{}
		{{- range .FlowFilters }}
		{{- if eq .Type "const" }}
		whereClauses = append(whereClauses, "{{ .Field }} = ?")
		args = append(args, {{ .Value }})
		{{- else if eq .Type "field" }}
		whereClauses = append(whereClauses, "{{ .Field }} = "+outer+".{{ .ValueColumn }}")
		{{- end }}
		{{- end }}
		{{- template "flowFilterWhere" . }}
		where := "1=1"
		if len(whereClauses) > 0 {
			where = strings.Join(whereClauses, " AND ")
		}
		{{- if eq .CalcFormula "Count" }}
		agg := "COUNT(*)"
		if useSIFT {
			agg = "COALESCE(SUM(cnt), 0)"
		}
		{{- else }}
		agg := "COALESCE(SUM({{ .SourceField }}), 0)"
		{{- end }}
		return "(SELECT " + agg + " FROM \"" + tableName + "\" WHERE " + where + ")", args
{{- end }}
{{- end }}
	}
	return "NULL", nil
}

{{ end -}}

// SetCurrentKey sets the sort order for queries (BC/NAV style)
// Unknown fields are ignored (the primary key order is used if none remain).
// Example: customer.SetCurrentKey("City", "Name")
func (t *{{ .BaseStructName }}) SetCurrentKey(fields ...string) {
	t.orderByFields = nil
	for _, field := range fields {
		column, ok := t.columnName(field)
		if !ok {
			fmt.Printf("Error: SetCurrentKey on unknown field %q of {{ .Table.Name }}\n", field)
			continue
		}
		t.orderByFields = append(t.orderByFields, column)
	}
}

// SetAscending sets the sort direction of the current key (BC/NAV Ascending).
// The default is ascending.
func (t *{{ .BaseStructName }}) SetAscending(ascending bool) {
	t.descending = !ascending
}

// SetSearch limits the records to those where any of the given fields contains text,
// case-insensitive (the list page search box). Unknown fields are ignored; an empty
// text clears the search.
func (t *{{ .BaseStructName }}) SetSearch(fields []string, text string) {
	t.searchColumns = nil
	t.searchText = strings.ToLower(strings.TrimSpace(text))
	if t.searchText == "" {
		return
	}
	for _, field := range fields {
		if column, ok := t.columnName(field); ok {
			t.searchColumns = append(t.searchColumns, column)
		}
	}
}

// HasColumn reports whether fieldName (case-insensitive) is a stored column of this
// table. Only such names may be used in filters and sort keys.
func (t *{{ .BaseStructName }}) HasColumn(fieldName string) bool {
	_, ok := t.columnName(fieldName)
	return ok
}

// columnName maps a field name (case-insensitive) to its database column. Field names
// end up in SQL text (WHERE / ORDER BY / SET), so only names from this allowlist are used.
func (t *{{ .BaseStructName }}) columnName(fieldName string) (string, bool) {
	switch strings.ToLower(fieldName) {
{{- range .Table.Fields }}
{{- if not .FlowField }}
	case strings.ToLower("{{ .DBName }}"):
		return "{{ .DBName }}", true
{{- end }}
{{- end }}
	}
	return "", false
}

// Reset clears all filters (BC/NAV style)
func (t *{{ .BaseStructName }}) Reset() {
	t.filters = nil
	t.oldValues = nil
	t.orderByFields = nil
	t.descending = false
	t.searchColumns = nil
	t.searchText = ""
	if t.currentRows != nil {
		t.currentRows.Close()
		t.currentRows = nil
	}
}

// buildWhereClause builds WHERE clause from current filters and search
func (t *{{ .BaseStructName }}) buildWhereClause() (string, []interface{}) {
	if len(t.filters) == 0 && t.searchText == "" {
		return "1=1", nil
	}

	var conditions []string
	var args []interface{}

	if t.searchText != "" {
		if len(t.searchColumns) == 0 {
			conditions = append(conditions, "1=0") // searching, but in no valid column
		} else {
			// LIKE wildcards in the text are matched literally
			pattern := "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(t.searchText) + "%"
			var ors []string
			for _, column := range t.searchColumns {
				ors = append(ors, "LOWER(CAST("+column+" AS TEXT)) LIKE ? ESCAPE '\\'")
				args = append(args, pattern)
			}
			conditions = append(conditions, "("+strings.Join(ors, " OR ")+")")
		}
	}

	for _, filter := range t.filters {
		if filter.invalidField {
			conditions = append(conditions, "1=0")
{{- if .HasFilterableFlowField }}
		} else if filter.flowField != "" {
			// BC filter syntax on the FlowField's value; an invalid expression matches no rows
			// (the API validates it first and reports it)
			kind, _ := t.FlowFieldFilterKind(filter.flowField)
			expr, exprArgs := t.flowFieldExpr(filter.flowField)
			clause, clauseArgs, err := flowfilter.ClauseFor(expr, exprArgs, kind, filter.filterExpr)
			if err != nil {
				conditions = append(conditions, "1=0")
			} else if clause != "" {
				conditions = append(conditions, clause)
				args = append(args, clauseArgs...)
			}
{{- end }}
		} else if filter.isExpression {
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

// getOrderByClause builds ORDER BY clause from current key and direction. The primary
// key columns always follow the current key, so the order is unique and paging
// (SetPage) never repeats or skips rows when the key has duplicate values.
func (t *{{ .BaseStructName }}) getOrderByClause() string {
	pk := []string{ {{- range $i, $f := .Table.Fields }}{{ if $f.PrimaryKey }}"{{ $f.DBName }}", {{ end }}{{ end -}} }
	columns := append([]string{}, t.orderByFields...)
	for _, column := range pk {
		if !slices.Contains(columns, column) {
			columns = append(columns, column)
		}
	}
	if t.descending {
		for i := range columns {
			columns[i] += " DESC"
		}
	}
	return strings.Join(columns, ", ")
}

// SetPage sets a pagination window for FindSet/FindSetBuffered: return at most
// limit rows, skipping the first offset rows. A limit of 0 disables pagination
// (all matching rows are returned). Negative values are treated as 0.
func (t *{{ .BaseStructName }}) SetPage(limit, offset int) {
	if limit < 0 {
		limit = 0
	}
	if offset < 0 {
		offset = 0
	}
	t.limit = limit
	t.offset = offset
}

// getLimitClause builds the LIMIT/OFFSET clause from the pagination window.
// Returns an empty string when no limit is set. LIMIT/OFFSET take integer
// literals (not placeholders), which both SQLite and PostgreSQL accept.
func (t *{{ .BaseStructName }}) getLimitClause() string {
	if t.limit <= 0 {
		return ""
	}
	if t.offset > 0 {
		return fmt.Sprintf(" LIMIT %d OFFSET %d", t.limit, t.offset)
	}
	return fmt.Sprintf(" LIMIT %d", t.limit)
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
	query := fmt.Sprintf(` + "`SELECT {{ range $i, $f := .Table.Fields }}{{ if not $f.FlowField }}{{ $f.DBName }}{{ if not (isLastDBField $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }} FROM \"%s\" WHERE %s ORDER BY %s%s`" + `, tableName, where, orderBy, t.getLimitClause())

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
	query := fmt.Sprintf(` + "`SELECT {{ range $i, $f := .Table.Fields }}{{ if not $f.FlowField }}{{ $f.DBName }}{{ if not (isLastDBField $i $.Table.Fields) }}, {{ end }}{{ end }}{{ end }} FROM \"%s\" WHERE %s ORDER BY %s%s`" + `, tableName, where, orderBy, t.getLimitClause())

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
	column, ok := t.columnName(fieldName)
	if !ok {
		fmt.Printf("Error: ModifyAll on unknown field %q of {{ .Table.Name }}\n", fieldName)
		return 0
	}
	where, args := t.buildWhereClause()

	// Build UPDATE SQL
	updateSQL := fmt.Sprintf(` + "`UPDATE \"%s\" SET %s = ? WHERE %s`" + `, tableName, column, where)

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
			invalidField: filter.invalidField,
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
		if filter.invalidField {
			parts = append(parts, "<invalid field>")
		} else if filter.isExpression {
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
		// Call OnValidate trigger (the wrapper's override if it defines one)
		if w, ok := t.self.(interface{ OnValidate_{{ upperFirst .Name }}() error }); ok {
			return w.OnValidate_{{ upperFirst .Name }}()
		}
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
{{- if .Sensitive }}
			Sensitive:  true,
{{- end }}
{{- if .Masked }}
			Masked:     true,
{{- end }}
		},
{{- end }}
	}
}

// SetFlowFilter sets a FlowFilter field's filter expression (BC/NAV SETFILTER on a
// FlowFilter field); FlowFields applying it use it from then on. "" clears it.
func (t *{{ .BaseStructName }}) SetFlowFilter(field, expr string) error {
	switch field {
{{- range .FlowFilterFields }}
	case "{{ .DBName }}":
		if err := flowfilter.Validate(flowfilter.{{ flowFilterKind . }}, expr); err != nil {
			return err
		}
		t.{{ upperFirst .Name }} = expr
		return nil
{{- end }}
	}
	return fmt.Errorf("%q is not a FlowFilter field of {{ .Table.Name }}", field)
}

// GetFlowFilterFields returns the FlowFilter fields (name and flowfilter kind)
func (t *{{ .BaseStructName }}) GetFlowFilterFields() []tables.FlowFilterFieldInfo {
	return []tables.FlowFilterFieldInfo{
{{- range .FlowFilterFields }}
		{Name: "{{ .DBName }}", Kind: string(flowfilter.{{ flowFilterKind . }})},
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
{{- if .HasTimeField }}
	"time"

{{- end }}
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
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

// InitWithDBType initializes the record with database context and type and sets up
// triggers. The API creates tables via the tables.Table interface and calls this method,
// so the wiring must live here for triggers and OnValidate_* overrides to fire.
func (t *{{ .StructName }}) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.{{ .BaseStructName }}.InitWithDBType(db, company, dbType)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
	t.SetSelf(t)
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
	if err := ftables.CheckRequired(gtables.{{ $.StructName }}TableName, "{{ .Name }}", t.{{ upperFirst .Name }}.IsEmpty()); err != nil {
		return err
	}
	{{- else if eq .Type "string" }}
	if err := ftables.CheckRequired(gtables.{{ $.StructName }}TableName, "{{ .Name }}", t.{{ upperFirst .Name }} == ""); err != nil {
		return err
	}
	{{- end }}
{{- end }}
{{- if .Length }}
	if err := ftables.CheckMaxLength(gtables.{{ $.StructName }}TableName, "{{ .Name }}", string(t.{{ upperFirst .Name }}), {{ .Length }}); err != nil {
		return err
	}
{{- end }}
{{- if .Validation }}
	if err := ftables.CheckRange(gtables.{{ $.StructName }}TableName, "{{ .Name }}", t.{{ upperFirst .Name }}, {{ .Validation.Min }}, {{ .Validation.Max }}); err != nil {
		return err
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
