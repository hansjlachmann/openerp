package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/session"
)

func newThemesTestApp(t *testing.T, userID string) *fiber.App {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, tbl := range []interface {
		CreateTableWithDBType(database.Executor, string, database.DBType) error
	}{&tables.UserPersonalization{}, &tables.Theme{}, &tables.ThemeAsset{}} {
		if err := tbl.CreateTableWithDBType(db, "", database.DBTypeSQLite); err != nil {
			t.Fatalf("create table: %v", err)
		}
	}
	h := NewThemesHandler(db, database.DBTypeSQLite)
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session", &session.Session{UserID: userID, Company: "TEST", Language: "en-US"})
		return c.Next()
	})
	app.Get("/api/themes", h.ListThemes)
	app.Get("/api/appearance", h.GetAppearance)
	app.Put("/api/appearance", h.SaveAppearance)
	return app
}

func putJSON(t *testing.T, app *fiber.App, url, body string) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest("PUT", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("PUT %s: %v", url, err)
	}
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %s: %v (%s)", url, err, raw)
	}
	return resp.StatusCode, out
}

func appearanceSettingsOf(t *testing.T, out map[string]interface{}) map[string]interface{} {
	t.Helper()
	data, _ := out["data"].(map[string]interface{})
	s, _ := data["settings"].(map[string]interface{})
	if s == nil {
		t.Fatalf("no settings in %v", out)
	}
	return s
}

func TestAppearanceDefaultsSaveAndChecks(t *testing.T) {
	app := newThemesTestApp(t, "ANNA")

	// A user without a record gets BC Light/Dark, dark mode, no wallpaper
	_, out := getJSON(t, app, "/api/appearance")
	s := appearanceSettingsOf(t, out)
	if s["light_theme"] != "BC-LIGHT" || s["dark_theme"] != "BC-DARK" || s["mode"] != "dark" || s["wallpaper"] != "" || out["data"].(map[string]interface{})["saved"] != false {
		t.Fatalf("defaults = %v", s)
	}
	dark := out["data"].(map[string]interface{})["dark"].(map[string]interface{})
	if dark["code"] != "BC-DARK" || dark["tokens"] == nil {
		t.Errorf("dark slot = %v", dark)
	}

	// Choose Tokyo Night for dark mode and switch to light mode
	code, out := putJSON(t, app, "/api/appearance", `{"dark_theme":"TOKYO-NIGHT","density":"compact"}`)
	if code != 200 {
		t.Fatalf("save = %d %v", code, out)
	}
	if _, out = putJSON(t, app, "/api/appearance", `{"mode":"light"}`); out["success"] != true {
		t.Fatalf("mode = %v", out)
	}
	// Only the fields sent change
	_, out = getJSON(t, app, "/api/appearance")
	s = appearanceSettingsOf(t, out)
	if s["dark_theme"] != "TOKYO-NIGHT" || s["mode"] != "light" || s["density"] != "compact" || s["light_theme"] != "BC-LIGHT" {
		t.Errorf("after save = %v", s)
	}

	// The table's triggers refuse bad values; nothing changes
	bad := map[string]string{
		`{"dark_theme":"NO-SUCH-THEME"}`:    "does not exist",
		`{"dark_theme":"CATPPUCCIN-LATTE"}`: "other mode",
		`{"light_theme":"NORD"}`:            "other mode",
		`{"panel_opacity":40}`:              "between 75 and 100",
		`{"wallpaper_opacity":101}`:         "between 0 and 100",
		`{"font_size":300}`:                 "between 80 and 150",
		`{"mode":"sepia"}`:                  "",
		`{"font":"comic_sans"}`:             "",
	}
	for body, want := range bad {
		code, out := putJSON(t, app, "/api/appearance", body)
		msg, _ := out["error"].(string)
		if code != 400 || out["success"] != false || !strings.Contains(msg, want) {
			t.Errorf("%s = %d %q, want 400 containing %q", body, code, msg, want)
		}
	}
	_, out = getJSON(t, app, "/api/appearance")
	if s2 := appearanceSettingsOf(t, out); s2["dark_theme"] != "TOKYO-NIGHT" || s2["panel_opacity"] != float64(92) {
		t.Errorf("after refused saves = %v", s2)
	}

	// Another user still has the defaults
	other := newThemesTestApp(t, "BJORN")
	_, out = getJSON(t, other, "/api/appearance")
	if s := appearanceSettingsOf(t, out); s["dark_theme"] != "BC-DARK" {
		t.Errorf("other user = %v", s)
	}
}

func TestListThemes(t *testing.T) {
	app := newThemesTestApp(t, "ANNA")
	_, out := getJSON(t, app, "/api/themes")
	data, _ := out["data"].(map[string]interface{})
	list, _ := data["themes"].([]interface{})
	// No roles assigned: the permission system is not configured, everything is allowed
	if data["can_install"] != true || data["can_delete"] != true {
		t.Errorf("permissions = %v", data)
	}
	codes := map[string]bool{}
	for _, item := range list {
		codes[item.(map[string]interface{})["code"].(string)] = true
	}
	for _, c := range []string{"BC-LIGHT", "BC-DARK", "TOKYO-NIGHT", "NORD", "CATPPUCCIN-LATTE"} {
		if !codes[c] {
			t.Errorf("theme %s missing from %v", c, codes)
		}
	}
}
