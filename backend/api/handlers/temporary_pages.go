package handlers

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/pages"
	"github.com/hansjlachmann/openerp/backend/foundation/session"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
)

// temporaryRecord is a generated record that can be made temporary (InitTemporary)
type temporaryRecord interface {
	InitTemporary(db database.Executor, company string, dbType database.DBType) error
}

// fillTemporaryPage: for a list page on a temporary source table (page_id of a page with
// source_table_temporary), the table becomes temporary and the page's OnFindRecords trigger
// fills it from the page's parameters (page_params) — the list's filters, search, sort and
// windows then read the temporary rows. Other requests leave the table as it is.
func (h *TablesHandler) fillTemporaryPage(c *fiber.Ctx, table ftables.Table, tableName string, sess *session.Session) error {
	pageID := c.QueryInt("page_id", 0)
	if pageID <= 0 {
		return nil
	}
	page, err := pages.GetRegistry().GetPage(pageID)
	if err != nil || !page.Page.SourceTableTemporary || !strings.EqualFold(page.Page.SourceTable, tableName) {
		return nil
	}
	temp, ok := table.(temporaryRecord)
	if !ok {
		return nil
	}
	if err := temp.InitTemporary(h.db, sess.GetCompany(), h.dbType); err != nil {
		return err
	}
	var params map[string]string
	if raw := c.Query("page_params", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &params)
	}
	// The trigger runs as the user (middleware.UserContext): it sees what the user may read
	return pages.RunFindRecords(pageID, table, params)
}
