package handlers

import (
	"slices"
	"testing"
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
