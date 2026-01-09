package pages

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Registry manages page definitions
type Registry struct {
	pages map[int]*PageDefinition
	menu  *MenuDefinition
	mu    sync.RWMutex
}

var (
	instance *Registry
	once     sync.Once
)

// GetRegistry returns the singleton page registry
func GetRegistry() *Registry {
	once.Do(func() {
		instance = &Registry{
			pages: make(map[int]*PageDefinition),
		}
		// Load page definitions automatically
		if err := instance.LoadPages(); err != nil {
			fmt.Printf("Warning: Failed to load page definitions: %v\n", err)
		}
		// Load menu
		if err := instance.LoadMenu(); err != nil {
			fmt.Printf("Warning: Failed to load menu: %v\n", err)
		}
	})
	return instance
}

// LoadPages loads all page definitions from YAML files
func (r *Registry) LoadPages() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Get project root
	rootPath, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	pagesPath := filepath.Join(rootPath, "src", "business-logic", "pages", "definitions")

	// Check if pages directory exists
	if _, err := os.Stat(pagesPath); os.IsNotExist(err) {
		return fmt.Errorf("pages directory not found: %s", pagesPath)
	}

	// Find all YAML files
	files, err := filepath.Glob(filepath.Join(pagesPath, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to read pages directory: %w", err)
	}

	loadedCount := 0
	for _, file := range files {
		if err := r.loadPageFile(file); err != nil {
			fmt.Printf("Warning: Failed to load %s: %v\n", filepath.Base(file), err)
		} else {
			loadedCount++
		}
	}

	if loadedCount == 0 {
		return fmt.Errorf("no page definitions loaded")
	}

	fmt.Printf("✓ Loaded %d page definition(s)\n", loadedCount)
	return nil
}

// loadPageFile loads a single page definition file
func (r *Registry) loadPageFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var pageDef PageDefinition
	if err := yaml.Unmarshal(data, &pageDef); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Store by page ID
	r.pages[pageDef.Page.ID] = &pageDef

	return nil
}

// LoadMenu loads the menu definition
func (r *Registry) LoadMenu() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Get project root
	rootPath, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	menuPath := filepath.Join(rootPath, "src", "business-logic", "pages", "menu.yaml")

	// Check if menu file exists
	if _, err := os.Stat(menuPath); os.IsNotExist(err) {
		return fmt.Errorf("menu file not found: %s", menuPath)
	}

	data, err := os.ReadFile(menuPath)
	if err != nil {
		return fmt.Errorf("failed to read menu file: %w", err)
	}

	var menuDef MenuDefinition
	if err := yaml.Unmarshal(data, &menuDef); err != nil {
		return fmt.Errorf("failed to parse menu YAML: %w", err)
	}

	// Set default Enabled to true for menu items
	for i := range menuDef.Menu {
		for j := range menuDef.Menu[i].Items {
			if menuDef.Menu[i].Items[j].PageID > 0 && !menuDef.Menu[i].Items[j].Separator {
				// If Enabled is not explicitly set to false, default to true
				if menuDef.Menu[i].Items[j].Enabled == false {
					// Check if it was explicitly set in YAML
					// If not set, default to true
					menuDef.Menu[i].Items[j].Enabled = true
				}
			}
		}
	}

	r.menu = &menuDef

	fmt.Printf("✓ Loaded menu with %d group(s)\n", len(menuDef.Menu))
	return nil
}

// GetPage returns a page definition by ID
func (r *Registry) GetPage(pageID int) (*PageDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	page, ok := r.pages[pageID]
	if !ok {
		return nil, fmt.Errorf("page %d not found", pageID)
	}

	return page, nil
}

// GetMenu returns the menu definition
func (r *Registry) GetMenu() *MenuDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.menu
}

// GetAllPages returns all loaded page definitions
func (r *Registry) GetAllPages() map[int]*PageDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	pages := make(map[int]*PageDefinition, len(r.pages))
	for id, page := range r.pages {
		pages[id] = page
	}

	return pages
}

// findProjectRoot finds the project root directory
func findProjectRoot() (string, error) {
	// Start from current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up until we find go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding go.mod
			return "", fmt.Errorf("project root not found (no go.mod)")
		}
		dir = parent
	}
}
