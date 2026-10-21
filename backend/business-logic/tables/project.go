package tables

import (
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// projectLogic is the business logic of table 12 Project.
// Add a trigger when it is needed: the available triggers are listed at the
// top of generated/tables/project_gen.go. The field rules of the definition
// (required, length, range) are checked after OnInsert and OnModify.
type projectLogic struct{}

func init() { gtables.RegisterProjectLogic(projectLogic{}) }

// OnInsert runs before a new record is inserted
func (projectLogic) OnInsert(rec *gtables.Project) error {
	return nil
}

// OnModify runs before a record is modified
func (projectLogic) OnModify(rec *gtables.Project) error {
	return nil
}

// OnDelete runs before a record is deleted
func (projectLogic) OnDelete(rec *gtables.Project) error {
	return nil
}

// OnRename runs before a primary key field is changed (rec.OldValue has the old value)
func (projectLogic) OnRename(rec *gtables.Project) error {
	return nil
}
