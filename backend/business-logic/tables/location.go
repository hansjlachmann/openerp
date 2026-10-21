package tables

import (
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// locationLogic is the business logic of table 14 Location.
// Add a trigger when it is needed: the available triggers are listed at the
// top of generated/tables/location_gen.go. The field rules of the definition
// (required, length, range) are checked after OnInsert and OnModify.
type locationLogic struct{}

func init() { gtables.RegisterLocationLogic(locationLogic{}) }

// OnInsert runs before a new record is inserted
func (locationLogic) OnInsert(rec *gtables.Location) error {
	return nil
}

// OnModify runs before a record is modified
func (locationLogic) OnModify(rec *gtables.Location) error {
	return nil
}

// OnDelete runs before a record is deleted
func (locationLogic) OnDelete(rec *gtables.Location) error {
	return nil
}

// OnRename runs before a primary key field is changed (rec.OldValue has the old value)
func (locationLogic) OnRename(rec *gtables.Location) error {
	return nil
}
