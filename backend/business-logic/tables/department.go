package tables

import (
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// departmentLogic is the business logic of table 11 Department.
// Add a trigger when it is needed: the available triggers are listed at the
// top of generated/tables/department_gen.go. The field rules of the definition
// (required, length, range) are checked after OnInsert and OnModify.
type departmentLogic struct{}

func init() { gtables.RegisterDepartmentLogic(departmentLogic{}) }

// OnInsert runs before a new record is inserted
func (departmentLogic) OnInsert(rec *gtables.Department) error {
	return nil
}

// OnModify runs before a record is modified
func (departmentLogic) OnModify(rec *gtables.Department) error {
	return nil
}

// OnDelete runs before a record is deleted
func (departmentLogic) OnDelete(rec *gtables.Department) error {
	return nil
}

// OnRename runs before a primary key field is changed (rec.OldValue has the old value)
func (departmentLogic) OnRename(rec *gtables.Department) error {
	return nil
}
