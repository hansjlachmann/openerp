package reports

import (
	"time"

	"github.com/hansjlachmann/openerp/backend/foundation/system"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// Report 675 Delete Job Queue Log Entries: deletes the log entries of job queues that are
// older than a number of days (a processing-only report)
const DeleteJobQueueLogID = 675

func init() {
	Register(DeleteJobQueueLogID, "delete-job-queue-log-entries", func() Report { return &deleteJobQueueLog{} })
}

type deleteJobQueueLog struct {
	before  types.DateTime
	window  *system.Window
	deleted int
}

func (r *deleteJobQueueLog) Definition() *Definition {
	return &Definition{
		OptionsCaption: "OPTIONS",
		Options:        []Option{{Name: "older_than_days", Type: "number", Default: "30", Required: true}},
		DataItems: []*DataItem{{
			Name:                "JobQueue",
			Table:               gtables.JobQueueTableName,
			Key:                 []string{"no"},
			RequestFilterFields: []string{"no"},
			DataItems: []*DataItem{{
				Name:                "LogEntry",
				Table:               gtables.JobQueueEntryTableName,
				Key:                 []string{"job_queue_no", "entry_no"},
				Link:                map[string]string{"job_queue_no": "no"},
				RequestFilterFields: []string{"status"},
				OnPreDataItem:       Trigger(r.onPreLogEntry),
				OnAfterGetRecord:    Trigger(r.deleteLogEntry),
			}},
		}},
		OnPreReport:  r.onPreReport,
		OnPostReport: r.onPostReport,
	}
}

func (r *deleteJobQueueLog) onPreReport(run *Run) error {
	days := run.OptionInt("older_than_days")
	if days < 0 {
		return system.Error(run.Text("DAYS_NEGATIVE"))
	}
	if system.GuiAllowed() && !system.Confirm(run.Text("CONFIRM", days)) {
		return Quit()
	}
	r.before = types.NewDateTimeFromTime(time.Now().AddDate(0, 0, -days))
	r.window = system.OpenWindow(run.Text("WINDOW"), "", 0)
	return nil
}

// Only entries started before the date (on top of the user's filters)
func (r *deleteJobQueueLog) onPreLogEntry(e *gtables.JobQueueEntry, run *Run) error {
	e.SetRange("start_date_time", nil, r.before)
	return nil
}

func (r *deleteJobQueueLog) deleteLogEntry(e *gtables.JobQueueEntry, run *Run) error {
	if e.Status == gtables.JobQueueEntry_Status.InProcess {
		return Skip() // still running
	}
	r.window.Update(1, e.Job_queue_no)
	if !e.Delete(false) {
		return Skip()
	}
	r.deleted++
	r.window.Update(2, r.deleted)
	return nil
}

func (r *deleteJobQueueLog) onPostReport(run *Run) error {
	r.window.Close()
	run.Message(run.Text("DONE", r.deleted))
	return nil
}
