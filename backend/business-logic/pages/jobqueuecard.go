// Package pages holds the business logic of pages: triggers that switch fields per record
// (pages.OnAfterGetRecord). One file per page, named after it.
package pages

import (
	"github.com/hansjlachmann/openerp/backend/foundation/pages"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// Page 674 Job Queue Card
func init() {
	pages.OnAfterGetRecord(674, func(rec *gtables.JobQueue, page *pages.Control) {
		// The minutes only count for a job that recurs every few minutes
		page.SetEditable("minutes_between_run", rec.Recurrence == gtables.JobQueue_Recurrence.Minutes)
		// The notification fields only count when a notification is sent
		notify := rec.Notify_on != gtables.JobQueue_Notify_on.Never
		page.SetEditable("notification_email", notify)
		page.SetEditable("notification_language", notify)
	})
}
