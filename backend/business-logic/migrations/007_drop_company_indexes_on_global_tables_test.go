package migrations

import "testing"

func TestIsCompanyCopy(t *testing.T) {
	for _, tc := range []struct {
		index, table string
		want         bool
	}{
		{"demo01$Company$Primary", "Company", true},
		{"ØYANS BIL$User_Member$Primary", "User_Member", true},
		{"Company$Primary", "Company", false}, // the shared index (new name)
		{"Company_pkey", "Company", false},    // primary key constraint
		{"User_Member_pkey", "User_Member", false},
		{"demo01$User$Primary", "User_Member", false}, // another table's name
		{"x$Company$", "Company", false},
	} {
		if got := isCompanyCopy(tc.index, tc.table); got != tc.want {
			t.Errorf("isCompanyCopy(%q, %q) = %v, want %v", tc.index, tc.table, got, tc.want)
		}
	}
}
