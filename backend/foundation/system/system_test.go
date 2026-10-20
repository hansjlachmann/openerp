package system

import (
	"testing"
	"time"

	"github.com/hansjlachmann/openerp/backend/foundation/types"
)

func date(s string) types.Date { return types.MustDate(s) }

func TestCalcDate(t *testing.T) {
	sat := date("2026-01-31") // a Saturday
	cases := []struct {
		formula string
		from    types.Date
		want    string
	}{
		{"<1M>", sat, "2026-02-28"}, // February keeps its last day
		{"<-1D>", sat, "2026-01-30"},
		{"<2W>", sat, "2026-02-14"},
		{"<1Q>", sat, "2026-04-30"},
		{"<1Y>", date("2024-02-29"), "2025-02-28"},
		{"<CD>", sat, "2026-01-31"},
		{"<CM>", date("2026-01-15"), "2026-01-31"},
		{"<-CM>", sat, "2026-01-01"},
		{"<1M+CM>", sat, "2026-02-28"},
		{"<CM+1D>", sat, "2026-02-01"},
		{"<CQ>", sat, "2026-03-31"},
		{"<-CQ>", date("2026-05-20"), "2026-04-01"},
		{"<CY>", sat, "2026-12-31"},
		{"<-CY>", sat, "2026-01-01"},
		{"<CW>", sat, "2026-02-01"},  // Sunday of the ISO week
		{"<-CW>", sat, "2026-01-26"}, // Monday
		{"<WD1>", sat, "2026-02-02"}, // next Monday
		{"<WD6>", sat, "2026-01-31"}, // already a Saturday
		{"1m+cm", sat, "2026-02-28"}, // lower case, without < >
		{"<-1Y-CY>", sat, "2025-01-01"},
	}
	for _, c := range cases {
		got, err := CalcDate(c.formula, c.from)
		if err != nil || got.String() != c.want {
			t.Errorf("CalcDate(%q, %s) = %s, %v; want %s", c.formula, c.from, got, err, c.want)
		}
	}
	for _, bad := range []string{"", "<>", "<1X>", "<C>", "<M>", "<WD8>", "<-WD1>", "<1M+>"} {
		if _, err := CalcDate(bad, sat); err == nil {
			t.Errorf("CalcDate(%q) accepted", bad)
		}
	}
	if got, err := CalcDate("<1M>", types.Date{}); err != nil || !got.IsZero() {
		t.Errorf("blank date: %v, %v", got, err)
	}
	if got, _ := CalcDate("<0D>"); !got.Equal(Today()) {
		t.Errorf("default date is the work date (today): %v", got)
	}
}

func TestDMY2DateAndDate2DMY(t *testing.T) {
	d, err := DMY2Date(29, 2, 2024)
	if err != nil || d.String() != "2024-02-29" {
		t.Errorf("DMY2Date(29, 2, 2024) = %v, %v", d, err)
	}
	for _, bad := range [][3]int{{31, 2, 2026}, {0, 1, 2026}, {1, 13, 2026}} {
		if _, err := DMY2Date(bad[0], bad[1], bad[2]); err == nil {
			t.Errorf("DMY2Date%v accepted", bad)
		}
	}
	if d, _ := DMY2Date(1); d.Month() != time.Now().Month() || d.Year() != time.Now().Year() {
		t.Errorf("DMY2Date(1) = %v, want this month", d)
	}
	x := date("2026-10-20")
	if Date2DMY(x, 1) != 20 || Date2DMY(x, 2) != 10 || Date2DMY(x, 3) != 2026 || Date2DMY(x, 4) != 0 {
		t.Error("Date2DMY")
	}
}

type status int

func (s status) String() string { return []string{" ", "Open", "Blocked"}[s] }

func TestFormatAndEvaluate(t *testing.T) {
	cases := []struct {
		value any
		want  string
	}{
		{nil, ""},
		{"x", "x"},
		{types.Code("C1"), "C1"},
		{date("2026-10-20"), "2026-10-20"},
		{types.Date{}, ""},
		{types.MustDecimal("1234.50"), "1234.5"},
		{true, "true"},
		{42, "42"},
		{int64(-7), "-7"},
		{1.5, "1.5"},
		{status(1), "Open"},
	}
	for _, c := range cases {
		if got := Format(c.value); got != c.want {
			t.Errorf("Format(%#v) = %q, want %q", c.value, got, c.want)
		}
	}
	if got := Format("ab", 4); got != "ab  " {
		t.Errorf("left-aligned %q", got)
	}
	if got := Format(42, -5); got != "   42" {
		t.Errorf("right-aligned %q", got)
	}
	if got := Format("abcdef", 3); got != "abc" {
		t.Errorf("cut %q", got)
	}

	var d types.Date
	if !Evaluate(&d, "2026-10-20") || d.String() != "2026-10-20" {
		t.Errorf("date %v", d)
	}
	if Evaluate(&d, "20.10.26") || d.String() != "2026-10-20" {
		t.Error("an invalid date must return false and keep the value")
	}
	var dec types.Decimal
	if !Evaluate(&dec, "1234.5") || dec.String() != "1234.5" || Evaluate(&dec, "1.234,5") {
		t.Errorf("decimal %v", dec)
	}
	var code types.Code
	if !Evaluate(&code, " c-01 ") || code != "C-01" {
		t.Errorf("code %q", code)
	}
	var n int
	if !Evaluate(&n, "42") || n != 42 || Evaluate(&n, "x") {
		t.Errorf("int %d", n)
	}
	var s status
	if !Evaluate(&s, "2") || s != 2 {
		t.Errorf("option %d", s)
	}
	var b bool
	if !Evaluate(&b, "true") || !b || Evaluate(&b, "Yes") {
		t.Error("bool")
	}
	if Evaluate(nil, "1") {
		t.Error("nil target accepted")
	}
}

func TestTextFunctions(t *testing.T) {
	if got := StrSubstNo("%1 of %2", 3, "C1"); got != "3 of C1" {
		t.Errorf("StrSubstNo %q", got)
	}
	if got := StrSubstNo("No. #1#### is %2.", "C1"); got != "No. C1 is ." {
		t.Errorf("StrSubstNo # and a missing value %q", got)
	}
	if got := StrSubstNo("100% sure, %1", date("2026-10-20")); got != "100% sure, 2026-10-20" {
		t.Errorf("StrSubstNo %% %q", got)
	}
	if CopyStr("Ærlig talt", 1, 5) != "Ærlig" || CopyStr("abc", 2) != "bc" || CopyStr("abc", 9) != "" || CopyStr("abc", 2, 0) != "" {
		t.Error("CopyStr")
	}
	if StrLen("Ærø") != 3 || StrPos("aéb", "b") != 3 || StrPos("abc", "x") != 0 || StrPos("abc", "") != 0 {
		t.Error("StrLen/StrPos")
	}
	if PadStr("ab", 4) != "ab  " || PadStr("ab", 4, "*") != "ab**" || PadStr("abcdef", 3) != "abc" || PadStr("7", -3, "0") != "007" {
		t.Error("PadStr")
	}
	if DelChr("  a b  ") != "ab" || DelChr("  a b  ", "<>") != "a b" || DelChr("--a-b", "<", "-") != "a-b" || DelChr("C1, C2", "=", ", ") != "C1C2" {
		t.Error("DelChr")
	}
	if SelectStr(2, "a,b,c") != "b" || SelectStr(4, "a,b,c") != "" || SelectStr(0, "a") != "" {
		t.Error("SelectStr")
	}
	for in, want := range map[string]string{"INV0099": "INV0100", "A9": "A10", "X9Y": "X10Y", "ABC": "", "": "", "0": "1"} {
		if got := IncStr(in); got != want {
			t.Errorf("IncStr(%q) = %q, want %q", in, got, want)
		}
	}
	if UpperCase("abc") != "ABC" || LowerCase("ÆØÅ") != "æøå" {
		t.Error("UpperCase/LowerCase")
	}
}

func TestNumberFunctions(t *testing.T) {
	dec := types.MustDecimal
	cases := []struct {
		got  types.Decimal
		want string
	}{
		{Round(dec("1.235")), "1.24"},
		{Round(dec("-1.235")), "-1.24"}, // half away from zero
		{Round(dec("1.231"), 0.01, ">"), "1.24"},
		{Round(dec("1.239"), 0.01, "<"), "1.23"},
		{Round(dec("-1.231"), 0.01, "<"), "-1.24"},
		{Round(dec("12.5"), 1), "13"},
		{Round(dec("1250"), 100), "1300"},
		{Round(dec("1234.567"), dec("0.05")), "1234.55"},
		{Abs(dec("-3.5")), "3.5"},
		{Power(dec("2"), dec("10")), "1024"},
		{Power(dec("4"), dec("0.5")), "2"},
	}
	for i, c := range cases {
		if c.got.String() != c.want {
			t.Errorf("case %d: %s, want %s", i, c.got, c.want)
		}
	}
}

func TestDialog(t *testing.T) {
	if err := Error("Customer %1 is blocked", "C1"); err == nil || err.Error() != "Customer C1 is blocked" {
		t.Errorf("Error: %v", err)
	}
	if r := Message("%1 records", 3); r.Dialog == nil || r.Dialog.Message != "3 records" || !r.Success {
		t.Errorf("Message: %+v", r)
	}
}
