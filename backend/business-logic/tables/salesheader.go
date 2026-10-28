package tables

import (
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// salesHeaderLogic is the business logic of table 36 Sales_Header.
// Add a trigger when it is needed: the available triggers are listed at the
// top of generated/tables/salesheader_gen.go. The field rules of the definition
// (required, length, range) are checked after OnInsert and OnModify.
type salesHeaderLogic struct{}

func init() { gtables.RegisterSalesHeaderLogic(salesHeaderLogic{}) }

// OnInsert runs before a new record is inserted
func (salesHeaderLogic) OnInsert(rec *gtables.SalesHeader) error {
	return nil
}

// OnModify runs before a record is modified
func (salesHeaderLogic) OnModify(rec *gtables.SalesHeader) error {
	return nil
}

// OnDelete runs before a record is deleted
func (salesHeaderLogic) OnDelete(rec *gtables.SalesHeader) error {
	return nil
}

// OnRename runs before a primary key field is changed (rec.OldValue has the old value)
func (salesHeaderLogic) OnRename(rec *gtables.SalesHeader) error {
	return nil
}
