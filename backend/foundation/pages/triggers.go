package pages

import (
	"fmt"
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

// Control is the page as a trigger sees it: it switches fields for the current record
type Control struct {
	states map[string]FieldState
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
	triggersMu.RLock()
	triggers := afterGetRecord[pageID]
	triggersMu.RUnlock()
	page := &Control{}
	for _, trigger := range triggers {
		if err := trigger(rec, page); err != nil {
			return nil, err
		}
	}
	return page.States(), nil
}
