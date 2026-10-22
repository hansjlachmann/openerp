package main

import (
	"strings"
	"testing"
)

func TestValidateRelationConditions(t *testing.T) {
	def := func(rel TableRelation, typ string) *TableDef {
		d := &TableDef{}
		d.Table.Name = "G_L_Entry"
		d.Table.Fields = []Field{
			{Name: "bal_account_type", Type: typ},
			{Name: "bal_account_no", Type: "types.Code", TableRelation: &rel},
		}
		return d
	}
	branch := func(value int, table string) RelationCondition {
		c := RelationCondition{TableRelation: TableRelation{Table: table, Field: "no"}}
		c.When.Field, c.When.Equal = "bal_account_type", value
		return c
	}
	ok := TableRelation{Conditions: []RelationCondition{branch(0, "G_L_Account"), branch(1, "Customer")}}
	if err := validateRelationConditions(def(ok, "Option")); err != nil {
		t.Errorf("valid conditions refused: %v", err)
	}
	for name, c := range map[string]struct {
		rel TableRelation
		typ string
	}{
		"table and conditions": {TableRelation{Table: "Customer", Conditions: ok.Conditions}, "Option"},
		"condition on a text":  {ok, "types.Text"},
		"two for one value":    {TableRelation{Conditions: []RelationCondition{branch(1, "Customer"), branch(1, "G_L_Account")}}, "Option"},
		"no table":             {TableRelation{Conditions: []RelationCondition{branch(0, "")}}, "Option"},
	} {
		if err := validateRelationConditions(def(c.rel, c.typ)); err == nil || !strings.Contains(err.Error(), "bal_account_no") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}
