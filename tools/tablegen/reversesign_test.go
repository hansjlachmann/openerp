package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// reverse_sign: a Sum FlowField with the opposite sign (classic -Sum); only for Sum FlowFields
func TestReverseSignFlowField(t *testing.T) {
	entry := &TableDef{}
	entry.Table.ID, entry.Table.Name = 21, "Cust Entry"
	entry.Table.Fields = []Field{
		{No: 1, Name: "entry_no", Type: "int", DBName: "entry_no", PrimaryKey: true},
		{No: 3, Name: "customer_no", Type: "types.Code", DBName: "customer_no", Length: 20},
		{No: 13, Name: "amount", Type: "types.Decimal", DBName: "amount"},
	}
	cust := &TableDef{}
	cust.Table.ID, cust.Table.Name = 18, "Cust"
	cust.Table.Fields = []Field{
		{No: 1, Name: "no", Type: "types.Code", DBName: "no", PrimaryKey: true, Length: 20},
		{No: 69, Name: "payments", Type: "types.Decimal", FlowField: true, CalcFormula: "Sum", ReverseSign: true,
			SourceTable: "CustEntry", SourceField: "amount", FlowFilters: []FlowFilter{{Field: "customer_no", Type: "field", Value: "no"}}},
	}
	byStruct := map[string]*TableDef{"CustEntry": entry, "Cust": cust}
	splitFlowFilterFields(cust)
	if err := validateFlowFilters(cust, byStruct); err != nil {
		t.Fatal(err)
	}
	resolveFlowFieldSIFT(cust, byStruct)
	out := filepath.Join(t.TempDir(), "cust_gen.go")
	if err := generateFile(out, "gen", boilerplateTemplate, prepareTemplateData(cust)); err != nil {
		t.Fatal(err)
	}
	code, _ := os.ReadFile(out)
	if n := strings.Count(string(code), "-COALESCE(SUM(amount), 0)"); n < 3 {
		t.Errorf("the reversed sum appears %d times, want the card, list and filter queries", n)
	}

	cust.Table.Fields[1].CalcFormula = "Count"
	if err := validateFlowFilters(cust, byStruct); err == nil {
		t.Error("reverse_sign on a Count FlowField accepted")
	}
}
