package pages

import (
	"fmt"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	"strings"
	"sync"
)

// Page triggers: a page's business logic (business-logic/pages) switches fields per record —
// OnAfterGetRecord runs for the record a card shows and sets fields editable or visible:
//
//	pages.OnAfterGetRecord(674, func(rec *gtables.JobQueue, page *pages.Control) {
//		page.SetEditable("minutes_between_run", rec.Recurrence == gtables.JobQueue_Recurrence.Minutes)
//	})
//
// The API returns the field states with the record (POST /api/pages/:id/field-states) and the
// card applies them; the page definition says whether a page has such a trigger.

// FieldState is what a page trigger set for a field (nil: as in the page definition)
type FieldState struct {
	Editable *bool `json:"editable,omitempty"`
	Visible  *bool `json:"visible,omitempty"`
}

// Control is the page as a trigger sees it: it switches fields for the current record and
// fills its calculated fields (page fields with calculated: true, classic page variables)
type Control struct {
	states map[string]FieldState
	values map[string]interface{}
}

// SetValue fills a calculated page field for the current record (e.g. the No. Series list's
// Last No. Used of the series' current line)
func (c *Control) SetValue(field string, value interface{}) {
	if c.values == nil {
		c.values = map[string]interface{}{}
	}
	// as the records carry their fields (ToMap): dates, codes, texts and decimals as text — a
	// blank date ""
	switch v := value.(type) {
	case types.Date:
		value = v.String()
	case types.DateTime:
		value = v.String()
	case types.Code:
		value = v.String()
	case types.Text:
		value = v.String()
	case types.Decimal:
		value = v.String()
	}
	c.values[strings.ToLower(field)] = value
}

// Values returns the calculated fields the triggers filled, by field name
func (c *Control) Values() map[string]interface{} {
	if c.values == nil {
		return map[string]interface{}{}
	}
	return c.values
}

// SetEditable makes a field editable or read-only for the current record
func (c *Control) SetEditable(field string, editable bool) {
	s := c.state(field)
	s.Editable = &editable
	c.states[strings.ToLower(field)] = s
}

// SetVisible shows or hides a field for the current record
func (c *Control) SetVisible(field string, visible bool) {
	s := c.state(field)
	s.Visible = &visible
	c.states[strings.ToLower(field)] = s
}

func (c *Control) state(field string) FieldState {
	if c.states == nil {
		c.states = map[string]FieldState{}
	}
	return c.states[strings.ToLower(field)]
}

// States returns the field states the triggers set, by field name
func (c *Control) States() map[string]FieldState {
	if c.states == nil {
		return map[string]FieldState{}
	}
	return c.states
}

var (
	triggersMu     sync.RWMutex
	afterGetRecord = map[int][]func(rec any, page *Control) error{}
	newRecord      = map[int][]func(rec, xRec any, belowXRec bool) error{}
)

// OnAfterGetRecord registers a trigger of a page that runs for each record the page shows
// (OnAfterGetRecord); rec is the page's source table record (*gtables.<Table>). Several
// triggers of a page run in registration order.
func OnAfterGetRecord[T any](pageID int, fn func(rec T, page *Control)) {
	triggersMu.Lock()
	defer triggersMu.Unlock()
	afterGetRecord[pageID] = append(afterGetRecord[pageID], func(rec any, page *Control) error {
		r, ok := rec.(T)
		if !ok {
			var want T
			return fmt.Errorf("page %d: OnAfterGetRecord takes %T, the page's record is %T", pageID, want, rec)
		}
		fn(r, page)
		return nil
	})
}

// HasRecordTriggers reports whether a page has OnAfterGetRecord triggers
func HasRecordTriggers(pageID int) bool {
	triggersMu.RLock()
	defer triggersMu.RUnlock()
	return len(afterGetRecord[pageID]) > 0
}

// RunAfterGetRecord runs a page's OnAfterGetRecord triggers for a record and returns the
// field states they set
func RunAfterGetRecord(pageID int, rec any) (map[string]FieldState, error) {
	page, err := RunAfterGetRecordControl(pageID, rec)
	if err != nil {
		return nil, err
	}
	return page.States(), nil
}

// RunAfterGetRecordControl runs a page's OnAfterGetRecord triggers for a record and returns
// the page as they left it (field states and calculated values)
func RunAfterGetRecordControl(pageID int, rec any) (*Control, error) {
	triggersMu.RLock()
	triggers := afterGetRecord[pageID]
	triggersMu.RUnlock()
	page := &Control{}
	for _, trigger := range triggers {
		if err := trigger(rec, page); err != nil {
			return nil, err
		}
	}
	return page, nil
}

// OnNewRecord registers a trigger of a page that runs when the user starts a new record on it
// (OnNewRecord): rec is the new record (with the table's initial values), xRec the record the
// new one is inserted next to — below it when belowXRec, above it otherwise; an empty record
// when the page has none. Used for values taken from the neighbour (SetupNewGLAcc).
func OnNewRecord[T any](pageID int, fn func(rec, xRec T, belowXRec bool)) {
	triggersMu.Lock()
	defer triggersMu.Unlock()
	newRecord[pageID] = append(newRecord[pageID], func(rec, xRec any, below bool) error {
		r, ok1 := rec.(T)
		x, ok2 := xRec.(T)
		if !ok1 || !ok2 {
			var want T
			return fmt.Errorf("page %d: OnNewRecord takes %T, the page's record is %T", pageID, want, rec)
		}
		fn(r, x, below)
		return nil
	})
}

// HasNewRecordTriggers reports whether a page has OnNewRecord triggers
func HasNewRecordTriggers(pageID int) bool {
	triggersMu.RLock()
	defer triggersMu.RUnlock()
	return len(newRecord[pageID]) > 0
}

// RunNewRecord runs a page's OnNewRecord triggers on a new record
func RunNewRecord(pageID int, rec, xRec any, belowXRec bool) error {
	triggersMu.RLock()
	triggers := newRecord[pageID]
	triggersMu.RUnlock()
	for _, trigger := range triggers {
		if err := trigger(rec, xRec, belowXRec); err != nil {
			return err
		}
	}
	return nil
}
