package pages

import (
	"github.com/hansjlachmann/openerp/backend/business-logic/resjnl"
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/pages"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// The resource journal page 207 Resource Journal: OnOpenPage opens a template and one of its
// batches (ResJnlManagement); a new line gets its values from the line above (SetUpNewLine);
// the footer shows the line's resource name.
const resJournalPageID = 207

func init() {
	pages.OnOpenPage(resJournalPageID, func(rec *gtables.ResJournalLine, page *pages.Open) error {
		return resjnl.OpenJournalPage(rec, page)
	})
	pages.OnNewRecord(resJournalPageID, func(rec, xRec *gtables.ResJournalLine, belowXRec bool) {
		_ = tables.SetUpNewResJnlLine(rec, xRec)
	})
	pages.OnAfterGetRecord(resJournalPageID, func(rec *gtables.ResJournalLine, page *pages.Control) {
		page.SetValue("resource_name", resjnl.GetRes(rec))
	})

	// Page 272 Resource Jnl. Batches: a new batch takes the template's numbering and reason code
	// (SetupNewBatch)
	pages.OnNewRecord(272, func(rec, xRec *gtables.ResJournalBatch, belowXRec bool) {
		_ = tables.SetupNewResJnlBatch(rec)
	})
}
