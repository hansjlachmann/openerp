package tables

import (
	"errors"

	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/mail"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// jobQueueLogic is the business logic of table 472 Job_Queue. The notification addresses
// are checked by OnValidate_Notification_email when a user changes them, not on Modify: a
// list saved before that check must not stop the scheduler from rescheduling the job (it
// sends to the valid addresses and logs the others).
type jobQueueLogic struct{}

func init() { gtables.RegisterJobQueueLogic(jobQueueLogic{}) }

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
