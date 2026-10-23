package tables

import (
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// fALedgerEntryLogic is the business logic of table 5601 FA_Ledger_Entry.
// Add a trigger when it is needed: the available triggers are listed at the
// top of generated/tables/faledgerentry_gen.go. The field rules of the definition
// (required, length, range) are checked after OnInsert and OnModify.
type fALedgerEntryLogic struct{}

func init() { gtables.RegisterFALedgerEntryLogic(fALedgerEntryLogic{}) }

// OnInsert runs before a new record is inserted
func (fALedgerEntryLogic) OnInsert(rec *gtables.FALedgerEntry) error {
	return nil
}

// OnModify runs before a record is modified
func (fALedgerEntryLogic) OnModify(rec *gtables.FALedgerEntry) error {
	return nil
}

// OnDelete runs before a record is deleted
func (fALedgerEntryLogic) OnDelete(rec *gtables.FALedgerEntry) error {
	return nil
}

// OnRename runs before a primary key field is changed (rec.OldValue has the old value)
func (fALedgerEntryLogic) OnRename(rec *gtables.FALedgerEntry) error {
	return nil
}
