package codeunits

import (
	"sync"

	fcodeunits "github.com/hansjlachmann/openerp/backend/foundation/codeunits"
)

var (
	registry       = make(map[int]fcodeunits.Factory)
	registryByName = make(map[string]int)
	registryMu     sync.RWMutex
)

// Register adds a codeunit factory to the registry
func Register(id int, name string, factory fcodeunits.Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[id] = factory
	registryByName[name] = id
}

// Get retrieves a codeunit factory by ID
func Get(id int) (fcodeunits.Factory, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	factory, ok := registry[id]
	return factory, ok
}

// GetByName retrieves a codeunit factory by name
func GetByName(name string) (fcodeunits.Factory, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	id, ok := registryByName[name]
	if !ok {
		return nil, false
	}
	return registry[id], true
}

// List returns all registered codeunit IDs and names
func List() map[int]string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	result := make(map[int]string)
	for name, id := range registryByName {
		result[id] = name
	}
	return result
}
