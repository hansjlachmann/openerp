package flowfilter

import (
	"fmt"
	"testing"
)

func TestClause(t *testing.T) {
	tests := []struct {
		kind Kind
		expr string
		sql  string
		args string
	}{
		{KindDate, "", "", "[]"},
		{KindDate, "2026-01-01..2026-03-31", "posting_date BETWEEN ? AND ?", "[2026-01-01 2026-03-31]"},
		{KindDate, "..2026-03-31", "posting_date <= ?", "[2026-03-31]"},
		{KindDate, "2026-01-01..", "posting_date >= ?", "[2026-01-01]"},
		{KindDate, " 2026-05-17 ", "posting_date = ?", "[2026-05-17]"},
		{KindDate, "2026-01-01..2026-01-31|2026-03-01..2026-03-31", "(posting_date BETWEEN ? AND ? OR posting_date BETWEEN ? AND ?)", "[2026-01-01 2026-01-31 2026-03-01 2026-03-31]"},
		{KindDate, "<>2026-05-17", "posting_date <> ?", "[2026-05-17]"},
		{KindInt, "10..20|30", "(posting_date BETWEEN ? AND ? OR posting_date = ?)", "[10 20 30]"},
		{KindBool, "Yes", "posting_date = ?", "[true]"},
		{KindBool, "no", "posting_date = ?", "[false]"},
		{KindCode, "c0001*", `posting_date LIKE ? ESCAPE '\'`, "[C0001%]"},
		{KindCode, "c00010", "posting_date = ?", "[C00010]"},
		{KindText, "*oslo*", `posting_date LIKE ? ESCAPE '\'`, "[%oslo%]"},
		{KindText, "50%*", `posting_date LIKE ? ESCAPE '\'`, `[50\%%]`},
		{KindText, "A..M", "posting_date BETWEEN ? AND ?", "[A M]"},
	}
	for _, tt := range tests {
		sql, args, err := Clause("posting_date", tt.kind, tt.expr)
		if err != nil {
			t.Errorf("%s %q: %v", tt.kind, tt.expr, err)
			continue
		}
		if sql != tt.sql || fmt.Sprint(args) != tt.args {
			t.Errorf("%s %q = %q %v, want %q %s", tt.kind, tt.expr, sql, args, tt.sql, tt.args)
		}
	}
}

func TestClauseErrors(t *testing.T) {
	for _, tt := range []struct {
		kind Kind
		expr string
	}{
		{KindDate, "01.01.26"},       // not ISO (the frontend converts local dates)
		{KindDate, "2026-02-30"},     // no such day
		{KindDate, ".."},             // range without ends
		{KindDate, "2026-01-01|"},    // empty alternative
		{KindInt, "ten"},             // not a number
		{KindBool, "maybe"},          // not Yes/No
		{KindDate, "<>"},             // missing value
		{KindDate, "x'; DROP TABLE"}, // never reaches SQL text
	} {
		if err := Validate(tt.kind, tt.expr); err == nil {
			t.Errorf("%s %q: no error", tt.kind, tt.expr)
		}
	}
}
