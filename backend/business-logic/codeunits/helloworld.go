package codeunits

import (
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	fcodeunits "github.com/hansjlachmann/openerp/backend/foundation/codeunits"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
)

// HelloWorld - Codeunit 50020: Hello World Test
const HelloWorldID = 50020
const HelloWorldName = "hello-world"

func init() {
	Register(HelloWorldID, HelloWorldName, NewHelloWorld)
}

type HelloWorld struct {
	db      database.Executor
	company string
	dbType  database.DBType
}

// NewHelloWorld creates a new instance of the codeunit
func NewHelloWorld(db database.Executor, company string, dbType database.DBType) fcodeunits.Codeunit {
	return &HelloWorld{
		db:      db,
		company: company,
		dbType:  dbType,
	}
}

// ID returns the codeunit ID
func (c *HelloWorld) ID() int {
	return HelloWorldID
}

// Name returns the codeunit name
func (c *HelloWorld) Name() string {
	return HelloWorldName
}

// SourceTable returns the table this codeunit operates on
func (c *HelloWorld) SourceTable() string {
	return "Job_Queue"
}

// UsesProgress returns false - HelloWorld shows a message dialog, not progress
func (c *HelloWorld) UsesProgress() bool {
	return false
}

// Run executes the codeunit with the given record
func (c *HelloWorld) Run(record interface{}) (fcodeunits.Result, error) {
	var LogQueueEntries tables.JobQueueEntry
	jobQueue := record.(*tables.JobQueue)

	LogQueueEntries.InitWithDBType(c.db, c.company, c.dbType)

	nextEntryNo := 1
	if LogQueueEntries.FindLast() {
		nextEntryNo = LogQueueEntries.Entry_no + 1
	}
	LogQueueEntries.Entry_no = nextEntryNo
	LogQueueEntries.Status = 0
	LogQueueEntries.User_id = types.NewCode(fcodeunits.CurrentUserID())
	LogQueueEntries.Description = jobQueue.Description
	LogQueueEntries.Job_queue_no = jobQueue.No
	LogQueueEntries.Start_date_time = types.Now()
	LogQueueEntries.End_date_time = types.Now()
	if !LogQueueEntries.Insert(true) {
		return fcodeunits.Message("failed to insert ledger entry"), nil
	}

	return fcodeunits.Message("Hello World!"), nil
}
