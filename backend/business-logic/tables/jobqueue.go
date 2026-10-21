package tables

import (
	"errors"

	fcodeunits "github.com/hansjlachmann/openerp/backend/foundation/codeunits"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/functions"
	"github.com/hansjlachmann/openerp/backend/foundation/i18n"
	"github.com/hansjlachmann/openerp/backend/foundation/mail"
	"github.com/hansjlachmann/openerp/backend/foundation/system"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// jobQueueLogic is the business logic of table 472 Job_Queue. The notification addresses
// are checked by OnValidate_Notification_email when a user changes them, not on Modify: a
// list saved before that check must not stop the scheduler from rescheduling the job (it
// sends to the valid addresses and logs the others).
type jobQueueLogic struct{}

func init() {
	gtables.RegisterJobQueueLogic(jobQueueLogic{})
	// Functions of the Job Queue Card's "Job Queue" menu (examples of card page functions)
	functions.Register("table", gtables.JobQueueTableName, "ShowStatus", functions.For(jobQueueLogic{}.ShowStatus))
	functions.Register("table", gtables.JobQueueTableName, "AddTestEntry", functions.For(jobQueueLogic{}.AddTestEntry))
}

// ShowStatus shows the job queue's status in a message (a card page function)
func (jobQueueLogic) ShowStatus(rec *gtables.JobQueue) (fcodeunits.Result, error) {
	text := i18n.GetInstance().Message("JOBQUEUE_STATUS_IS", fcodeunits.CurrentLanguage())
	return system.Message(text, rec.No, rec.Status.String()), nil
}

// AddTestEntry adds a log entry for the job queue and asks the card to refresh, so the new
// line shows in its Log Entries subpage at once (a header function that changes the lines)
func (jobQueueLogic) AddTestEntry(rec *gtables.JobQueue) (fcodeunits.Result, error) {
	var entry gtables.JobQueueEntry
	entry.InitWithDBType(rec.GetDB(), rec.GetCompany(), rec.GetDBType())
	next := 1
	if entry.FindLast() {
		next = entry.Entry_no + 1
	}
	entry.Init()
	entry.Entry_no = next
	entry.Job_queue_no = rec.No
	entry.Status = gtables.JobQueueEntry_Status.Success
	entry.Description = types.NewText(i18n.GetInstance().Message("JOBQUEUE_TEST_ENTRY", fcodeunits.CurrentLanguage()))
	entry.Start_date_time = types.Now()
	entry.End_date_time = types.Now()
	if !entry.Insert(true) {
		if err := entry.TriggerError(); err != nil {
			return fcodeunits.Result{}, err
		}
		return fcodeunits.Result{}, errors.New("the log entry could not be added")
	}
	result := system.Message(i18n.GetInstance().Message("JOBQUEUE_TEST_ENTRY_ADDED", fcodeunits.CurrentLanguage()), next)
	result.Refresh = true
	return result, nil
}

// OnValidate_Notification_email checks the notification address list (one or more
// addresses separated by ";") and stores it tidied up: "a@x.no; b@x.no".
func (jobQueueLogic) OnValidate_Notification_email(rec *gtables.JobQueue) error {
	items, err := mail.ParseAddressList(rec.Notification_email.String())
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
	rec.Notification_email = types.NewText(mail.JoinAddressList(items))
	return ftables.CheckMaxLength(gtables.JobQueueTableName, "notification_email", string(rec.Notification_email), 250)
}

// OnValidate_Notification_language checks that the notification language exists
func (jobQueueLogic) OnValidate_Notification_language(rec *gtables.JobQueue) error {
	if rec.Notification_language.IsEmpty() {
		return nil
	}
	var language gtables.Language
	language.InitWithDBType(rec.GetDB(), rec.GetCompany(), rec.GetDBType())
	if !language.Get(rec.Notification_language) {
		return apperrors.RelatedNotFound(gtables.JobQueueTableName, "notification_language", rec.Notification_language.String())
	}
	return nil
}
