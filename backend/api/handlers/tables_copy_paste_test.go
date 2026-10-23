package handlers

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
)

// List copy/paste: /copy returns the stored fields of the selected records, /insert-many
// inserts records in one transaction — the first one that fails rolls back all.
func TestCopyAndPasteRecords(t *testing.T) {
	app := newTablesTestApp(t)
	for i, name := range []string{"Anna", "Bjørn", "Carl"} {
		postJSON(t, app, "/api/tables/Customer/insert", fmt.Sprintf(`{"no":"C%02d","name":"%s","city":"Oslo"}`, i+1, name))
	}
	nos := func() string {
		t.Helper()
		_, out := getJSON(t, app, "/api/tables/Customer/list")
		n, _ := listNos(t, out)
		return fmt.Sprint(n)
	}

	// Copy: stored fields only (no FlowFields, no FlowFilter fields)
	out := postJSON(t, app, "/api/tables/Customer/copy", `{"keys":["C01","C02"]}`)
	records, _ := out["data"].(map[string]interface{})["records"].([]interface{})
	if len(records) != 2 {
		t.Fatalf("copy = %v", out)
	}
	first := records[0].(map[string]interface{})
	if first["no"] != "C01" || first["name"] != "Anna" || first["city"] != "Oslo" {
		t.Errorf("copied record = %v", first)
	}
	for _, f := range []string{"balance_lcy", "no_of_ledger_entries", "date_filter"} {
		if _, ok := first[f]; ok {
			t.Errorf("copied record has %s (FlowField / FlowFilter)", f)
		}
	}
	// Copy all of a filtered list
	filters := url.QueryEscape(`[{"field":"name","expression":"*r*"}]`)
	out = postJSON(t, app, "/api/tables/Customer/copy?filters="+filters, `{"all":true}`)
	if recs, _ := out["data"].(map[string]interface{})["records"].([]interface{}); len(recs) != 2 {
		t.Errorf("copy all *r* = %d records, want 2 (Bjørn, Carl)", len(recs))
	}

	// Paste with new keys
	out = postJSON(t, app, "/api/tables/Customer/insert-many",
		`{"records":[{"no":"C11","name":"Anna","city":"Oslo"},{"no":"C12","name":"Bjørn","city":"Oslo"}]}`)
	if out["success"] != true || fmt.Sprint(out["data"].(map[string]interface{})["keys"]) != "[C11 C12]" || nos() != "[C01 C02 C03 C11 C12]" {
		t.Fatalf("paste = %v, list %s", out, nos())
	}

	// A record that already exists: nothing is pasted, the record is named
	out = postJSON(t, app, "/api/tables/Customer/insert-many",
		`{"records":[{"no":"C13","name":"New one"},{"no":"C01","name":"Anna"}]}`)
	msg, _ := out["error"].(string)
	if out["success"] != false || !strings.Contains(msg, "Record 2 of 2") || !strings.Contains(msg, "C01") || nos() != "[C01 C02 C03 C11 C12]" {
		t.Errorf("duplicate paste = %q, list %s (want nothing pasted)", msg, nos())
	}

	// A field check (OnValidate: the Gen. Bus. Posting Group must exist) fails: nothing is pasted
	out = postJSON(t, app, "/api/tables/Customer/insert-many",
		`{"records":[{"no":"C14","name":"Fine"},{"no":"C15","name":"ab","gen_bus_posting_group":"NONE"}]}`)
	if out["success"] != false || nos() != "[C01 C02 C03 C11 C12]" {
		t.Errorf("invalid paste = %v, list %s", out, nos())
	}

	// Companies are never pasted
	if out := postJSON(t, app, "/api/tables/Company/insert-many", `{"records":[{"name":"x"}]}`); out["success"] != false {
		t.Errorf("company paste = %v, want refused", out)
	}
}
