package tables

import (
	"strconv"

	"github.com/hansjlachmann/openerp/backend/foundation/system"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// DateFilter-Calc (codeunit 358): date filters of accounting periods and fiscal years for the
// statistics pages. Without accounting periods the filter and the name are blank.

// endOfTime is the last date a filter can reach (31.12.9999)
var endOfTime = types.NewDate(9999, 12, 31)

// CreateFiscalYearFilter returns the date filter and the name ("Fiscal Year 2026") of the
// fiscal year of a date, or of the fiscal year nextStep years after (or before) it
func CreateFiscalYearFilter(from ftables.Source, date types.Date, nextStep int, fiscalYearText string) (string, string) {
	return createAccountingDateFilter(from, true, date, nextStep, fiscalYearText)
}

// CreateAccountingPeriodFilter returns the date filter and the name of the accounting period of
// a date, or of the period nextStep periods after (or before) it
func CreateAccountingPeriodFilter(from ftables.Source, date types.Date, nextStep int) (string, string) {
	return createAccountingDateFilter(from, false, date, nextStep, "")
}

// createAccountingDateFilter: fiscalYearText is the name of a fiscal year ("Fiscal Year %1",
// translated)
func createAccountingDateFilter(from ftables.Source, fiscalYear bool, date types.Date, nextStep int, fiscalYearText string) (string, string) {
	accountingPeriod := gtables.NewAccountingPeriod(from)
	if fiscalYear {
		accountingPeriod.SetRange("new_fiscal_year", true)
	}
	accountingPeriod.Starting_date = date
	if !accountingPeriod.Find("=<>") {
		return "", ""
	}
	if accountingPeriod.Starting_date.After(date) {
		nextStep = nextStep - 1
	}
	if nextStep != 0 {
		if nextAccountingPeriod(accountingPeriod, nextStep) != nextStep {
			if nextStep < 0 {
				return ".." + accountingPeriod.Starting_date.AddDays(-1).String(), "..."
			}
			return accountingPeriod.Starting_date.String() + ".." + endOfTime.String(), "..."
		}
	}
	startDate := accountingPeriod.Starting_date
	var name string
	if fiscalYear {
		name = system.StrSubstNo(fiscalYearText, strconv.Itoa(startDate.Year()))
	} else {
		name = accountingPeriod.Name.String()
	}
	if nextAccountingPeriod(accountingPeriod, 1) != 0 {
		return startDate.String() + ".." + accountingPeriod.Starting_date.AddDays(-1).String(), name
	}
	return startDate.String() + ".." + endOfTime.String(), name + "..."
}

// nextAccountingPeriod moves steps periods forward (or back) within the filters and returns
// how many it moved (Next(steps))
func nextAccountingPeriod(accountingPeriod *gtables.AccountingPeriod, steps int) int {
	moved := 0
	for moved != steps {
		if steps > 0 {
			if !accountingPeriod.Find(">") {
				return moved
			}
			moved++
		} else {
			if !accountingPeriod.Find("<") {
				return moved
			}
			moved--
		}
	}
	return moved
}
