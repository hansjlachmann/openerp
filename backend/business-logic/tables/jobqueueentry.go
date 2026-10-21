package tables

import (
	fcodeunits "github.com/hansjlachmann/openerp/backend/foundation/codeunits"
	"github.com/hansjlachmann/openerp/backend/foundation/functions"
	"github.com/hansjlachmann/openerp/backend/foundation/i18n"
	"github.com/hansjlachmann/openerp/backend/foundation/system"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// jobQueueEntryLogic is the business logic of table 474 Job_Queue_Entry.
// Add a trigger when it is needed: the available triggers are listed at the
// top of generated/tables/jobqueueentry_gen.go. The field rules of the definition
// (required, length, range) are checked after OnInsert and OnModify.
type jobQueueEntryLogic struct{}

func init() {
	gtables.RegisterJobQueueEntryLogic(jobQueueEntryLogic{})
	// Functions pages call with the current line (run_object: table:Job_Queue_Entry.<name>)
	functions.Register("table", gtables.JobQueueEntryTableName, "ShowLineNo", functions.For(jobQueueEntryLogic{}.ShowLineNo))
	functions.Register("table", gtables.JobQueueEntryTableName, "Test", functions.For(jobQueueEntryLogic{}.Test))
}

// Test shows a simple message (button "TEST" on the Job Queue Entries page: a plain action
// button calling a page function)
func (jobQueueEntryLogic) Test(rec *gtables.JobQueueEntry) (fcodeunits.Result, error) {
	text := i18n.GetInstance().Message("JOBQUEUEENTRY_TEST", fcodeunits.CurrentLanguage())
	return system.Message(text, rec.Entry_no), nil
}

// ShowLineNo shows the number of the current line in a message (menu item "Test Line 1" on
// the Job Queue Entries page: an example of a page function on a table)
func (jobQueueEntryLogic) ShowLineNo(rec *gtables.JobQueueEntry) (fcodeunits.Result, error) {
	text := i18n.GetInstance().Message("JOBQUEUEENTRY_LINE_NO", fcodeunits.CurrentLanguage())
	return system.Message(text, rec.Entry_no, rec.Job_queue_no), nil
}

// OnInsert runs before a new record is inserted
func (jobQueueEntryLogic) OnInsert(rec *gtables.JobQueueEntry) error {
	return nil
}

// OnModify runs before a record is modified
func (jobQueueEntryLogic) OnModify(rec *gtables.JobQueueEntry) error {
	return nil
}

// OnDelete runs before a record is deleted
func (jobQueueEntryLogic) OnDelete(rec *gtables.JobQueueEntry) error {
	return nil
}

// OnRename runs before a primary key field is changed (rec.OldValue has the old value)
func (jobQueueEntryLogic) OnRename(rec *gtables.JobQueueEntry) error {
	return nil
}
