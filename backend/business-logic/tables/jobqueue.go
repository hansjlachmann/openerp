package tables

import (
	"errors"

	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/mail"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
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
	// The addresses themselves are checked by OnValidate_Notification_email when a user
	// changes them, not here: a list saved before that check must not stop the scheduler
	// from rescheduling the job (it sends to the valid addresses and logs the others).
	if err := ftables.CheckMaxLength(gtables.JobQueueTableName, "notification_email", string(t.Notification_email), 250); err != nil {
		return err
	}

	return nil
}

// ========================================
// Field Validation Overrides
// ========================================

// OnValidate_Notification_email checks the notification address list (one or more
// addresses separated by ";") and stores it tidied up: "a@x.no; b@x.no".
func (t *JobQueue) OnValidate_Notification_email() error {
	items, err := mail.ParseAddressList(t.Notification_email.String())
	if err != nil {
		var listErr *mail.AddressListError
		if errors.As(err, &listErr) && listErr.Duplicate {
			return apperrors.EmailDuplicate(gtables.JobQueueTableName, "notification_email", listErr.Address)
		}
		if errors.As(err, &listErr) {
			return apperrors.EmailInvalid(gtables.JobQueueTableName, "notification_email", listErr.Address)
		}
		return err
	}
	t.Notification_email = types.NewText(mail.JoinAddressList(items))
	return ftables.CheckMaxLength(gtables.JobQueueTableName, "notification_email", string(t.Notification_email), 250)
}

// OnValidate_Notification_language checks that the notification language exists
func (t *JobQueue) OnValidate_Notification_language() error {
	if t.Notification_language.IsEmpty() {
		return nil
	}
	var language Language
	language.InitWithDBType(t.GetDB(), t.GetCompany(), t.GetDBType())
	if !language.Get(t.Notification_language) {
		return apperrors.RelatedNotFound(gtables.JobQueueTableName, "notification_language", t.Notification_language.String())
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
