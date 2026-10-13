package main

import (
	"strings"
	"testing"
)

func TestValidateFieldNumbers(t *testing.T) {
	table := func(fields ...Field) *TableDef {
		d := &TableDef{}
		d.Table.Name = "Customer"
		d.Table.Fields = fields
		return d
	}
	if err := validateFieldNumbers(table(Field{No: 1, Name: "no"}, Field{No: 2, Name: "name"},
		Field{No: 55, Name: "date_filter", FlowFilter: true}, Field{No: 50000, Name: "loyalty_points"})); err != nil {
		t.Errorf("valid numbers: %v", err)
	}
	cases := map[string]struct {
		def  *TableDef
		want string
	}{
		"missing":   {table(Field{No: 1, Name: "no"}, Field{Name: "name"}), "field name has no field number"},
		"duplicate": {table(Field{No: 1, Name: "no"}, Field{No: 1, Name: "name"}), "fields no and name have the same number 1"},
		"negative":  {table(Field{No: -3, Name: "no"}), "number -3 outside"},
		"too high":  {table(Field{No: 100000, Name: "no"}), "number 100000 outside"},
	}
	for name, c := range cases {
		if err := validateFieldNumbers(c.def); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want %q", name, err, c.want)
		}
	}
}

func TestValidateNumberSeries(t *testing.T) {
	app := &application{}
	app.Application.Name = "Base"
	app.Application.NumberSeries.Objects = []numberRange{{1, 999}}
	app.Application.NumberSeries.Fields = []numberRange{{1, 999}}
	app.Application.SystemTables = []numberRange{{2000000000, 2000000199}}
	table := func(id int, fields ...Field) *TableDef {
		d := &TableDef{}
		d.Table.ID, d.Table.Name, d.Table.Fields = id, "T", fields
		return d
	}
	ok := []*TableDef{
		table(18, Field{No: 1, Name: "no"}),
		table(2000000002, Field{No: 999, Name: "x"}),
		table(18, Field{No: 1, Name: "no"}, Field{No: 50000, Name: "loyalty", App: "11111111-1111-4111-8111-111111111111"}),
	}
	for _, d := range ok {
		if err := validateNumberSeries(d, app); err != nil {
			t.Errorf("table %d: %v", d.Table.ID, err)
		}
	}
	bad := map[string]*TableDef{
		"table above the series": table(5100, Field{No: 1, Name: "no"}),
		"system range too high":  table(2000000200, Field{No: 1, Name: "no"}),
		"base field 1000":        table(18, Field{No: 1000, Name: "x"}),
	}
	for name, d := range bad {
		if err := validateNumberSeries(d, app); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
