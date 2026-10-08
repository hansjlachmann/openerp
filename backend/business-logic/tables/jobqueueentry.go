package tables

import (
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"

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

// InitWithDBType initializes the record with database context and type and sets up
// triggers. The API creates tables via the tables.Table interface and calls this method,
// so the wiring must live here for triggers and OnValidate_* overrides to fire.
func (t *JobQueueEntry) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.JobQueueEntryBase.InitWithDBType(db, company, dbType)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
	t.SetSelf(t)
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
	if err := ftables.CheckMaxLength(gtables.JobQueueEntryTableName, "job_queue_no", string(t.Job_queue_no), 20); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.JobQueueEntryTableName, "user_id", string(t.User_id), 20); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.JobQueueEntryTableName, "description", string(t.Description), 100); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.JobQueueEntryTableName, "error_message", string(t.Error_message), 250); err != nil {
		return err
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
