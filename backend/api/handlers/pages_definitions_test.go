package handlers

import (
	"path/filepath"
	"testing"

	"github.com/hansjlachmann/openerp/backend/foundation/pages"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// Every page definition loads; its source table exists, its fields are fields of that table
// (or calculated or virtual masked page fields), and the pages it opens (card, list, drilldowns, actions) exist
func TestPageDefinitions(t *testing.T) {
	registry := pages.GetRegistry()
	if err := registry.LoadPages(); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob("../../definitions/pages/*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	all := registry.GetAllPages()
	if len(all) != len(files) {
		t.Errorf("%d page files, %d pages loaded (a file that does not load is only a startup warning)", len(files), len(all))
	}
	exists := func(id int) bool {
		_, ok := all[id]
		return ok
	}
	for id, p := range all {
		factory, ok := gtables.GetTableFactory(p.Page.SourceTable)
		if !ok {
			t.Errorf("page %d: source table %q does not exist", id, p.Page.SourceTable)
			continue
		}
		rec := factory()
		fields := map[string]bool{}
		for _, f := range rec.GetFields() {
			fields[f.Name] = true
		}
		for _, f := range rec.GetFlowFilterFields() {
			fields[f.Name] = true
		}
		var pageFields []pages.Field
		for _, s := range p.Page.Layout.Sections {
			pageFields = append(pageFields, s.Fields...)
		}
		if p.Page.Layout.Repeater != nil {
			pageFields = append(pageFields, p.Page.Layout.Repeater.Fields...)
		}
		for _, f := range pageFields {
			// a masked page field may be virtual (the User Card's password)
			if !f.Calculated && !f.Masked && !fields[f.Source] {
				t.Errorf("page %d %s: %q is not a field of %s", id, p.Page.Name, f.Source, p.Page.SourceTable)
			}
			if f.Drilldown != 0 && !exists(f.Drilldown) {
				t.Errorf("page %d: field %s drills down to page %d, which does not exist", id, f.Source, f.Drilldown)
			}
		}
		for _, other := range []int{p.Page.CardPageID, p.Page.ListPageID} {
			if other != 0 && !exists(other) {
				t.Errorf("page %d: card/list page %d does not exist", id, other)
			}
		}
		var check func(actions []pages.Action)
		check = func(actions []pages.Action) {
			for _, a := range actions {
				if a.RunPage != 0 && !exists(a.RunPage) {
					t.Errorf("page %d: action %s opens page %d, which does not exist", id, a.Name, a.RunPage)
				}
				check(a.Items)
			}
		}
		check(p.Page.Actions)
	}
}
