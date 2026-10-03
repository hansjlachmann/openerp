package tables

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// autoFillCustomer is a test wrapper whose OnValidate_No fills a sibling field,
// the way a BC trigger fills Account Name when Account No. is validated.
type autoFillCustomer struct {
	gtables.CustomerBase
}

func (t *autoFillCustomer) OnValidate_No() error {
	t.Name = types.NewText("Filled from " + t.No.String())
	return nil
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1) // every connection to :memory: is a separate database
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestInitRecordAppliesYAMLDefaults(t *testing.T) {
	pt := &PaymentTerms{}
	pt.InitWithDBType(nil, "TEST", database.DBTypeSQLite)
	pt.Active = false
	pt.InitRecord()
	if !pt.Active {
		t.Errorf("PaymentTerms.Active = false after InitRecord, want YAML default true")
	}

	smtp := &SMTPSetup{}
	smtp.InitWithDBType(nil, "TEST", database.DBTypeSQLite)
	smtp.Smtp_server_port = 0
	smtp.InitRecord()
	if smtp.Smtp_server_port != 587 {
		t.Errorf("SMTPSetup.Smtp_server_port = %d after InitRecord, want YAML default 587", smtp.Smtp_server_port)
	}
}

// The API holds tables as ftables.Table and calls InitWithDBType, so wrapper
// OnValidate_* overrides must be reached through that path.
func TestValidateFieldDispatchesToWrapperOverride(t *testing.T) {
	factory, ok := GetTableFactory("Customer")
	if !ok {
		t.Fatal("Customer table not registered")
	}
	var table ftables.Table = factory()
	table.InitWithDBType(nil, "TEST", database.DBTypeSQLite)

	// Customer.OnValidate_Name rejects names shorter than 3 characters
	if err := table.ValidateField("name", "ab"); err == nil {
		t.Error("ValidateField(name, \"ab\") = nil, want Customer.OnValidate_Name error")
	}
	if err := table.ValidateField("name", "Abc"); err != nil {
		t.Errorf("ValidateField(name, \"Abc\") = %v, want nil", err)
	}
}

func TestOnValidateCanFillSiblingFields(t *testing.T) {
	c := &autoFillCustomer{}
	c.InitWithDBType(nil, "TEST", database.DBTypeSQLite)
	c.SetSelf(c)

	if err := c.ValidateField("no", "C1"); err != nil {
		t.Fatalf("ValidateField: %v", err)
	}
	if got := c.ToMap()["name"]; got != "Filled from C1" {
		t.Errorf("ToMap()[name] = %v, want %q", got, "Filled from C1")
	}
}

// Insert through the interface (as the API does) must run the wrapper's OnInsert.
func TestInsertThroughInterfaceRunsOnInsert(t *testing.T) {
	db := newTestDB(t)
	if err := (&Customer{}).CreateTableWithDBType(db, "TEST", database.DBTypeSQLite); err != nil {
		t.Fatalf("create table: %v", err)
	}

	factory, _ := GetTableFactory("Customer")
	table := factory()
	table.InitWithDBType(db, "TEST", database.DBTypeSQLite)
	table.FromMap(map[string]interface{}{"no": "C1", "status": float64(gtables.Customer_Status.Blocked)})
	if !table.Insert(true) {
		t.Fatal("Insert failed")
	}

	// Customer.OnInsert forces Status to Open
	if got := table.ToMap()["status"]; got != int(gtables.Customer_Status.Open) {
		t.Errorf("status after Insert = %v, want Open (%d)", got, gtables.Customer_Status.Open)
	}
}

// Changing a primary key field of a loaded record must rename it (BC/NAV Rename), not
// silently do nothing. The list page relies on this: a User_Member row is inserted with a
// blank company as soon as user and role are filled, and the company is set afterwards.
func TestModifyRenamesChangedPrimaryKey(t *testing.T) {
	db := newTestDB(t)
	if err := (&UserMember{}).CreateTableWithDBType(db, "", database.DBTypeSQLite); err != nil {
		t.Fatalf("create table: %v", err)
	}

	m := &UserMember{}
	m.InitWithDBType(db, "", database.DBTypeSQLite)
	m.User_id, m.Role_id = types.NewCode("HANS"), types.NewCode("READER")
	if !m.Insert(true) {
		t.Fatal("Insert failed")
	}

	loaded := &UserMember{}
	loaded.InitWithDBType(db, "", database.DBTypeSQLite)
	if !loaded.GetByPK(types.NewCode("HANS"), types.NewCode("READER"), types.NewText("")) {
		t.Fatal("inserted record not found")
	}
	if err := loaded.ValidateField("company", "TEST-COMPANY"); err != nil {
		t.Fatalf("ValidateField: %v", err)
	}
	if !loaded.Modify(true) {
		t.Fatal("Modify failed")
	}

	probe := &UserMember{}
	probe.InitWithDBType(db, "", database.DBTypeSQLite)
	if probe.GetByPK(types.NewCode("HANS"), types.NewCode("READER"), types.NewText("")) {
		t.Error("record with blank company still exists after rename (would grant all companies)")
	}
	if !probe.GetByPK(types.NewCode("HANS"), types.NewCode("READER"), types.NewText("TEST-COMPANY")) {
		t.Error("renamed record not found under the new key")
	}
	if n := probe.Count(); n != 1 {
		t.Errorf("Count() = %d after rename, want 1", n)
	}

	// A second Modify on the same instance must match the renamed key
	if err := loaded.ValidateField("role_id", "WRITER"); err != nil {
		t.Fatalf("ValidateField: %v", err)
	}
	if !loaded.Modify(true) {
		t.Fatal("second Modify failed")
	}
	if !probe.GetByPK(types.NewCode("HANS"), types.NewCode("WRITER"), types.NewText("TEST-COMPANY")) {
		t.Error("second rename did not match the already-renamed key")
	}
}

// A failing trigger's error is kept for the caller (the API shows it to the user), and
// cleared by the next Insert/Modify/Delete.
func TestTriggerErrorIsReported(t *testing.T) {
	db := newTestDB(t)
	if err := (&Customer{}).CreateTableWithDBType(db, "TEST", database.DBTypeSQLite); err != nil {
		t.Fatalf("create table: %v", err)
	}

	c := &Customer{}
	c.InitWithDBType(db, "TEST", database.DBTypeSQLite)
	c.No = types.NewCode("C1")
	c.Name = types.NewText(strings.Repeat("x", 60)) // Customer.Validate: max 50
	if c.Insert(true) {
		t.Fatal("Insert succeeded, want OnInsert to reject a 60-character name")
	}
	if err := c.TriggerError(); err == nil || !strings.Contains(err.Error(), "name cannot exceed 50") {
		t.Errorf("TriggerError() = %v, want the OnInsert validation message", err)
	}

	c.Name = types.NewText("Acme")
	if !c.Insert(true) {
		t.Fatal("Insert with a valid name failed")
	}
	if err := c.TriggerError(); err != nil {
		t.Errorf("TriggerError() after a successful Insert = %v, want nil", err)
	}
}

// Deleting a user removes their preferences, which are stored without a company
// (handlers/preferences.go), whatever company the deleting session is in.
func TestUserDeleteRemovesTheirPreferences(t *testing.T) {
	db := newTestDB(t)
	if err := (&User{}).CreateTableWithDBType(db, "", database.DBTypeSQLite); err != nil {
		t.Fatalf("create user table: %v", err)
	}
	if err := (&UserPreferences{}).CreateTableWithDBType(db, "", database.DBTypeSQLite); err != nil {
		t.Fatalf("create preferences table: %v", err)
	}

	for _, id := range []string{"HANS", "OTHER"} {
		u := &User{}
		u.InitWithDBType(db, "", database.DBTypeSQLite)
		u.User_id = types.NewCode(id)
		if !u.Insert(false) {
			t.Fatalf("insert user %s failed", id)
		}
		p := &UserPreferences{}
		p.InitWithDBType(db, "", database.DBTypeSQLite)
		p.User_id, p.Page_id = types.NewCode(id), 22
		p.Preference_type, p.Preference_name = types.NewCode("COLUMNS"), types.NewCode("DEFAULT")
		if !p.Insert(false) {
			t.Fatalf("insert preference for %s failed", id)
		}
	}

	// Delete from a session in company CRONUS: preferences still live under company ""
	u := &User{}
	u.InitWithDBType(db, "CRONUS", database.DBTypeSQLite)
	if !u.Get("HANS") {
		t.Fatal("user HANS not found")
	}
	if !u.Delete(true) {
		t.Fatalf("Delete failed: %v", u.TriggerError())
	}

	prefs := &UserPreferences{}
	prefs.InitWithDBType(db, "", database.DBTypeSQLite)
	prefs.SetRange("user_id", "HANS")
	if n := prefs.Count(); n != 0 {
		t.Errorf("HANS has %d preferences after delete, want 0", n)
	}
	prefs.ClearFilters()
	prefs.SetRange("user_id", "OTHER")
	if n := prefs.Count(); n != 1 {
		t.Errorf("OTHER has %d preferences, want 1 (must be untouched)", n)
	}
}
