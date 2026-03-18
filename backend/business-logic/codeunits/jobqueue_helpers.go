package codeunits

import (
	"fmt"

	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	fcodeunits "github.com/hansjlachmann/openerp/backend/foundation/codeunits"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
)

// CreateJobQueueEntry creates a new Job Queue Entry record for the given job queue.
func CreateJobQueueEntry(db database.Executor, company string, dbType database.DBType, jobQueue *tables.JobQueue, status gtables.JobQueueEntryStatus, errorMsg string) error {
	var entry tables.JobQueueEntry
	entry.InitWithDBType(db, company, dbType)

	nextEntryNo := 1
	if entry.FindLast() {
		nextEntryNo = entry.Entry_no + 1
	}

	// Re-initialize to clear fields loaded by FindLast (e.g. previous error_message)
	entry.InitWithDBType(db, company, dbType)
	entry.Entry_no = nextEntryNo
	entry.Status = status
	entry.User_id = types.NewCode(fcodeunits.CurrentUserID())
	entry.Description = jobQueue.Description
	entry.Job_queue_no = jobQueue.No
	entry.Start_date_time = types.Now()
	entry.End_date_time = types.Now()
	if errorMsg != "" {
		entry.Error_message = types.NewText(errorMsg)
	}

	if !entry.Insert(true) {
		return fmt.Errorf("failed to insert job queue entry")
	}
	return nil
}

// SetJobQueueStatus updates the status of a Job Queue record.
func SetJobQueueStatus(db database.Executor, company string, dbType database.DBType, jobQueueNo types.Code, status gtables.JobQueueStatus) error {
	var jq tables.JobQueue
	jq.InitWithDBType(db, company, dbType)
	if !jq.Get(jobQueueNo.String()) {
		return fmt.Errorf("job queue %s not found", jobQueueNo)
	}
	jq.Status = status
	if !jq.Modify(true) {
		return fmt.Errorf("failed to update job queue status")
	}
	return nil
}
