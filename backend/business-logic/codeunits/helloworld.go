package codeunits

import (
	fcodeunits "github.com/hansjlachmann/openerp/backend/foundation/codeunits"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
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

// Run executes the codeunit with the given record
func (c *HelloWorld) Run(record interface{}) (fcodeunits.Result, error) {
	return fcodeunits.Result{
		Success: true,
		Message: "Hello World",
	}, nil
}
