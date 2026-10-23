package pages

import (
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/pages"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// Page 124 Comment Sheet: the first comment of a record gets the work date (OnNewRecord →
// SetUpNewLine)
func init() {
	pages.OnNewRecord(124, func(rec, xRec *gtables.CommentLine, belowXRec bool) {
		tables.SetUpNewLine(rec)
	})
}
