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
	primaryKeys map[string]string // table name -> primary key field name
	mu          sync.RWMutex
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
	PrimaryKey bool   `yaml:"primary_key"`
}

var (
	tableMetadata     *TableMetadata
	tableMetadataOnce sync.Once
)

// GetTableMetadata returns the singleton table metadata instance
func GetTableMetadata() *TableMetadata {
	tableMetadataOnce.Do(func() {
		tableMetadata = &TableMetadata{
			primaryKeys: make(map[string]string),
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

	tablesPath := filepath.Join(rootPath, "src", "business-logic", "tables", "definitions")

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

	// Find the primary key field
	for _, field := range tableDef.Table.Fields {
		if field.PrimaryKey {
			tm.primaryKeys[tableDef.Table.Name] = field.Name
			break
		}
	}

	return nil
}

// GetPrimaryKeyField returns the primary key field name for a table
func (tm *TableMetadata) GetPrimaryKeyField(tableName string) string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.primaryKeys[tableName]
}
