package pages

import (
	"github.com/hansjlachmann/openerp/backend/business-logic/noseries"
	"github.com/hansjlachmann/openerp/backend/foundation/pages"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// Page 456 No. Series shows each series' current line — the open line with the latest
// starting date on or before the work date (UpdateLine; blank when there is none)
func init() {
	pages.OnAfterGetRecord(456, func(rec *gtables.NoSeries, page *pages.Control) {
		line := gtables.NewNoSeriesLine(rec)
		noseries.SetNoSeriesLineFilter(line, rec.Code, types.Date{})
		if !line.FindFirst() {
			line.InitRecord()
		}
		page.SetValue("starting_date", line.Starting_date)
		page.SetValue("starting_no", line.Starting_no)
		page.SetValue("ending_no", line.Ending_no)
		page.SetValue("last_date_used", line.Last_date_used)
		page.SetValue("last_no_used", line.Last_no_used)
		page.SetValue("warning_no", line.Warning_no)
		page.SetValue("increment_by_no", line.Increment_by_no)
	})
}
