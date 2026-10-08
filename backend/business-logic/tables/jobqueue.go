package tables

import (
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// JobQueue wraps JobQueueBase and adds trigger methods
type JobQueue struct {
	gtables.JobQueueBase
}

// NewJobQueue creates a new JobQueue instance
func NewJobQueue() *JobQueue {
	return &JobQueue{}
}

// InitWithDBType initializes the record with database context and type and sets up
// triggers. The API creates tables via the tables.Table interface and calls this method,
// so the wiring must live here for triggers and OnValidate_* overrides to fire.
func (t *JobQueue) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.JobQueueBase.InitWithDBType(db, company, dbType)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
	t.SetSelf(t)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *JobQueue) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *JobQueue) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *JobQueue) OnDelete(db database.Executor, company string) error {
	// TODO: Add checks for related records (if any)
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *JobQueue) OnRename() error {
	// TODO: Update related records if needed
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *JobQueue) Validate() error {
	if err := ftables.CheckRequired(gtables.JobQueueTableName, "no", t.No.IsEmpty()); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.JobQueueTableName, "no", string(t.No), 20); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.JobQueueTableName, "description", string(t.Description), 100); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.JobQueueTableName, "description_2", string(t.Description_2), 100); err != nil {
		return err
	}

	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
// Example:
// func (t *JobQueue) CalculateSomething() error {
//     // Your logic here
//     return nil
// }
