package main

import (
	"strings"
	"testing"
)

func TestValidateFieldProperties(t *testing.T) {
	no := false
	def := func(fields ...Field) *TableDef {
		d := &TableDef{}
		d.Table.Name = "Currency"
		d.Table.Fields = append([]Field{{Name: "code", Type: "types.Code", PrimaryKey: true}}, fields...)
		return d
	}
	ok := def(
		Field{Name: "rate", Type: "types.Decimal", DecimalPlaces: "2:5", Validation: &Validation{Min: 0}},
		Field{Name: "percent", Type: "types.Decimal", Validation: &Validation{Min: 0, Max: 100}},
		Field{Name: "days", Type: "int", Validation: &Validation{Max: 31}},
		Field{Name: "last_date_modified", Type: "types.Date", Editable: &no},
		Field{Name: "description", Type: "types.Text"},
	)
	ok.Table.DataCaptionFields = []string{"code", "description"}
	if err := validateFieldProperties(ok); err != nil {
		t.Errorf("valid properties refused: %v", err)
	}

	for name, bad := range map[string]Field{
		"no bounds":            {Name: "rate", Type: "types.Decimal", Validation: &Validation{}},
		"min above max":        {Name: "rate", Type: "types.Decimal", Validation: &Validation{Min: 5, Max: 1}},
		"range on text":        {Name: "rate", Type: "types.Text", Validation: &Validation{Min: 0}},
		"places on int":        {Name: "rate", Type: "int", DecimalPlaces: "2:2"},
		"places not min:max":   {Name: "rate", Type: "types.Decimal", DecimalPlaces: "2"},
		"places min above max": {Name: "rate", Type: "types.Decimal", DecimalPlaces: "5:2"},
	} {
		if err := validateFieldProperties(def(bad)); err == nil || !strings.Contains(err.Error(), "rate") {
			t.Errorf("%s: err = %v, want an error naming the field", name, err)
		}
	}

	// A read-only key field gets its value from a filter or link (Exchange Rates: the currency)
	key := def()
	key.Table.Fields[0].Editable = &no
	if err := validateFieldProperties(key); err != nil {
		t.Errorf("a primary key with editable: false refused: %v", err)
	}
	for _, fields := range [][]string{{"missing"}, {"balance"}} {
		d := def(Field{Name: "balance", Type: "types.Decimal", FlowField: true})
		d.Table.DataCaptionFields = fields
		if err := validateFieldProperties(d); err == nil {
			t.Errorf("data_caption_fields %v accepted", fields)
		}
	}
}
