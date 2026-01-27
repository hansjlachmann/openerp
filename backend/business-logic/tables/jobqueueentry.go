package tables

import (
	"errors"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// JobQueueEntry wraps JobQueueEntryBase and adds trigger methods
type JobQueueEntry struct {
	gtables.JobQueueEntryBase
}

// NewJobQueueEntry creates a new JobQueueEntry instance
func NewJobQueueEntry() *JobQueueEntry {
	return &JobQueueEntry{}
}

// Init initializes the record with database context and sets up triggers
func (t *JobQueueEntry) Init(db database.Executor, company string) {
	t.JobQueueEntryBase.Init(db, company)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *JobQueueEntry) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *JobQueueEntry) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *JobQueueEntry) OnDelete(db database.Executor, company string) error {
	// TODO: Add checks for related records (if any)
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *JobQueueEntry) OnRename() error {
	// TODO: Update related records if needed
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *JobQueueEntry) Validate() error {
	if len(t.Job_queue_no) > 20 {
		return errors.New("job_queue_no cannot exceed 20 characters")
	}
	if len(t.User_id) > 20 {
		return errors.New("user_id cannot exceed 20 characters")
	}
	if len(t.Description) > 100 {
		return errors.New("description cannot exceed 100 characters")
	}

	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
// Example:
// func (t *JobQueueEntry) CalculateSomething() error {
//     // Your logic here
//     return nil
// }
