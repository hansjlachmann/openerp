package codeunits

import (
	fcodeunits "github.com/hansjlachmann/openerp/backend/foundation/codeunits"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/system"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
	"time"
)

// ProgressDemo - Codeunit 991: Progress Bar Demo
const ProgressDemoID = 991
const ProgressDemoName = "progress-demo"

func init() {
	Register(ProgressDemoID, ProgressDemoName, NewProgressDemo)
}

type ProgressDemo struct {
	db      database.Executor
	company string
	dbType  database.DBType
}

// NewProgressDemo creates a new instance of the codeunit
func NewProgressDemo(db database.Executor, company string, dbType database.DBType) fcodeunits.Codeunit {
	return &ProgressDemo{
		db:      db,
		company: company,
		dbType:  dbType,
	}
}

// ID returns the codeunit ID
func (c *ProgressDemo) ID() int {
	return ProgressDemoID
}

// Name returns the codeunit name
func (c *ProgressDemo) Name() string {
	return ProgressDemoName
}

// SourceTable returns the table this codeunit operates on
func (c *ProgressDemo) SourceTable() string {
	return "Job_Queue"
}

// UsesProgress returns true - this codeunit uses progress dialog updates
func (c *ProgressDemo) UsesProgress() bool {
	return true
}

// Run executes the codeunit with progress updates
func (c *ProgressDemo) Run(record interface{}) (fcodeunits.Result, error) {
	// Let the user choose the speed (StrMenu), or cancel
	speed := system.StrMenu("Fast,Normal,Slow", 2, "Start progress demo: choose the speed")
	if speed == 0 {
		return fcodeunits.Message("Progress demo cancelled."), nil
	}
	stepTime := []time.Duration{100, 500, 1500}[speed-1] * time.Millisecond

	jobQueue := record.(*gtables.JobQueue)

	// Get the current dialog (set by job handler)
	dialog := fcodeunits.GetCurrentDialog()

	// Simulate a long-running process with progress updates
	totalSteps := 10
	for i := 1; i <= totalSteps; i++ {
		// Calculate progress percentage
		progress := (i * 100) / totalSteps

		// Update progress if dialog is available
		if dialog != nil {
			dialog.UpdateWithMessage(1, progress, getProgressMessage(i))
		}

		// Simulate work
		time.Sleep(stepTime)
	}

	if err := CreateJobQueueEntry(c.db, c.company, c.dbType, jobQueue, gtables.JobQueueEntry_Status.Success, "", types.Now()); err != nil {
		return fcodeunits.Message(err.Error()), nil
	}

	return fcodeunits.Message("Processing completed successfully!"), nil
}

// getProgressMessage returns a message for each step
func getProgressMessage(step int) string {
	messages := []string{
		"Initializing...",
		"Loading data...",
		"Processing records...",
		"Validating entries...",
		"Updating database...",
		"Calculating totals...",
		"Generating reports...",
		"Finalizing changes...",
		"Cleaning up...",
		"Completing...",
	}

	if step > 0 && step <= len(messages) {
		return messages[step-1]
	}
	return "Processing..."
}
