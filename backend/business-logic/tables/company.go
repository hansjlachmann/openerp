package tables

import (
	"fmt"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	"sort"

	"github.com/hansjlachmann/openerp/backend/foundation/company"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// Company wraps CompanyBase and adds trigger methods
type Company struct {
	gtables.CompanyBase
}

// NewCompany creates a new Company instance
func NewCompany() *Company {
	return &Company{}
}

// InitWithDBType initializes the record with database context and type and sets up
// triggers. The API creates tables via the tables.Table interface and calls this method,
// so the wiring must live here for triggers and OnValidate_* overrides to fire.
func (t *Company) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.CompanyBase.InitWithDBType(db, company, dbType)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
	t.SetSelf(t)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *Company) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *Company) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *Company) OnDelete(db database.Executor, company string) error {
	return nil
}

// OnRename trigger - called before renaming (changing primary key). The name prefixes
// every table of the company ("name$Customer"), so the company's data moves with it: its
// tables, indexes and sequences are renamed and its indexes and SIFT totals built for the
// new name, in the transaction of the Modify (the API's modifyInTransaction). The
// generated Modify then carries the name over to User Member (table relation).
func (t *Company) OnRename() error {
	oldName := fmt.Sprint(t.OldValue("name"))
	newName, err := company.NormalizeName(t.Name.String())
	if err != nil {
		return err
	}
	t.Name = types.NewText(newName)
	if newName == oldName {
		return nil
	}

	var existing Company
	existing.InitWithDBType(t.GetDB(), "", t.GetDBType())
	if existing.Get(types.NewText(newName)) {
		return apperrors.CompanyAlreadyExists(newName)
	}

	if err := company.RenameObjects(t.GetDB(), t.GetDBType(), oldName, newName); err != nil {
		return err
	}
	return syncCompanyKeys(t.GetDB(), t.GetDBType(), newName)
}

// syncCompanyKeys creates the indexes and SIFT totals of every company table of company
// (after a rename: SQLite indexes and all SIFT objects are built again for the new name).
func syncCompanyKeys(db database.Executor, dbType database.DBType, companyName string) error {
	existing, err := company.Tables(db, dbType, companyName)
	if err != nil {
		return err
	}
	has := map[string]bool{}
	for _, name := range existing {
		has[name] = true
	}
	done := map[string]bool{}
	names := ListTableNames()
	sort.Strings(names)
	for _, name := range names {
		factory, ok := GetTableFactory(name)
		if !ok {
			continue
		}
		table := factory()
		table.InitWithDBType(db, companyName, dbType)
		syncer, ok := table.(interface {
			SyncKeys(db database.Executor, company string, dbType database.DBType) error
			IsGlobal() bool
		})
		if !ok || syncer.IsGlobal() || done[table.GetTableName()] || !has[companyName+"$"+table.GetTableName()] {
			continue
		}
		done[table.GetTableName()] = true
		if err := syncer.SyncKeys(db, companyName, dbType); err != nil {
			return fmt.Errorf("%s: %w", table.GetTableName(), err)
		}
	}
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *Company) Validate() error {
	if err := ftables.CheckRequired(gtables.CompanyTableName, "name", t.Name.IsEmpty()); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.CompanyTableName, "name", string(t.Name), 100); err != nil {
		return err
	}

	return nil
}
