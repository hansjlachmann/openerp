package types

import "testing"

// A DATE column read into a string on PostgreSQL arrives as a timestamp: the date part counts
func TestNewDateFromStringPostgresDate(t *testing.T) {
	for in, want := range map[string]string{
		"2026-10-21":                "2026-10-21",
		"2026-10-21T00:00:00Z":      "2026-10-21",
		"2026-10-21T00:00:00+02:00": "2026-10-21",
		"C2026-12-31":               "C2026-12-31",
		"":                          "",
	} {
		d, err := NewDateFromString(in)
		if err != nil || d.String() != want {
			t.Errorf("%q: %q %v, want %q", in, d.String(), err, want)
		}
	}
	for _, in := range []string{"21.10.2026", "2026-13-01", "2026-10-21X00"} {
		if _, err := NewDateFromString(in); err == nil {
			t.Errorf("%q accepted", in)
		}
	}
}
