package reports

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	fcodeunits "github.com/hansjlachmann/openerp/backend/foundation/codeunits"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/i18n"
	"github.com/hansjlachmann/openerp/backend/foundation/system"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// Run is a running report (CurrReport): its database, options and result
type Run struct {
	ID      int
	DB      database.Executor
	Company string
	DBType  database.DBType

	options map[string]string
	filters map[string][]ftables.FieldFilter // data item name → the user's filters
	message string
}

// Option returns an option of the request page as text
func (r *Run) Option(name string) string { return r.options[name] }

// OptionInt returns a number option (0 when blank or not a number)
func (r *Run) OptionInt(name string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(r.options[name]))
	return n
}

// OptionDate returns a date option (blank when blank or not a date)
func (r *Run) OptionDate(name string) types.Date {
	d, _ := types.NewDateFromString(strings.TrimSpace(r.options[name]))
	return d
}

// Message sets the message shown when the report is done (filled in with StrSubstNo; pass a
// translated text — Text gives the report's own)
func (r *Run) Message(text string, values ...any) {
	r.message = system.StrSubstNo(text, values...)
}

// Text returns a translated text of the report (reports.<id>.texts.<key>) in the user's
// language, filled in with StrSubstNo
func (r *Run) Text(key string, values ...any) string {
	return system.StrSubstNo(translate(r.ID, "texts."+key), values...)
}

// RunOptions say how a report runs
type RunOptions struct {
	// RequestPage: show the request page first (only with a dialog; a scheduled run never
	// shows it)
	RequestPage bool
	// Record: its filters (all filter groups) are the starting request filters of the data
	// items of its table (Report.Run with a record)
	Record ftables.Table
	// Options: starting values of the request page options
	Options map[string]string
}

// Result is how a report ended
type Result struct {
	Cancelled bool   // the user cancelled the request page: nothing ran
	Message   string // set by the report with Run.Message
}

// Execute runs a report (Report.Run): request page, OnPreReport, the data items, OnPostReport
func Execute(id int, db database.Executor, company string, dbType database.DBType, opts RunOptions) (Result, error) {
	factory, ok := Get(id)
	if !ok {
		return Result{}, fmt.Errorf("report %d does not exist", id)
	}
	def := factory().Definition()
	run := &Run{ID: id, DB: db, Company: company, DBType: dbType, options: map[string]string{}, filters: map[string][]ftables.FieldFilter{}}
	for _, o := range def.Options {
		run.options[o.Name] = o.Default
	}
	for name, value := range opts.Options {
		run.options[name] = value
	}
	if opts.Record != nil {
		source, ok := opts.Record.(interface{ FilterExpressions() []ftables.FieldFilter })
		if ok {
			for _, item := range allDataItems(def.DataItems) {
				if sameTable(item.Table, opts.Record) {
					run.filters[item.Name] = append(run.filters[item.Name], source.FilterExpressions()...)
				}
			}
		}
	}

	if opts.RequestPage && fcodeunits.GetCurrentDialog() != nil {
		if !requestPage(run, def) {
			return Result{Cancelled: true}, nil
		}
	}

	err := func() error {
		if def.OnPreReport != nil {
			err := def.OnPreReport(run)
			if err != nil {
				return err
			}
		}
		for _, item := range def.DataItems {
			err := runDataItem(run, item, nil)
			if err != nil {
				return err
			}
		}
		if def.OnPostReport != nil {
			return def.OnPostReport(run)
		}
		return nil
	}()
	if errors.Is(err, errQuit) {
		err = nil
	}
	return Result{Message: run.message}, err
}

// requestPage asks for the options and the data items' filters (any field, as on a list
// page; the request filter fields to start with); false when cancelled
func requestPage(run *Run, def *Definition) bool {
	language := fcodeunits.CurrentLanguage()
	var fields []fcodeunits.InputField
	for _, o := range def.Options {
		fields = append(fields, fcodeunits.InputField{
			Name:     "option:" + o.Name,
			Label:    translate(run.ID, "options."+o.Name),
			Type:     o.Type,
			Required: o.Required,
			Default:  run.options[o.Name],
			Options:  o.Options,
		})
	}
	var items []fcodeunits.RequestDataItem
	for _, entry := range indentedDataItems(def.DataItems, 0) {
		item := entry.item
		rec, ok := newRecord(run, item)
		if !ok {
			continue
		}
		requestItem := fcodeunits.RequestDataItem{Name: item.Name, Caption: rec.GetTableName(), Indent: entry.indent}
		c, ok := rec.(interface{ GetCaption(string) string })
		if ok {
			requestItem.Caption = c.GetCaption(language)
		}
		caption := func(field string) string {
			c, ok := rec.(interface{ GetFieldCaption(string, string) string })
			if ok {
				return c.GetFieldCaption(field, language)
			}
			return field
		}
		for _, f := range ftables.FilterableFields(rec) {
			requestItem.Fields = append(requestItem.Fields, fcodeunits.RequestField{Name: f.Name, Caption: caption(f.Name), Type: string(f.Type)})
		}
		// The request filter fields first (blank or with the starting filter), then the other
		// starting filters
		start := map[string]string{}
		for _, f := range run.filters[item.Name] {
			start[f.Field] = f.Expression
		}
		listed := map[string]bool{}
		for _, field := range item.RequestFilterFields {
			requestItem.Filters = append(requestItem.Filters, fcodeunits.RequestFilter{Field: field, Expression: start[field]})
			listed[field] = true
		}
		for _, f := range run.filters[item.Name] {
			if !listed[f.Field] {
				requestItem.Filters = append(requestItem.Filters, fcodeunits.RequestFilter{Field: f.Field, Expression: f.Expression})
			}
		}
		items = append(items, requestItem)
	}
	optionsCaption := ""
	if def.OptionsCaption != "" {
		optionsCaption = run.Text(def.OptionsCaption)
	}
	values := fcodeunits.RequestPage(translate(run.ID, "caption"), fields, items, optionsCaption)
	if len(values) == 0 {
		return false
	}
	for key, value := range values {
		name, ok := strings.CutPrefix(key, "option:")
		if ok {
			run.options[name] = value
		} else {
			name, ok := strings.CutPrefix(key, "filters:")
			if ok {
				var filters []ftables.FieldFilter
				err := json.Unmarshal([]byte(value), &filters)
				if err == nil {
					run.filters[name] = filters
				}
			}
		}
	}
	return true
}

// indented is a data item with its level (0 top level)
type indented struct {
	item   *DataItem
	indent int
}

// indentedDataItems returns the data items and their nested ones, in order, with their level
func indentedDataItems(items []*DataItem, level int) []indented {
	var all []indented
	for _, item := range items {
		all = append(all, indented{item, level})
		all = append(all, indentedDataItems(item.DataItems, level+1)...)
	}
	return all
}

// reportRecord: what the runner needs of a generated record
type reportRecord interface {
	ftables.Table
	SetRange(fieldName string, values ...interface{})
	FilterGroup(group ...int) int
	FindSetBuffered() bool
}

// newRecord makes a data item's record, initialized for the run
func newRecord(run *Run, item *DataItem) (reportRecord, bool) {
	factory, ok := gtables.GetTableFactory(item.Table)
	if !ok {
		return nil, false
	}
	rec, ok := factory().(reportRecord)
	if !ok {
		return nil, false
	}
	rec.InitWithDBType(run.DB, run.Company, run.DBType)
	return rec, true
}

// runDataItem runs a data item: filters, OnPreDataItem, the loop with OnAfterGetRecord and
// the nested data items, OnPostDataItem. parent is the parent data item's current record.
func runDataItem(run *Run, item *DataItem, parent ftables.Table) error {
	rec, ok := newRecord(run, item)
	if !ok {
		return fmt.Errorf("report %d: data item %s: table %q does not exist", run.ID, item.Name, item.Table)
	}
	if len(item.Key) > 0 {
		rec.SetCurrentKey(item.Key...)
	}
	rec.SetAscending(!item.Descending)
	// Fixed filters of the table view (group 2), the parent link (group 4), the user's
	// request filters (group 0)
	if len(item.TableView) > 0 {
		rec.FilterGroup(2)
		for _, f := range item.TableView {
			rec.SetFilter(f.Field, f.Expression)
		}
	}
	if parent != nil && len(item.Link) > 0 {
		values := parent.ToMap()
		rec.FilterGroup(4)
		for field, parentField := range item.Link {
			rec.SetRange(field, values[parentField])
		}
	}
	// The user's filters as on a list page (stored fields, Sum/Count FlowFields)
	rec.FilterGroup(0)
	for _, f := range run.filters[item.Name] {
		if strings.TrimSpace(f.Expression) == "" {
			continue
		}
		switch err := ftables.ApplyFilter(rec, ftables.FieldFilter{Field: f.Field, Expression: strings.TrimSpace(f.Expression)}); {
		case errors.Is(err, ftables.ErrFilterExpression):
			return apperrors.FilterInvalid(item.Table, f.Field, f.Expression)
		case err != nil:
			return apperrors.InvalidFilters()
		}
	}

	if item.OnPreDataItem != nil {
		err := item.OnPreDataItem(rec, run)
		if err != nil {
			if errors.Is(err, errBreak) {
				return nil
			}
			return err
		}
	}

	// SQLite reads all at once: its connection cannot read and write the table at once
	var found bool
	if run.DBType == database.DBTypeSQLite {
		found = rec.FindSetBuffered()
	} else {
		found = rec.FindSet()
	}
	for ; found; found = rec.Next() {
		if item.OnAfterGetRecord != nil {
			err := item.OnAfterGetRecord(rec, run)
			if errors.Is(err, errSkip) {
				continue
			}
			if errors.Is(err, errBreak) {
				break
			}
			if err != nil {
				return err
			}
		}
		for _, child := range item.DataItems {
			err := runDataItem(run, child, rec)
			if err != nil {
				return err
			}
		}
	}

	if item.OnPostDataItem != nil {
		err := item.OnPostDataItem(rec, run)
		if err != nil && !errors.Is(err, errBreak) {
			return err
		}
	}
	return nil
}

// allDataItems returns the data items and their nested ones, in order
func allDataItems(items []*DataItem) []*DataItem {
	var all []*DataItem
	for _, item := range items {
		all = append(all, item)
		all = append(all, allDataItems(item.DataItems)...)
	}
	return all
}

// sameTable reports whether a data item's table (API name) is the record's table
func sameTable(table string, rec ftables.Table) bool {
	factory, ok := gtables.GetTableFactory(table)
	return ok && factory().GetTableName() == rec.GetTableName()
}

// translate returns reports.<id>.<key> in the user's language
func translate(id int, key string) string {
	return i18n.GetInstance().Translate(fmt.Sprintf("reports.%d.%s", id, key), fcodeunits.CurrentLanguage())
}
