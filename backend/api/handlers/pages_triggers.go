package handlers

import (
	"database/sql"
	"strconv"

	"github.com/gofiber/fiber/v2"

	apitypes "github.com/hansjlachmann/openerp/backend/api/types"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/pages"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// PageTriggersHandler runs page triggers (business-logic/pages) for the records pages show
type PageTriggersHandler struct {
	db     *sql.DB
	dbType database.DBType
}

// NewPageTriggersHandler creates the page triggers handler
func NewPageTriggersHandler(db *sql.DB, dbType database.DBType) *PageTriggersHandler {
	return &PageTriggersHandler{db: db, dbType: dbType}
}

// FieldStates runs the page's OnAfterGetRecord triggers for the record a card shows (body:
// the record's fields) and returns the fields they switched: {"field_states": {field:
// {"editable": false, "visible": true}}}
// POST /api/pages/:id/field-states
func (h *PageTriggersHandler) FieldStates(c *fiber.Ctx) error {
	sess := getSession(c)
	if sess == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}
	language := sess.GetLanguage()
	pageID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apitypes.NewErrorResponse(apperrors.InvalidPageID().Message(language)))
	}
	page, err := pages.GetRegistry().GetPage(pageID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(apitypes.NewErrorResponse(apperrors.PageNotFound(c.Params("id")).Message(language)))
	}
	var record map[string]interface{}
	if err := c.BodyParser(&record); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apitypes.NewErrorResponse(apperrors.InvalidRequestBody().Message(language)))
	}
	factory, ok := gtables.GetTableFactory(page.Page.SourceTable)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(apitypes.NewErrorResponse(apperrors.TableNotFound(page.Page.SourceTable).Message(language)))
	}
	rec := factory()
	rec.InitWithDBType(h.db, sess.GetCompany(), h.dbType)
	rec.FromMap(record)
	states, err := pages.RunAfterGetRecord(pageID, rec)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apitypes.NewErrorResponse(err.Error()))
	}
	return c.JSON(apitypes.NewSuccessResponse(map[string]interface{}{"field_states": states}))
}
