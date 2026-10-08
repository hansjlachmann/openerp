package tables

import (
	"testing"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
)

func TestJobQueueNotificationEmail(t *testing.T) {
	var jq JobQueue
	jq.InitWithDBType(nil, "TEST", database.DBTypeSQLite)

	if err := jq.ValidateField("notification_email", " ops@example.com;hans@example.com ; "); err != nil {
		t.Fatalf("valid list rejected: %v", err)
	}
	if got := jq.Notification_email.String(); got != "ops@example.com; hans@example.com" {
		t.Errorf("stored list = %q, want it tidied up", got)
	}
	if err := jq.ValidateField("notification_email", ""); err != nil {
		t.Errorf("blank list rejected: %v", err)
	}

	for _, tc := range []struct{ list, message string }{
		{"ops@example.com; ops@", "Notification E-mail: 'ops@' is not a valid e-mail address"},
		{"ops@example.com; OPS@example.com", "Notification E-mail: 'OPS@example.com' is listed more than once"},
	} {
		err := jq.ValidateField("notification_email", tc.list)
		if err == nil || err.Error() != tc.message {
			t.Errorf("%q: error %v, want %q", tc.list, err, tc.message)
		}
	}
}

func TestJobQueueNotificationLanguage(t *testing.T) {
	db := newTestDB(t)
	var language Language
	language.InitWithDBType(db, "", database.DBTypeSQLite)
	if err := language.CreateTableWithDBType(db, "", database.DBTypeSQLite); err != nil {
		t.Fatal(err)
	}
	language.Code = types.NewCode("NB-NO")
	language.Translation_key = types.NewText("nb-NO")
	if !language.Insert(true) {
		t.Fatal("insert language")
	}

	var jq JobQueue
	jq.InitWithDBType(db, "TEST", database.DBTypeSQLite)
	if err := jq.ValidateField("notification_language", "NB-NO"); err != nil {
		t.Errorf("existing language rejected: %v", err)
	}
	if err := jq.ValidateField("notification_language", "XX-XX"); err == nil {
		t.Error("unknown language accepted")
	}
}
