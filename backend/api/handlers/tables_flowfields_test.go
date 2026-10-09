package handlers

import (
	"database/sql"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
)

func TestFlowFieldsToCalc(t *testing.T) {
	flow := []string{"balance_lcy", "sales_lcy", "no_of_ledger_entries"}
	tests := []struct {
		name      string
		requested []string
		want      []string
	}{
		{"no field list: all", nil, flow},
		{"only requested", []string{"no", "name", "balance_lcy", "sales_lcy"}, []string{"balance_lcy", "sales_lcy"}},
		{"none requested", []string{"no", "name"}, nil},
	}
	for _, tt := range tests {
		if got := flowFieldsToCalc(flow, tt.requested); !slices.Equal(got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

// Lists filter on Sum/Count FlowFields (BC: SETFILTER on a FlowField): the filter applies to
// the calculated value, also with a FlowFilter (Date Filter) set.
func TestListFilterOnFlowFields(t *testing.T) {
	app := newTablesTestApp(t)
	db, err := sql.Open("sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	var cle tables.CustomerLedgerEntry
	if err := cle.CreateTableWithDBType(db, "TEST", database.DBTypeSQLite); err != nil {
		t.Fatal(err)
	}
	for _, no := range []string{"C01", "C02", "C03"} {
		postJSON(t, app, "/api/tables/Customer/insert", `{"no":"`+no+`","name":"Customer `+no+`"}`)
	}
	entry := 0
	add := func(customer, date string, remaining string, open bool) {
		t.Helper()
		entry++
		var e tables.CustomerLedgerEntry
		e.InitWithDBType(db, "TEST", database.DBTypeSQLite)
		e.Entry_no = entry
		e.Customer_no = types.NewCode(customer)
		e.Posting_date = types.MustDate(date)
		e.Remaining_amt_lcy = types.MustDecimal(remaining)
		e.Open = open
		if !e.Insert(false) {
			t.Fatalf("insert entry %d", entry)
		}
	}
	add("C01", "2026-01-10", "100", true)
	add("C01", "2026-02-10", "50.5", true)
	add("C01", "2026-03-10", "0", false) // closed: not in the balance, counted as an entry
	add("C02", "2026-04-10", "20000", true)
	// C03 has no entries: balance 0, no entries

	list := func(filters, flowFilters string) (int, string, int) {
		t.Helper()
		target := "/api/tables/Customer/list?filters=" + url.QueryEscape(filters)
		if flowFilters != "" {
			target += "&flow_filters=" + url.QueryEscape(flowFilters)
		}
		status, out := getJSON(t, app, target)
		nos, total := listNos(t, out)
		return status, fmt.Sprint(nos), total
	}
	for _, tc := range []struct{ filters, want string }{
		{`[{"field":"balance_lcy","expression":">10000"}]`, "[C02]"},
		{`[{"field":"balance_lcy","expression":"0"}]`, "[C03]"},
		{`[{"field":"balance_lcy","expression":"<>0"}]`, "[C01 C02]"},
		{`[{"field":"balance_lcy","expression":"150,5"}]`, "[C01]"}, // decimal comma
		{`[{"field":"balance_lcy","expression":"100..200"}]`, "[C01]"},
		{`[{"field":"balance_lcy","expression":"0|>10000"}]`, "[C02 C03]"},
		{`[{"field":"no_of_ledger_entries","expression":"3.."}]`, "[C01]"},
		{`[{"field":"no_of_ledger_entries","expression":"0"},{"field":"name","expression":"*C03"}]`, "[C03]"},
	} {
		status, nos, total := list(tc.filters, "")
		if status != 200 || nos != tc.want || total != len(strings.Fields(strings.Trim(tc.want, "[]"))) {
			t.Errorf("%s = %d %s total %d, want %s", tc.filters, status, nos, total, tc.want)
		}
	}

	// With a Date Filter the FlowField only sums the entries in the period
	status, nos, _ := list(`[{"field":"balance_lcy","expression":">=100"}]`, `[{"field":"date_filter","expression":"2026-01-01..2026-01-31"}]`)
	if status != 200 || nos != "[C01]" {
		t.Errorf("balance >=100 in January = %d %s, want [C01]", status, nos)
	}
	status, nos, _ = list(`[{"field":"balance_lcy","expression":">=150"}]`, `[{"field":"date_filter","expression":"2026-01-01..2026-01-31"}]`)
	if status != 200 || nos != "[]" {
		t.Errorf("balance >=150 in January = %d %s, want none (only 100 in January)", status, nos)
	}

	// An invalid filter is reported with the field's caption
	status, out := getJSON(t, app, "/api/tables/Customer/list?filters="+url.QueryEscape(`[{"field":"balance_lcy","expression":"abc"}]`))
	if msg, _ := out["error"].(string); status != 400 || !strings.Contains(msg, "'abc' is not a valid filter") {
		t.Errorf("invalid FlowField filter = %d %v", status, out)
	}
}
