package apps

import (
	"strings"
	"testing"
)

func TestBaseApplication(t *testing.T) {
	app, err := LoadBase()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		objectType string
		id         int
		want       bool
	}{
		{Table, 18, true}, {Page, 999, true}, {Codeunit, 1000, false}, {Report, 0, false},
		{Table, 2000000002, true}, {Table, 2000000200, false},
		{Page, 2000000002, false}, {Codeunit, 2000000006, false}, // system range: tables only
	}
	for _, c := range cases {
		if got := app.ObjectAllowed(c.objectType, c.id); got != c.want {
			t.Errorf("%s %d allowed = %v, want %v", c.objectType, c.id, got, c.want)
		}
	}
	if !app.FieldAllowed(999) || app.FieldAllowed(1000) || app.FieldAllowed(0) {
		t.Error("base fields are 1–999")
	}
	if err := app.CheckObject(Codeunit, 50100, "Create Demo Data"); err == nil || !strings.Contains(err.Error(), "codeunit 50100 Create Demo Data is outside") {
		t.Errorf("CheckObject: %v", err)
	}
}

func TestParseRejects(t *testing.T) {
	base := `application:
  id: 760e20ec-7a66-4e3f-b6e5-83c94db5a4bd
  name: X
  number_series:
    objects: [{from: 1, to: 999}]
    fields: [{from: 1, to: 999}]
`
	if _, err := Parse([]byte(base)); err != nil {
		t.Fatalf("valid: %v", err)
	}
	cases := map[string]string{
		"bad id":        strings.Replace(base, "760e20ec-7a66-4e3f-b6e5-83c94db5a4bd", "abc", 1),
		"no name":       strings.Replace(base, "name: X", "name: ''", 1),
		"no fields":     strings.Replace(base, "    fields: [{from: 1, to: 999}]\n", "", 1),
		"reversed":      strings.Replace(base, "objects: [{from: 1, to: 999}]", "objects: [{from: 9, to: 1}]", 1),
		"overlap":       strings.Replace(base, "objects: [{from: 1, to: 999}]", "objects: [{from: 1, to: 999}, {from: 500, to: 1500}]", 1),
		"unknown field": base + "  colour: red\n",
	}
	for name, data := range cases {
		if _, err := Parse([]byte(data)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestCheckOverlap(t *testing.T) {
	app := func(id string, from, to int) *App {
		a := &App{ID: id, Name: id}
		a.NumberSeries.Objects = []Range{{from, to}}
		a.NumberSeries.Fields = []Range{{from, to}}
		return a
	}
	base := app("760e20ec-7a66-4e3f-b6e5-83c94db5a4bd", 1, 999)
	if err := CheckOverlap([]*App{base, app("11111111-1111-4111-8111-111111111111", 50000, 50999)}); err != nil {
		t.Errorf("separate series: %v", err)
	}
	if err := CheckOverlap([]*App{base, app("11111111-1111-4111-8111-111111111111", 900, 1999)}); err == nil {
		t.Error("overlapping series accepted")
	}
	if err := CheckOverlap([]*App{base, base}); err == nil {
		t.Error("same id twice accepted")
	}
}
