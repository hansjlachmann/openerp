package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// PageDef represents a page definition from YAML
type PageDef struct {
	Page Page `yaml:"page"`
}

// Page represents the page configuration
type Page struct {
	ID               int       `yaml:"id"`
	Type             string    `yaml:"type"`
	Name             string    `yaml:"name"`
	SourceTable      string    `yaml:"source_table"`
	Caption          string    `yaml:"caption"`
	ListPageID       int       `yaml:"list_page_id,omitempty"`
	CardPageID       int       `yaml:"card_page_id,omitempty"`
	EnableNavigation bool      `yaml:"enable_navigation,omitempty"`
	Editable         bool      `yaml:"editable,omitempty"`
	FocusField       string    `yaml:"focus_field,omitempty"`
	Layout           Layout    `yaml:"layout,omitempty"`
	Actions          []Action  `yaml:"actions,omitempty"`
	Columns          []Column  `yaml:"columns,omitempty"`
}

// Layout represents the page layout
type Layout struct {
	Sections []Section `yaml:"sections,omitempty"`
}

// Section represents a layout section
type Section struct {
	Name    string  `yaml:"name"`
	Caption string  `yaml:"caption,omitempty"`
	Fields  []Field `yaml:"fields,omitempty"`
}

// Field represents a field in a section
type Field struct {
	Source        string `yaml:"source"`
	Caption       string `yaml:"caption,omitempty"`
	Editable      bool   `yaml:"editable,omitempty"`
	Visible       *bool  `yaml:"visible,omitempty"`
	Importance    string `yaml:"importance,omitempty"`
	TableRelation string `yaml:"table_relation,omitempty"`
}

// Column represents a column in a list page
type Column struct {
	Source        string `yaml:"source"`
	Caption       string `yaml:"caption,omitempty"`
	Width         int    `yaml:"width,omitempty"`
	Sortable      bool   `yaml:"sortable,omitempty"`
	Filterable    bool   `yaml:"filterable,omitempty"`
	Importance    string `yaml:"importance,omitempty"`
	TableRelation string `yaml:"table_relation,omitempty"`
}

// Action represents a page action
type Action struct {
	Name      string `yaml:"name"`
	Caption   string `yaml:"caption,omitempty"`
	Shortcut  string `yaml:"shortcut,omitempty"`
	Promoted  bool   `yaml:"promoted,omitempty"`
	Enabled   *bool  `yaml:"enabled,omitempty"`
	RunPage   int    `yaml:"run_page,omitempty"`
	RunObject string `yaml:"run_object,omitempty"`
}

// PageExtension represents a page extension definition
type PageExtension struct {
	Extends int      `yaml:"extends"`
	Version string   `yaml:"version,omitempty"`
	Actions []Action `yaml:"actions,omitempty"`
	Layout  Layout   `yaml:"layout,omitempty"`
}

// TableExtension represents a table extension definition
type TableExtension struct {
	ExtendsTable int           `yaml:"extends_table"`
	Fields       []TableField  `yaml:"fields,omitempty"`
}

// TableField represents a field in a table extension
type TableField struct {
	Name    string `yaml:"name"`
	Type    string `yaml:"type"`
	Length  int    `yaml:"length,omitempty"`
	Caption string `yaml:"caption,omitempty"`
}

func main() {
	corePath := flag.String("core", "", "Path to core business-logic directory")
	extensionsPath := flag.String("extensions", "", "Path to extensions directory")
	outputPath := flag.String("output", "", "Path to output merged directory")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	flag.Parse()

	if *corePath == "" || *extensionsPath == "" || *outputPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: extmerge --core <path> --extensions <path> --output <path>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintln(os.Stderr, "  --core        Path to core business-logic directory")
		fmt.Fprintln(os.Stderr, "  --extensions  Path to extensions directory")
		fmt.Fprintln(os.Stderr, "  --output      Path to output merged directory")
		fmt.Fprintln(os.Stderr, "  --verbose     Enable verbose output")
		os.Exit(1)
	}

	if err := run(*corePath, *extensionsPath, *outputPath, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(corePath, extensionsPath, outputPath string, verbose bool) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Copy core files to output
	if err := copyDir(corePath, outputPath, verbose); err != nil {
		return fmt.Errorf("failed to copy core files: %w", err)
	}

	// Find and process page extensions
	pageExtensions, err := findExtensions(extensionsPath, "pages", ".extend.yaml")
	if err != nil {
		return fmt.Errorf("failed to find page extensions: %w", err)
	}

	for _, extPath := range pageExtensions {
		if err := mergePageExtension(extPath, outputPath, verbose); err != nil {
			return fmt.Errorf("failed to merge page extension %s: %w", extPath, err)
		}
	}

	// Find and process table extensions
	tableExtensions, err := findExtensions(extensionsPath, "tables", ".extend.yaml")
	if err != nil {
		return fmt.Errorf("failed to find table extensions: %w", err)
	}

	for _, extPath := range tableExtensions {
		if err := mergeTableExtension(extPath, outputPath, verbose); err != nil {
			return fmt.Errorf("failed to merge table extension %s: %w", extPath, err)
		}
	}

	// Copy new objects (non-extend files)
	if err := copyNewObjects(extensionsPath, outputPath, verbose); err != nil {
		return fmt.Errorf("failed to copy new objects: %w", err)
	}

	if verbose {
		fmt.Printf("Merge completed successfully\n")
		fmt.Printf("  Page extensions merged: %d\n", len(pageExtensions))
		fmt.Printf("  Table extensions merged: %d\n", len(tableExtensions))
	}

	return nil
}

func copyDir(src, dst string, verbose bool) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		if verbose {
			fmt.Printf("Copying: %s\n", relPath)
		}

		return copyFile(path, dstPath)
	})
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func findExtensions(basePath, subdir, suffix string) ([]string, error) {
	var extensions []string
	searchPath := filepath.Join(basePath, subdir)

	if _, err := os.Stat(searchPath); os.IsNotExist(err) {
		return extensions, nil
	}

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), suffix) {
			extensions = append(extensions, path)
		}
		return nil
	})

	return extensions, err
}

func mergePageExtension(extPath, outputPath string, verbose bool) error {
	// Read extension file
	extData, err := os.ReadFile(extPath)
	if err != nil {
		return fmt.Errorf("failed to read extension: %w", err)
	}

	var ext PageExtension
	if err := yaml.Unmarshal(extData, &ext); err != nil {
		return fmt.Errorf("failed to parse extension: %w", err)
	}

	if ext.Extends == 0 {
		return fmt.Errorf("extension must specify 'extends' field with page ID")
	}

	// Find core page file
	corePagePath, err := findCorePageFile(outputPath, ext.Extends)
	if err != nil {
		return fmt.Errorf("failed to find core page %d: %w", ext.Extends, err)
	}

	// Read core page
	coreData, err := os.ReadFile(corePagePath)
	if err != nil {
		return fmt.Errorf("failed to read core page: %w", err)
	}

	var pageDef PageDef
	if err := yaml.Unmarshal(coreData, &pageDef); err != nil {
		return fmt.Errorf("failed to parse core page: %w", err)
	}

	// Merge extension into page
	mergePage(&pageDef.Page, &ext)

	// Write merged page
	mergedData, err := yaml.Marshal(&pageDef)
	if err != nil {
		return fmt.Errorf("failed to marshal merged page: %w", err)
	}

	if err := os.WriteFile(corePagePath, mergedData, 0644); err != nil {
		return fmt.Errorf("failed to write merged page: %w", err)
	}

	if verbose {
		fmt.Printf("Merged page extension: %s -> page %d\n", filepath.Base(extPath), ext.Extends)
	}

	return nil
}

func findCorePageFile(outputPath string, pageID int) (string, error) {
	pagesPath := filepath.Join(outputPath, "pages", "definitions")

	var found string
	err := filepath.Walk(pagesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// Match files like "21-customer-card.yaml"
		name := info.Name()
		if strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".extend.yaml") {
			// Parse the ID from filename
			parts := strings.SplitN(name, "-", 2)
			if len(parts) >= 1 {
				var id int
				if _, err := fmt.Sscanf(parts[0], "%d", &id); err == nil && id == pageID {
					found = path
					return filepath.SkipAll
				}
			}
		}
		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return "", err
	}

	if found == "" {
		return "", fmt.Errorf("page with ID %d not found", pageID)
	}

	return found, nil
}

func mergePage(page *Page, ext *PageExtension) {
	// Append actions
	page.Actions = append(page.Actions, ext.Actions...)

	// Merge layout sections
	for _, extSection := range ext.Layout.Sections {
		merged := false
		for i, pageSection := range page.Layout.Sections {
			if pageSection.Name == extSection.Name {
				// Merge fields into existing section
				page.Layout.Sections[i].Fields = mergeFields(pageSection.Fields, extSection.Fields)
				merged = true
				break
			}
		}
		if !merged {
			// Add as new section
			page.Layout.Sections = append(page.Layout.Sections, extSection)
		}
	}
}

func mergeFields(existing, new []Field) []Field {
	result := make([]Field, len(existing))
	copy(result, existing)

	for _, newField := range new {
		found := false
		for i, existingField := range result {
			if existingField.Source == newField.Source {
				// Override existing field
				result[i] = newField
				found = true
				break
			}
		}
		if !found {
			// Add new field
			result = append(result, newField)
		}
	}

	return result
}

func mergeTableExtension(extPath, outputPath string, verbose bool) error {
	// Read extension file
	extData, err := os.ReadFile(extPath)
	if err != nil {
		return fmt.Errorf("failed to read extension: %w", err)
	}

	var ext TableExtension
	if err := yaml.Unmarshal(extData, &ext); err != nil {
		return fmt.Errorf("failed to parse extension: %w", err)
	}

	if ext.ExtendsTable == 0 {
		return fmt.Errorf("extension must specify 'extends_table' field with table ID")
	}

	// Find core table file
	coreTablePath, err := findCoreTableFile(outputPath, ext.ExtendsTable)
	if err != nil {
		return fmt.Errorf("failed to find core table %d: %w", ext.ExtendsTable, err)
	}

	// Read core table as raw YAML to preserve structure
	coreData, err := os.ReadFile(coreTablePath)
	if err != nil {
		return fmt.Errorf("failed to read core table: %w", err)
	}

	var tableDef map[string]interface{}
	if err := yaml.Unmarshal(coreData, &tableDef); err != nil {
		return fmt.Errorf("failed to parse core table: %w", err)
	}

	// Merge table extension
	if table, ok := tableDef["table"].(map[string]interface{}); ok {
		if fields, ok := table["fields"].([]interface{}); ok {
			// Add new fields
			for _, newField := range ext.Fields {
				fieldMap := map[string]interface{}{
					"name": newField.Name,
					"type": newField.Type,
				}
				if newField.Length > 0 {
					fieldMap["length"] = newField.Length
				}
				if newField.Caption != "" {
					fieldMap["caption"] = newField.Caption
				}
				fields = append(fields, fieldMap)
			}
			table["fields"] = fields
		}
	}

	// Write merged table
	mergedData, err := yaml.Marshal(&tableDef)
	if err != nil {
		return fmt.Errorf("failed to marshal merged table: %w", err)
	}

	if err := os.WriteFile(coreTablePath, mergedData, 0644); err != nil {
		return fmt.Errorf("failed to write merged table: %w", err)
	}

	if verbose {
		fmt.Printf("Merged table extension: %s -> table %d\n", filepath.Base(extPath), ext.ExtendsTable)
	}

	return nil
}

func findCoreTableFile(outputPath string, tableID int) (string, error) {
	tablesPath := filepath.Join(outputPath, "tables", "definitions")

	var found string
	err := filepath.Walk(tablesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()
		if strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".extend.yaml") {
			// Read file to check table ID
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			var def struct {
				Table struct {
					ID int `yaml:"id"`
				} `yaml:"table"`
			}
			if err := yaml.Unmarshal(data, &def); err != nil {
				return nil
			}
			if def.Table.ID == tableID {
				found = path
				return filepath.SkipAll
			}
		}
		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return "", err
	}

	if found == "" {
		return "", fmt.Errorf("table with ID %d not found", tableID)
	}

	return found, nil
}

func copyNewObjects(extensionsPath, outputPath string, verbose bool) error {
	// Copy new tables
	newTablesPath := filepath.Join(extensionsPath, "custom", "tables")
	if _, err := os.Stat(newTablesPath); err == nil {
		destPath := filepath.Join(outputPath, "tables", "definitions")
		if err := copyYAMLFiles(newTablesPath, destPath, verbose); err != nil {
			return err
		}
	}

	// Copy new pages
	newPagesPath := filepath.Join(extensionsPath, "custom", "pages")
	if _, err := os.Stat(newPagesPath); err == nil {
		destPath := filepath.Join(outputPath, "pages", "definitions")
		if err := copyYAMLFiles(newPagesPath, destPath, verbose); err != nil {
			return err
		}
	}

	// Copy new codeunits
	newCodeunitsPath := filepath.Join(extensionsPath, "codeunits")
	if _, err := os.Stat(newCodeunitsPath); err == nil {
		destPath := filepath.Join(outputPath, "codeunits")
		if err := copyGoFiles(newCodeunitsPath, destPath, verbose); err != nil {
			return err
		}
	}

	return nil
}

func copyYAMLFiles(src, dst string, verbose bool) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".yaml") {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".extend.yaml") {
			return nil
		}

		destFile := filepath.Join(dst, info.Name())
		if verbose {
			fmt.Printf("Copying new object: %s\n", info.Name())
		}
		return copyFile(path, destFile)
	})
}

func copyGoFiles(src, dst string, verbose bool) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		destFile := filepath.Join(dst, info.Name())
		if verbose {
			fmt.Printf("Copying new codeunit: %s\n", info.Name())
		}
		return copyFile(path, destFile)
	})
}
