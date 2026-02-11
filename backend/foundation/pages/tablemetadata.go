package pages

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// TableMetadata holds metadata about tables for page rendering
type TableMetadata struct {
	primaryKeys    map[string][]string         // table name -> primary key field names (supports composite keys)
	requiredFields map[string]map[string]bool   // table name -> field name -> required
	fieldTypes     map[string]map[string]string // table name -> field name -> type (e.g., "bool", "code", "text")
	mu             sync.RWMutex
}

// tableDefYAML represents the structure of table YAML files
type tableDefYAML struct {
	Table struct {
		Name   string          `yaml:"name"`
		Fields []tableFieldDef `yaml:"fields"`
	} `yaml:"table"`
}

// tableFieldDef represents a field in the table definition
type tableFieldDef struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	PrimaryKey bool   `yaml:"primary_key"`
	Required   bool   `yaml:"required"`
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
			fieldTypes:     make(map[string]map[string]string),
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

	tablesPath := filepath.Join(rootPath, "backend", "business-logic", "tables", "definitions")

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
	fTypes := make(map[string]string)
	for _, field := range tableDef.Table.Fields {
		if field.PrimaryKey {
			pkFields = append(pkFields, field.Name)
		}
		if field.Required {
			reqFields[field.Name] = true
		}
		if field.Type != "" {
			fTypes[field.Name] = field.Type
		}
	}
	if len(pkFields) > 0 {
		tm.primaryKeys[tableDef.Table.Name] = pkFields
	}
	if len(reqFields) > 0 {
		tm.requiredFields[tableDef.Table.Name] = reqFields
	}
	if len(fTypes) > 0 {
		tm.fieldTypes[tableDef.Table.Name] = fTypes
	}

	return nil
}

// GetPrimaryKeyField returns the first primary key field name for a table
func (tm *TableMetadata) GetPrimaryKeyField(tableName string) string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if fields := tm.primaryKeys[tableName]; len(fields) > 0 {
		return fields[0]
	}
	return ""
}

// GetPrimaryKeyFields returns all primary key field names for a table (supports composite keys)
func (tm *TableMetadata) GetPrimaryKeyFields(tableName string) []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.primaryKeys[tableName]
}

// GetFieldType returns the YAML type for a field (e.g., "bool", "code", "text")
func (tm *TableMetadata) GetFieldType(tableName, fieldName string) string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if types, ok := tm.fieldTypes[tableName]; ok {
		return types[fieldName]
	}
	return ""
}

// IsFieldRequired returns whether a field is required for a table
func (tm *TableMetadata) IsFieldRequired(tableName, fieldName string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if reqFields, ok := tm.requiredFields[tableName]; ok {
		return reqFields[fieldName]
	}
	return false
}
