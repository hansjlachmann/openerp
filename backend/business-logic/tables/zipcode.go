package tables

import (
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// zIPCodeLogic is the business logic of table 225 ZIP_Code.
// Add a trigger when it is needed: the available triggers are listed at the
// top of generated/tables/zipcode_gen.go. The field rules of the definition
// (required, length, range) are checked after OnInsert and OnModify.
type zIPCodeLogic struct{}

func init() { gtables.RegisterZIPCodeLogic(zIPCodeLogic{}) }

// OnInsert runs before a new record is inserted
func (zIPCodeLogic) OnInsert(rec *gtables.ZIPCode) error {
	return nil
}

// OnModify runs before a record is modified
func (zIPCodeLogic) OnModify(rec *gtables.ZIPCode) error {
	return nil
}

// OnDelete runs before a record is deleted
func (zIPCodeLogic) OnDelete(rec *gtables.ZIPCode) error {
	return nil
}

// OnRename runs before a primary key field is changed (rec.OldValue has the old value)
func (zIPCodeLogic) OnRename(rec *gtables.ZIPCode) error {
	return nil
}
