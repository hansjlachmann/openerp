package handlers

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v2"

	_ "github.com/hansjlachmann/openerp/backend/business-logic/pages"
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/pages"
	"github.com/hansjlachmann/openerp/backend/foundation/session"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// The Job Queue card's OnAfterGetRecord switches fields per record; the page says it has
// such a trigger
func TestPageFieldStates(t *testing.T) {
	if err := pages.GetRegistry().LoadPages(); err != nil {
		t.Skipf("pages: %v", err)
	}
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session", &session.Session{Company: "TEST", Language: "en-US"})
		return c.Next()
	})
	app.Post("/api/pages/:id/field-states", NewPageTriggersHandler(db, database.DBTypeSQLite).FieldStates)

	states := func(body string) map[string]interface{} {
		t.Helper()
		out := postJSON(t, app, "/api/pages/674/field-states", body)
		data, _ := out["data"].(map[string]interface{})
		s, _ := data["field_states"].(map[string]interface{})
		if s == nil {
			t.Fatalf("field states: %v", out)
		}
		return s
	}
	editable := func(s map[string]interface{}, field string) interface{} {
		f, _ := s[field].(map[string]interface{})
		return f["editable"]
	}
	s := states(`{"no":"JQ1","recurrence":1,"notify_on":0}`) // Minutes, Never
	if editable(s, "minutes_between_run") != true || editable(s, "notification_email") != false || editable(s, "notification_language") != false {
		t.Errorf("Minutes, Never: %v", s)
	}
	s = states(`{"no":"JQ1","recurrence":3,"notify_on":2}`) // Daily, On Error
	if editable(s, "minutes_between_run") != false || editable(s, "notification_email") != true {
		t.Errorf("Daily, On Error: %v", s)
	}
	if !pages.HasRecordTriggers(674) || pages.HasRecordTriggers(22) {
		t.Error("HasRecordTriggers")
	}

	// A trigger for the wrong record type is an error, not a panic
	pages.OnAfterGetRecord(99991, func(rec *gtables.Customer, page *pages.Control) { page.SetVisible("name", false) })
	if _, err := pages.RunAfterGetRecord(99991, &gtables.JobQueue{}); err == nil {
		t.Error("wrong record type accepted")
	}
	if st, err := pages.RunAfterGetRecord(99991, &gtables.Customer{}); err != nil || *st["name"].Visible {
		t.Errorf("SetVisible: %v %v", st, err)
	}
}

// A journal line entered on its page gets its read-only fields from the page's OnNewRecord
// (the template's Source Code), not from the payload — the browser cannot send them
func TestInsertTakesReadOnlyFieldsFromNewRecord(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err := tables.CreateCompanyTables(db, "TEST", database.DBTypeSQLite); err != nil {
		t.Fatal(err)
	}
	src := ftables.DBSource(db, "TEST", database.DBTypeSQLite)
	template := gtables.NewResJournalTemplate(src)
	template.Name, template.Page_id, template.Source_code = "RESOURCES", 207, "RESJNL"
	batch := gtables.NewResJournalBatch(src)
	batch.Journal_template_name, batch.Name = "RESOURCES", "DEFAULT"
	for _, err := range []error{template.InsertOrError(false), batch.InsertOrError(false)} {
		if err != nil {
			t.Fatal(err)
		}
	}
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session", &session.Session{Company: "TEST", Language: "en-US"})
		return c.Next()
	})
	app.Post("/api/tables/:table/insert", NewTablesHandler(db).InsertRecord)

	line := `{"journal_template_name":"RESOURCES","journal_batch_name":"DEFAULT","line_no":%d,"source_code":"HACK"}`
	status, out := sendJSON(t, app, "POST", "/api/tables/Res_Journal_Line/insert?page_id=207", fmt.Sprintf(line, 10000))
	if status != 200 || out["data"].(map[string]interface{})["source_code"] != "RESJNL" {
		t.Errorf("insert on page 207: %d %v", status, out)
	}
	// without the page the read-only field stays blank (only business logic sets it)
	status, out = sendJSON(t, app, "POST", "/api/tables/Res_Journal_Line/insert", fmt.Sprintf(line, 20000))
	if status != 200 || out["data"].(map[string]interface{})["source_code"] != "" {
		t.Errorf("insert without a page: %d %v", status, out)
	}
}
