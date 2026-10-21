// Package functions holds the functions pages can call with their current record: page
// actions and menu items with run_object: <type>:<object>.<function>, e.g.
//
//	run_object: table:Job_Queue_Entry.ShowLineNo
//	run_object: codeunit:900.PostLine
//
// Business logic registers each callable function explicitly (only registered functions can be
// called through the API), with the record type it takes:
//
//	functions.Register("table", gtables.JobQueueEntryTableName, "ShowLineNo",
//		functions.For(jobQueueEntryLogic{}.ShowLineNo))
//
//	func (jobQueueEntryLogic) ShowLineNo(rec *gtables.JobQueueEntry) (codeunits.Result, error)
//
// The API reads the record from the database by the key the page sends and calls the function
// like a codeunit; its Result (a message, an error) is shown as a codeunit's.
package functions

import (
	"fmt"
	"strings"
	"sync"

	"github.com/hansjlachmann/openerp/backend/foundation/codeunits"
)

// Func is a registered function: it gets the page's current record (a generated record)
type Func func(rec interface{}) (codeunits.Result, error)

// For makes a Func of a function on a typed record (*gtables.<Table>)
func For[T any](fn func(rec T) (codeunits.Result, error)) Func {
	return func(rec interface{}) (codeunits.Result, error) {
		r, ok := rec.(T)
		if !ok {
			var want T
			return codeunits.Result{}, fmt.Errorf("the function takes %T, the page's record is %T", want, rec)
		}
		return fn(r)
	}
}

// Function is a registered function with the table of the record it takes
type Function struct {
	ObjectType string // "table" or "codeunit"
	Object     string // the table's name, or the codeunit's number
	Name       string
	Table      string // the record's table (for a table function: the table itself)
	// Dialog: run with a codeunit dialog (Confirm, StrMenu, progress) — async
	Dialog bool
	Fn     Func
}

// Option changes how a function is registered
type Option func(*Function)

// OnTable says which table's record a codeunit function takes (a table function takes its own)
func OnTable(table string) Option { return func(f *Function) { f.Table = table } }

// WithDialog runs the function with a codeunit dialog, so it can ask (Confirm, StrMenu) and
// show progress
func WithDialog() Option { return func(f *Function) { f.Dialog = true } }

var (
	mu       sync.RWMutex
	registry = map[string]*Function{}
)

func key(objectType, object, name string) string {
	return strings.ToLower(objectType + ":" + object + "." + name)
}

// Register adds a function of a table ("table", table name) or a codeunit ("codeunit", its
// number as text; give the record's table with OnTable)
func Register(objectType, object, name string, fn Func, opts ...Option) {
	f := &Function{ObjectType: strings.ToLower(objectType), Object: object, Name: name, Fn: fn}
	if f.ObjectType == "table" {
		f.Table = object
	}
	for _, o := range opts {
		o(f)
	}
	mu.Lock()
	defer mu.Unlock()
	registry[key(f.ObjectType, object, name)] = f
}

// Lookup finds a function by its run_object text ("table:Job_Queue_Entry.ShowLineNo")
func Lookup(runObject string) (*Function, bool) {
	objectType, rest, ok := strings.Cut(runObject, ":")
	if !ok {
		return nil, false
	}
	dot := strings.LastIndex(rest, ".")
	if dot <= 0 || dot == len(rest)-1 {
		return nil, false
	}
	mu.RLock()
	defer mu.RUnlock()
	f, ok := registry[key(objectType, rest[:dot], rest[dot+1:])]
	return f, ok
}

// All returns the registered functions
func All() []*Function {
	mu.RLock()
	defer mu.RUnlock()
	all := make([]*Function, 0, len(registry))
	for _, f := range registry {
		all = append(all, f)
	}
	return all
}

// Codeunit runs the function like a codeunit (the job API's way of running page objects):
// on the record of its table, async with a dialog when it was registered WithDialog
func (f *Function) Codeunit() codeunits.Codeunit { return functionCodeunit{f} }

type functionCodeunit struct{ f *Function }

func (c functionCodeunit) ID() int                                          { return 0 }
func (c functionCodeunit) Name() string                                     { return c.f.ObjectType + ":" + c.f.Object + "." + c.f.Name }
func (c functionCodeunit) SourceTable() string                              { return c.f.Table }
func (c functionCodeunit) UsesProgress() bool                               { return c.f.Dialog }
func (c functionCodeunit) Run(record interface{}) (codeunits.Result, error) { return c.f.Fn(record) }
