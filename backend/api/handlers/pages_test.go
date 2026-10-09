package handlers

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/i18n"
	"github.com/hansjlachmann/openerp/backend/foundation/pages"
	"github.com/hansjlachmann/openerp/backend/foundation/session"
)

func TestCaptionKey(t *testing.T) {
	for name, want := range map[string]string{
		"Send Test E-mail":  "send_test_e_mail",
		"Back to List":      "back_to_list",
		"Refresh":           "refresh",
		" Ledger  Entries ": "ledger_entries",
	} {
		if got := captionKey(name); got != want {
			t.Errorf("captionKey(%q) = %q, want %q", name, got, want)
		}
	}
}

// Every action and card section of every page has a caption in every language
func TestPageCaptionsTranslated(t *testing.T) {
	ts := i18n.GetInstance()
	for id, page := range pages.GetRegistry().GetAllPages() {
		for _, lang := range []string{"en-US", "nb-NO", "da-DK"} {
			for _, a := range page.Page.Actions {
				key := "actions." + captionKey(a.Name)
				if ts.CommonTranslation(key, lang) == "common."+key {
					t.Errorf("page %d, %s: no translation common.%s (action %q)", id, lang, key, a.Name)
				}
			}
			for _, s := range page.Page.Layout.Sections {
				key := "sections." + captionKey(s.Name)
				if ts.CommonTranslation(key, lang) == "common."+key {
					t.Errorf("page %d, %s: no translation common.%s (section %q)", id, lang, key, s.Name)
				}
			}
		}
	}
}

// GetPage translates action and section captions into the user's language on a copy: the
// registry's shared definition keeps its YAML captions.
func TestGetPageTranslatesCopy(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session", &session.Session{Company: "TEST", Language: "nb-NO"})
		return c.Next()
	})
	app.Get("/api/pages/:id", NewPagesHandler().GetPage)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/pages/674", nil))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		Data pages.PageDefinition `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode: %v (%s)", err, raw)
	}
	captions := map[string]string{}
	for _, a := range out.Data.Page.Actions {
		captions[a.Name] = a.Caption
	}
	for _, s := range out.Data.Page.Layout.Sections {
		captions[s.Name] = s.Caption
	}
	for name, want := range map[string]string{"Send Test E-mail": "Send test-e-post", "Notification": "Varsling", "General": "Generelt"} {
		if captions[name] != want {
			t.Errorf("%s = %q, want %q", name, captions[name], want)
		}
	}

	shared, _ := pages.GetRegistry().GetPage(674)
	for _, a := range shared.Page.Actions {
		if a.Name == "Send Test E-mail" && a.Caption != "Send Test E-mail" {
			t.Errorf("registry page changed: caption %q", a.Caption)
		}
	}
	for _, s := range shared.Page.Layout.Sections {
		for _, f := range s.Fields {
			if f.PrimaryKey {
				t.Errorf("registry page changed: field %s flagged as primary key", f.Source)
			}
		}
	}
}

// Every option value of every table has a caption in every language, and optionCaptions
// sends it under the option's index
func TestOptionCaptionsTranslated(t *testing.T) {
	ts := i18n.GetInstance()
	for _, name := range tables.ListTableNames() {
		if strings.HasPrefix(name, "Test_") {
			continue // registered by tests
		}
		factory, _ := tables.GetTableFactory(name)
		table := factory()
		for field, values := range table.GetOptionFields() {
			for _, value := range values {
				if captionKey(value) == "" {
					continue // blank option " ": shown blank
				}
				for _, lang := range []string{"en-US", "nb-NO", "da-DK"} {
					if c := ts.OptionCaption(name, field, captionKey(value), lang); strings.HasPrefix(c, "tables.") {
						t.Errorf("%s: no caption %s (option %q of %s.%s)", lang, c, value, name, field)
					}
				}
			}
		}
	}

	var jq tables.JobQueue
	got := optionCaptions("Job_Queue", &jq, "NB-NO")
	if got["status"]["0"] != "Avvent" || got["status"]["1"] != "Klar" || got["notify_on"]["2"] != "Ved feil" {
		t.Errorf("Job_Queue options in nb-NO = %v", got)
	}
}
