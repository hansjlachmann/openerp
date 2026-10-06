package handlers

import (
	"fmt"
	"net/url"
	"testing"
)

// listNos returns the "no" values and total of a /list response.
func listNos(t *testing.T, out map[string]interface{}) ([]string, int) {
	t.Helper()
	data, _ := out["data"].(map[string]interface{})
	recs, _ := data["records"].([]interface{})
	var nos []string
	for _, r := range recs {
		nos = append(nos, r.(map[string]interface{})["no"].(string))
	}
	total, _ := data["total"].(float64)
	return nos, int(total)
}


// List pages load a window of rows (offset/limit) with server-side sort and search.
func TestListWindowSortAndSearch(t *testing.T) {
	app := newTablesTestApp(t)
	// Duplicate cities: sorting on city needs the primary key as tie-breaker
	for i, city := range []string{"Oslo", "Bergen", "Oslo", "Aarhus", "Oslo", "Bergen", "London"} {
		postJSON(t, app, "/api/tables/Customer/insert",
			fmt.Sprintf(`{"no":"C%02d","name":"Customer %d","city":"%s"}`, i+1, i+1, city))
	}

	// Window + total
	status, out := getJSON(t, app, "/api/tables/Customer/list?offset=2&limit=3")
	nos, total := listNos(t, out)
	if status != 200 || total != 7 || fmt.Sprint(nos) != "[C03 C04 C05]" {
		t.Errorf("offset=2&limit=3 = %d %v total %d", status, nos, total)
	}

	// Descending
	_, out = getJSON(t, app, "/api/tables/Customer/list?offset=0&limit=2&sort_order=desc")
	if nos, _ := listNos(t, out); fmt.Sprint(nos) != "[C07 C06]" {
		t.Errorf("desc = %v", nos)
	}

	// Consecutive windows on a non-unique key neither overlap nor skip rows
	seen := map[string]bool{}
	for offset := 0; offset < 7; offset += 2 {
		_, out = getJSON(t, app, fmt.Sprintf("/api/tables/Customer/list?sort_by=city&offset=%d&limit=2", offset))
		nos, _ := listNos(t, out)
		for _, no := range nos {
			if seen[no] {
				t.Errorf("%s appears in two windows", no)
			}
			seen[no] = true
		}
	}
	if len(seen) != 7 {
		t.Errorf("windows covered %d of 7 customers", len(seen))
	}

	// Search: case-insensitive "contains" over any of the given fields, counted in total
	fields := url.QueryEscape(`["no","name","city"]`)
	_, out = getJSON(t, app, "/api/tables/Customer/list?offset=0&limit=50&search=OSL&search_fields="+fields)
	if nos, total := listNos(t, out); total != 3 || fmt.Sprint(nos) != "[C01 C03 C05]" {
		t.Errorf("search OSL = %v total %d", nos, total)
	}
	_, out = getJSON(t, app, "/api/tables/Customer/list?search=c07&search_fields="+fields)
	if nos, _ := listNos(t, out); fmt.Sprint(nos) != "[C07]" {
		t.Errorf("search c07 = %v", nos)
	}
	// LIKE wildcards in the search text are literal
	_, out = getJSON(t, app, "/api/tables/Customer/list?search=%25&search_fields="+fields)
	if nos, _ := listNos(t, out); len(nos) != 0 {
		t.Errorf("search %% = %v, want none", nos)
	}

	// Unknown search field or sort order is rejected
	for _, target := range []string{
		"/api/tables/Customer/list?search=x&search_fields=" + url.QueryEscape(`["no; DROP TABLE x"]`),
		"/api/tables/Customer/list?sort_order=sideways",
	} {
		if status, _ := getJSON(t, app, target); status != 400 {
			t.Errorf("GET %s = %d, want 400", target, status)
		}
	}
}
