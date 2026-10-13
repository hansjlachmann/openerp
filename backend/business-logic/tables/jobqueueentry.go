package tables

import (
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// jobQueueEntryLogic is the business logic of table 474 Job_Queue_Entry.
// Add a trigger when it is needed (as in BC): the available triggers are listed at the
// top of generated/tables/jobqueueentry_gen.go. The field rules of the definition
// (required, length, range) are checked after OnInsert and OnModify.
type jobQueueEntryLogic struct{}

func init() { gtables.RegisterJobQueueEntryLogic(jobQueueEntryLogic{}) }

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
