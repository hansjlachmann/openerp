package tables

import (
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// salesLineLogic is the business logic of table 37 Sales_Line.
// Add a trigger when it is needed: the available triggers are listed at the
// top of generated/tables/salesline_gen.go. The field rules of the definition
// (required, length, range) are checked after OnInsert and OnModify.
type salesLineLogic struct{}

func init() { gtables.RegisterSalesLineLogic(salesLineLogic{}) }

// OnInsert runs before a new record is inserted
func (salesLineLogic) OnInsert(rec *gtables.SalesLine) error {
	return nil
}

// OnModify runs before a record is modified
func (salesLineLogic) OnModify(rec *gtables.SalesLine) error {
	return nil
}

// OnDelete runs before a record is deleted
func (salesLineLogic) OnDelete(rec *gtables.SalesLine) error {
	return nil
}

// OnRename runs before a primary key field is changed (rec.OldValue has the old value)
func (salesLineLogic) OnRename(rec *gtables.SalesLine) error {
	return nil
}
