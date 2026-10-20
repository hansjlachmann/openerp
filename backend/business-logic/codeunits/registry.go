package codeunits

import (
	"fmt"
	"sync"

	fcodeunits "github.com/hansjlachmann/openerp/backend/foundation/codeunits"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"

	// The table triggers (business logic) run for the records written here
	_ "github.com/hansjlachmann/openerp/backend/business-logic/tables"
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

// Run runs a codeunit by its ID on a record (Codeunit.Run): the codeunit works in the
// record's database and company. It is an error when no codeunit has that ID, or when the
// record is not of the codeunit's source table.
func Run(id int, record interface {
	GetTableName() string
	GetDB() database.Executor
	GetCompany() string
	GetDBType() database.DBType
}) (fcodeunits.Result, error) {
	factory, ok := Get(id)
	if !ok {
		return fcodeunits.Result{}, fmt.Errorf("codeunit %d does not exist", id)
	}
	cu := factory(record.GetDB(), record.GetCompany(), record.GetDBType())
	if source := cu.SourceTable(); source != "" {
		newRecord, ok := gtables.GetTableFactory(source)
		if !ok || newRecord().GetTableName() != record.GetTableName() {
			return fcodeunits.Result{}, fmt.Errorf("codeunit %d %s runs on %s, not on %s", id, cu.Name(), source, record.GetTableName())
		}
	}
	return cu.Run(record)
}
