package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	_ "github.com/mattn/go-sqlite3"

	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/session"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// autoFillCustomer fills a sibling field from its OnValidate_No trigger.
type autoFillCustomer struct {
	gtables.CustomerBase
}

func (t *autoFillCustomer) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.CustomerBase.InitWithDBType(db, company, dbType)
	t.SetSelf(t)
}

func (t *autoFillCustomer) OnValidate_No() error {
	t.Name = types.NewText("Filled from " + t.No.String())
	return nil
}

func init() {
	tables.RegisterTableFactory("Test_AutoFill", 99001, func() ftables.Table {
		return &autoFillCustomer{}
	})
}

func newTablesTestApp(t *testing.T) *fiber.App {
	t.Helper()
	// Named shared-cache in-memory DB: handlers keep a result set open while running
	// a second query (e.g. Count), so more than one connection must see the same data
	db, err := sql.Open("sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := (&tables.Customer{}).CreateTableWithDBType(db, "TEST", database.DBTypeSQLite); err != nil {
		t.Fatalf("create table: %v", err)
	}

	h := NewTablesHandler(db)
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session", &session.Session{Company: "TEST", Language: "en-US"})
		return c.Next()
	})
	app.Post("/api/tables/:table/validate", h.ValidateField)
	app.Post("/api/tables/:table/init", h.InitRecord)
	app.Post("/api/tables/:table/insert", h.InsertRecord)
	app.Get("/api/tables/:table/list", h.ListRecords)
	app.Get("/api/tables/:table/ids", h.GetRecordIDs)
	return app
}

func postJSON(t *testing.T, app *fiber.App, url, body string) map[string]interface{} {
	t.Helper()
	req := httptest.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %s: %v (%s)", url, err, raw)
	}
	return out
}

func TestValidateFieldReturnsRecordWithSiblingsFilled(t *testing.T) {
	app := newTablesTestApp(t)

	out := postJSON(t, app, "/api/tables/Test_AutoFill/validate",
		`{"field":"no","value":"C1","record":{"no":"","name":"","city":"Oslo"}}`)

	if out["success"] != true {
		t.Fatalf("success = %v, error = %v", out["success"], out["error"])
	}
	data, _ := out["data"].(map[string]interface{})
	if data["name"] != "Filled from C1" {
		t.Errorf("data.name = %v, want trigger-filled %q", data["name"], "Filled from C1")
	}
	if data["no"] != "C1" {
		t.Errorf("data.no = %v, want validated value C1", data["no"])
	}
	if data["city"] != "Oslo" {
		t.Errorf("data.city = %v, want Oslo carried over from the in-progress record", data["city"])
	}
}

func TestValidateFieldReportsTriggerError(t *testing.T) {
	app := newTablesTestApp(t)

	// Customer.OnValidate_Name rejects names shorter than 3 characters
	out := postJSON(t, app, "/api/tables/Customer/validate", `{"field":"name","value":"ab","record":{}}`)
	if out["success"] != false || out["error"] == "" {
		t.Errorf("got success=%v error=%v, want a validation error", out["success"], out["error"])
	}
}

func TestInitRecordReturnsDefaults(t *testing.T) {
	app := newTablesTestApp(t)

	out := postJSON(t, app, "/api/tables/Payment_terms/init", `{}`)
	if out["success"] != true {
		t.Fatalf("success = %v, error = %v", out["success"], out["error"])
	}
	data, _ := out["data"].(map[string]interface{})
	if data["active"] != true {
		t.Errorf("data.active = %v, want YAML default true", data["active"])
	}
	if data["code"] != "" {
		t.Errorf("data.code = %v, want blank", data["code"])
	}
}

// A stale sibling value in the payload must not overwrite what another field's
// OnValidate trigger filled in.
func TestValidateChangedFieldsKeepsTriggerFilledSibling(t *testing.T) {
	table := &autoFillCustomer{}
	table.InitWithDBType(nil, "TEST", database.DBTypeSQLite)
	table.FromMap(map[string]interface{}{"no": "C1", "name": "Old"})

	data := map[string]interface{}{"no": "C2", "name": "Old", "city": "Oslo"}
	if err := validateChangedFields(table, "Test_AutoFill", data); err != nil {
		t.Fatalf("validateChangedFields: %v", err)
	}

	got := table.ToMap()
	if got["name"] != "Filled from C2" {
		t.Errorf("name = %v, want trigger-filled %q", got["name"], "Filled from C2")
	}
	if got["city"] != "Oslo" {
		t.Errorf("city = %v, want Oslo", got["city"])
	}
}

func TestValidateChangedFieldsReportsUnknownField(t *testing.T) {
	table := &autoFillCustomer{}
	table.InitWithDBType(nil, "TEST", database.DBTypeSQLite)

	if err := validateChangedFields(table, "Test_AutoFill", map[string]interface{}{"bogus": "x"}); err == nil {
		t.Error("validateChangedFields with unknown field = nil, want error")
	}
}

// The list page sends the whole init payload (every table field, including blank dates,
// FlowFields and nulls) with the user's edits on insert; that must insert cleanly.
func TestInitPayloadIsInsertable(t *testing.T) {
	app := newTablesTestApp(t)

	initOut := postJSON(t, app, "/api/tables/Customer/init", `{}`)
	record, _ := initOut["data"].(map[string]interface{})
	if record == nil {
		t.Fatalf("init returned no record: %v", initOut)
	}
	record["no"] = "c1"
	body, _ := json.Marshal(record)

	out := postJSON(t, app, "/api/tables/Customer/insert", string(body))
	if out["success"] != true {
		t.Fatalf("insert of init payload failed: %v", out["error"])
	}
	if data, _ := out["data"].(map[string]interface{}); data["no"] != "C1" {
		t.Errorf("inserted no = %v, want C1", data["no"])
	}
}

func getJSON(t *testing.T, app *fiber.App, target string) (int, map[string]interface{}) {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest("GET", target, nil))
	if err != nil {
		t.Fatalf("GET %s: %v", target, err)
	}
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %s: %v (%s)", target, err, raw)
	}
	return resp.StatusCode, out
}

// sort_by and filter field names from the query string must be real columns
// (they end up in SQL text); anything else is rejected before reaching SQL.
func TestListRejectsUnknownFieldNames(t *testing.T) {
	app := newTablesTestApp(t)
	postJSON(t, app, "/api/tables/Customer/insert", `{"no":"C1","name":"Acme"}`)

	injected := url.QueryEscape("no; DROP TABLE x --")
	badFilter := url.QueryEscape(`[{"field":"no) OR (1=1","expression":"x"}]`)
	for _, target := range []string{
		"/api/tables/Customer/list?sort_by=" + injected,
		"/api/tables/Customer/ids?sort_by=" + injected,
		"/api/tables/Customer/list?filters=" + badFilter,
	} {
		status, out := getJSON(t, app, target)
		if status != 400 || out["success"] != false {
			t.Errorf("GET %s = %d %v, want 400", target, status, out)
		}
	}

	goodFilter := url.QueryEscape(`[{"field":"no","expression":"C1"}]`)
	status, out := getJSON(t, app, "/api/tables/Customer/list?sort_by=name&filters="+goodFilter)
	if status != 200 || out["success"] != true {
		t.Fatalf("valid sort/filter = %d %v, want 200", status, out)
	}
	data, _ := out["data"].(map[string]interface{})
	if total, _ := data["total"].(float64); total != 1 {
		t.Errorf("total = %v, want 1", data["total"])
	}
}
