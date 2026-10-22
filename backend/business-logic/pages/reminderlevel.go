package pages

import (
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/pages"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// Page 432 Reminder Levels: a new level gets the next No. of its reminder terms (OnNewRecord →
// NewRecord)
func init() {
	pages.OnNewRecord(432, func(rec, xRec *gtables.ReminderLevel, belowXRec bool) {
		tables.NewReminderLevel(rec)
	})
}
