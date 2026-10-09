package tables

import (
	"fmt"
	"sort"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
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
	RegisterTableFactory("Country_Region", gtables.CountryRegionTableID, func() ftables.Table {
		return &CountryRegion{}
	})
	RegisterTableFactory("SMTP_Setup", gtables.SMTPSetupTableID, func() ftables.Table {
		return &SMTPSetup{}
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
	RegisterTableFactory("Company", gtables.CompanyTableID, func() ftables.Table {
		return &Company{}
	})
	RegisterTableFactory("User_Role", gtables.UserRoleTableID, func() ftables.Table {
		return &UserRole{}
	})
	RegisterTableFactory("User_Member", gtables.UserMemberTableID, func() ftables.Table {
		return &UserMember{}
	})
	RegisterTableFactory("Permission", gtables.PermissionTableID, func() ftables.Table {
		return &Permission{}
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

// EncryptStoredSecrets encrypts values of encrypted fields (encrypted: true in the table
// YAML) that are stored as typed — saved before the field was encrypted or while no key
// was set — in every table that has such fields: global tables once, company tables in
// each company. Run at startup; does nothing without an encryption key. Returns how many
// values it encrypted.
func EncryptStoredSecrets(db database.Executor, dbType database.DBType, companies []string) (int, error) {
	type secretTable interface {
		ftables.Table
		IsGlobal() bool
		EncryptStoredSecrets() (int, error)
	}
	total := 0
	names := make([]string, 0, len(tableRegistry))
	for name := range tableRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		probe, ok := tableRegistry[name]().(secretTable)
		if !ok {
			continue // no encrypted fields
		}
		scopes := companies
		if probe.IsGlobal() {
			scopes = []string{""}
		}
		for _, company := range scopes {
			t := tableRegistry[name]().(secretTable)
			t.InitWithDBType(db, company, dbType)
			n, err := t.EncryptStoredSecrets()
			total += n
			if err != nil {
				return total, fmt.Errorf("encrypt stored secrets of %s (%s): %w", name, company, err)
			}
		}
	}
	return total, nil
}
