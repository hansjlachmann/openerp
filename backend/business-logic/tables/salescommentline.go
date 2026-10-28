package tables

import (
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// salesCommentLineLogic is the business logic of table 44 Sales_Comment_Line.
// Add a trigger when it is needed: the available triggers are listed at the
// top of generated/tables/salescommentline_gen.go. The field rules of the definition
// (required, length, range) are checked after OnInsert and OnModify.
type salesCommentLineLogic struct{}

func init() { gtables.RegisterSalesCommentLineLogic(salesCommentLineLogic{}) }

// OnInsert runs before a new record is inserted
func (salesCommentLineLogic) OnInsert(rec *gtables.SalesCommentLine) error {
	return nil
}

// OnModify runs before a record is modified
func (salesCommentLineLogic) OnModify(rec *gtables.SalesCommentLine) error {
	return nil
}

// OnDelete runs before a record is deleted
func (salesCommentLineLogic) OnDelete(rec *gtables.SalesCommentLine) error {
	return nil
}

// OnRename runs before a primary key field is changed (rec.OldValue has the old value)
func (salesCommentLineLogic) OnRename(rec *gtables.SalesCommentLine) error {
	return nil
}
