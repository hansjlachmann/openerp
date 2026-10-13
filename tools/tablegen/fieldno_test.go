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
