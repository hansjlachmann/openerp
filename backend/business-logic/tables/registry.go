package tables

import (
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// tableRegistry holds all registered table factories
var tableRegistry = make(map[string]ftables.TableFactory)

// init registers all table factories at package initialization
func init() {
	// Register all tables by name
	RegisterTableFactory("Customer", gtables.CustomerTableID, func() ftables.Table {
		return &Customer{}
	})
	RegisterTableFactory("Payment_terms", gtables.PaymentTermsTableID, func() ftables.Table {
		return &PaymentTerms{}
	})
	RegisterTableFactory("Customer_ledger_entry", gtables.CustomerLedgerEntryTableID, func() ftables.Table {
		return &CustomerLedgerEntry{}
	})
	RegisterTableFactory("User", gtables.UserTableID, func() ftables.Table {
		return &User{}
	})
	RegisterTableFactory("User_preferences", gtables.UserPreferencesTableID, func() ftables.Table {
		return &UserPreferences{}
	})
	RegisterTableFactory("Language", gtables.LanguageTableID, func() ftables.Table {
		return &Language{}
	})
	RegisterTableFactory("Menu", gtables.MenuTableID, func() ftables.Table {
		return &Menu{}
	})
	RegisterTableFactory("Job_Queue", gtables.JobQueueTableID, func() ftables.Table {
		return &JobQueue{}
	})
	RegisterTableFactory("Job_Queue_Entry", gtables.JobQueueEntryTableID, func() ftables.Table {
		return &JobQueueEntry{}
	})
}

// RegisterTableFactory registers a table factory by name
func RegisterTableFactory(name string, id int, factory ftables.TableFactory) {
	tableRegistry[name] = factory
}

// GetTableFactory returns a table factory by name
func GetTableFactory(name string) (ftables.TableFactory, bool) {
	factory, ok := tableRegistry[name]
	return factory, ok
}

// ListTableNames returns all registered table names
func ListTableNames() []string {
	names := make([]string, 0, len(tableRegistry))
	for name := range tableRegistry {
		names = append(names, name)
	}
	return names
}

// TableExists checks if a table name is registered
func TableExists(name string) bool {
	_, ok := tableRegistry[name]
	return ok
}
