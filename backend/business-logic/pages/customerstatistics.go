package pages

import (
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	fcodeunits "github.com/hansjlachmann/openerp/backend/foundation/codeunits"
	"github.com/hansjlachmann/openerp/backend/foundation/i18n"
	"github.com/hansjlachmann/openerp/backend/foundation/pages"
	"github.com/hansjlachmann/openerp/backend/foundation/system"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// Page 151 Customer Statistics: the customer's balance and, for this accounting period, this
// fiscal year, last fiscal year and to date, its sales, profit, discounts and other amounts
// (OnAfterGetRecord; the periods of the work date, DateFilter-Calc)
var customerStatisticsColumns = []string{"this_period", "this_year", "last_year", "to_date"}

func init() {
	pages.OnAfterGetRecord(151, func(rec *gtables.Customer, page *pages.Control) {
		currentDate := system.WorkDate()
		fiscalYearText := i18n.GetInstance().Message("FISCAL_YEAR_NAME", fcodeunits.CurrentLanguage())
		var custDateFilter, custDateName [4]string
		custDateFilter[0], custDateName[0] = tables.CreateAccountingPeriodFilter(rec, currentDate, 0)
		custDateFilter[1], custDateName[1] = tables.CreateFiscalYearFilter(rec, currentDate, 0, fiscalYearText)
		custDateFilter[2], custDateName[2] = tables.CreateFiscalYearFilter(rec, currentDate, -1, fiscalYearText)

		cust := gtables.NewCustomer(rec)
		cust.No = rec.No
		_ = cust.SetFlowFilter("date_filter", ".."+currentDate.String())
		cust.CalcFields("balance_lcy", "balance_due_lcy")
		page.SetValue("total_lcy", cust.Balance_lcy)
		page.SetValue("balance_due_to_date_lcy", cust.Balance_due_lcy)

		for i, column := range customerStatisticsColumns {
			page.SetValue(column+"_name", custDateName[i])
			// without accounting periods only To Date has amounts
			custPeriod := gtables.NewCustomer(rec)
			custPeriod.No = rec.No
			if custDateFilter[i] != "" || column == "to_date" {
				_ = custPeriod.SetFlowFilter("date_filter", custDateFilter[i])
				custPeriod.CalcFields("sales_lcy", "profit_lcy", "inv_discounts_lcy", "pmt_discounts_lcy",
					"fin_charge_memo_amounts_lcy", "cr_memo_amounts_lcy", "payments_lcy", "reminder_amounts_lcy")
			}
			profitPct := types.ZeroDecimal()
			if !custPeriod.Sales_lcy.IsZero() {
				profitPct = system.Round(custPeriod.Profit_lcy.Mul(types.NewDecimal(100)).Div(custPeriod.Sales_lcy), types.MustDecimal("0.1"))
			}
			page.SetValue(column+"_sales_lcy", custPeriod.Sales_lcy)
			page.SetValue(column+"_profit_lcy", custPeriod.Profit_lcy)
			page.SetValue(column+"_profit_pct", profitPct)
			page.SetValue(column+"_inv_discounts_lcy", custPeriod.Inv_discounts_lcy)
			page.SetValue(column+"_pmt_discounts_lcy", custPeriod.Pmt_discounts_lcy)
			page.SetValue(column+"_fin_charge_memo_amounts_lcy", custPeriod.Fin_charge_memo_amounts_lcy)
			page.SetValue(column+"_cr_memo_amounts_lcy", custPeriod.Cr_memo_amounts_lcy)
			page.SetValue(column+"_payments_lcy", custPeriod.Payments_lcy)
			page.SetValue(column+"_reminder_amounts_lcy", custPeriod.Reminder_amounts_lcy)
		}
	})
}
