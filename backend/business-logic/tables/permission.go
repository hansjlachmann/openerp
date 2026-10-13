package tables

import (
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// permissionLogic is the business logic of table 2000000005 Permission.
// Add a trigger when it is needed (as in BC): the available triggers are listed at the
// top of generated/tables/permission_gen.go. The field rules of the definition
// (required, length, range) are checked after OnInsert and OnModify.
type permissionLogic struct{}

func init() { gtables.RegisterPermissionLogic(permissionLogic{}) }

// OnInsert runs before a new record is inserted
func (permissionLogic) OnInsert(rec *gtables.Permission) error {
	return nil
}

// OnModify runs before a record is modified
func (permissionLogic) OnModify(rec *gtables.Permission) error {
	return nil
}

// OnDelete runs before a record is deleted
func (permissionLogic) OnDelete(rec *gtables.Permission) error {
	return nil
}

// OnRename runs before a primary key field is changed (rec.OldValue has the old value)
func (permissionLogic) OnRename(rec *gtables.Permission) error {
	return nil
}
