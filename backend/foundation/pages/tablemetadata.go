package pages

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// TableMetadata holds metadata about tables for page rendering
type TableMetadata struct {
	primaryKeys    map[string][]string          // table name -> primary key field names (supports composite keys)
	requiredFields map[string]map[string]bool   // table name -> field name -> required
	maskedFields   map[string]map[string]bool   // table name -> field name -> masked (write-only secret)
	fieldTypes     map[string]map[string]string // table name -> field name -> type (e.g., "bool", "code", "text")
	captionFields  map[string][]string          // table name -> data_caption_fields (the card title)
	lookupPages    map[string]int               // table name -> lookup_page_id (the lookup's "New" opens it)
	autoKeys       map[string]bool              // table name -> auto_key (OnInsert assigns a blank key)
	mu             sync.RWMutex
}

// tableDefYAML represents the structure of table YAML files
type tableDefYAML struct {
	Table struct {
		Name              string          `yaml:"name"`
		Fields            []tableFieldDef `yaml:"fields"`
		DataCaptionFields []string        `yaml:"data_caption_fields"`
		LookupPageID      int             `yaml:"lookup_page_id"`
		AutoKey           bool            `yaml:"auto_key"`
	} `yaml:"table"`
}

// tableFieldDef represents a field in the table definition
type tableFieldDef struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	PrimaryKey bool   `yaml:"primary_key"`
	Required   bool   `yaml:"required"`
	Masked     bool   `yaml:"masked"`
}

var (
	tableMetadata     *TableMetadata
	tableMetadataOnce sync.Once
)

// GetTableMetadata returns the singleton table metadata instance
func GetTableMetadata() *TableMetadata {
	tableMetadataOnce.Do(func() {
		tableMetadata = &TableMetadata{
			primaryKeys:    make(map[string][]string),
			requiredFields: make(map[string]map[string]bool),
			maskedFields:   make(map[string]map[string]bool),
			fieldTypes:     make(map[string]map[string]string),
			captionFields:  make(map[string][]string),
			lookupPages:    make(map[string]int),
			autoKeys:       make(map[string]bool),
		}
		if err := tableMetadata.Load(); err != nil {
			fmt.Printf("Warning: Failed to load table metadata: %v\n", err)
		}
	})
	return tableMetadata
}

// Load reads all table definitions and extracts primary key information
func (tm *TableMetadata) Load() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	rootPath, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	tablesPath := filepath.Join(rootPath, "backend", "definitions", "tables")

	if _, err := os.Stat(tablesPath); os.IsNotExist(err) {
		return fmt.Errorf("tables directory not found: %s", tablesPath)
	}

	files, err := filepath.Glob(filepath.Join(tablesPath, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to read tables directory: %w", err)
	}

	for _, file := range files {
		if err := tm.loadTableFile(file); err != nil {
			fmt.Printf("Warning: Failed to load table metadata from %s: %v\n", filepath.Base(file), err)
		}
	}

	fmt.Printf("✓ Loaded primary key metadata for %d table(s)\n", len(tm.primaryKeys))
	return nil
}

func (tm *TableMetadata) loadTableFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var tableDef tableDefYAML
	if err := yaml.Unmarshal(data, &tableDef); err != nil {
		return err
	}

	// Find all primary key fields, required fields, and field types
	var pkFields []string
	reqFields := make(map[string]bool)
	maskFields := make(map[string]bool)
	fTypes := make(map[string]string)
	for _, field := range tableDef.Table.Fields {
		if field.PrimaryKey {
			pkFields = append(pkFields, field.Name)
		}
		if field.Required {
			reqFields[field.Name] = true
		}
		if field.Masked {
			maskFields[field.Name] = true
		}
		if field.Type != "" {
			fTypes[field.Name] = field.Type
		}
	}
	if len(pkFields) > 0 {
		tm.primaryKeys[metadataKey(tableDef.Table.Name)] = pkFields
	}
	if len(reqFields) > 0 {
		tm.requiredFields[metadataKey(tableDef.Table.Name)] = reqFields
	}
	if len(fTypes) > 0 {
		tm.fieldTypes[metadataKey(tableDef.Table.Name)] = fTypes
	}
	if tableDef.Table.AutoKey {
		tm.autoKeys[metadataKey(tableDef.Table.Name)] = true
	}
	if len(tableDef.Table.DataCaptionFields) > 0 {
		if tm.captionFields == nil {
			tm.captionFields = make(map[string][]string)
		}
		tm.captionFields[metadataKey(tableDef.Table.Name)] = tableDef.Table.DataCaptionFields
		if tableDef.Table.LookupPageID != 0 {
			tm.lookupPages[metadataKey(tableDef.Table.Name)] = tableDef.Table.LookupPageID
		}
	}
	if len(maskFields) > 0 {
		if tm.maskedFields == nil {
			tm.maskedFields = make(map[string]map[string]bool)
		}
		tm.maskedFields[metadataKey(tableDef.Table.Name)] = maskFields
	}

	return nil
}

// GetPrimaryKeyField returns the first primary key field name for a table
func (tm *TableMetadata) GetPrimaryKeyField(tableName string) string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if fields := tm.primaryKeys[metadataKey(tableName)]; len(fields) > 0 {
		return fields[0]
	}
	return ""
}

// GetPrimaryKeyFields returns all primary key field names for a table (supports composite keys)
func (tm *TableMetadata) GetPrimaryKeyFields(tableName string) []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.primaryKeys[metadataKey(tableName)]
}

// GetFieldType returns the YAML type for a field (e.g., "bool", "code", "text")
func (tm *TableMetadata) GetFieldType(tableName, fieldName string) string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if types, ok := tm.fieldTypes[metadataKey(tableName)]; ok {
		return types[fieldName]
	}
	return ""
}

// IsFieldMasked returns whether a field of a table is masked (YAML masked: true)
func (tm *TableMetadata) IsFieldMasked(tableName, fieldName string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.maskedFields[metadataKey(tableName)][fieldName]
}

// GetDataCaptionFields returns a table's data_caption_fields (the fields shown in a card's
// title), nil without
func (tm *TableMetadata) GetDataCaptionFields(tableName string) []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.captionFields[metadataKey(tableName)]
}

// GetLookupPageID returns a table's lookup page (YAML lookup_page_id, classic LookupFormID), 0
// when the table names none
// IsAutoKey reports whether a table's OnInsert assigns the key when it is blank (auto_key:
// true — the next number of a number series); such records are inserted without a key
func (tm *TableMetadata) IsAutoKey(tableName string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.autoKeys[metadataKey(tableName)]
}

// GetLookupPageID returns a table's lookup_page_id (0 when it has none)
func (tm *TableMetadata) GetLookupPageID(tableName string) int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.lookupPages[metadataKey(tableName)]
}

// IsFieldRequired returns whether a field is required for a table
func (tm *TableMetadata) IsFieldRequired(tableName, fieldName string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if reqFields, ok := tm.requiredFields[metadataKey(tableName)]; ok {
		return reqFields[fieldName]
	}
	return false
}

// metadataKey normalizes a table name for the metadata maps: pages use registry names
// ("Customer_ledger_entry", "Payment_terms"), table YAML files display names ("Customer
// Ledger Entry", "Payment Terms"). Without this, such pages got no primary_key_fields, the
// frontend keyed their rows randomly and lost focus on every render.
func metadataKey(tableName string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(tableName), " ", "_"))
}
