package handlers

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	apitypes "github.com/hansjlachmann/openerp/backend/api/types"
	"github.com/hansjlachmann/openerp/backend/business-logic/codeunits"
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	fcodeunits "github.com/hansjlachmann/openerp/backend/foundation/codeunits"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
)

// CodeunitsHandler handles codeunit-related API requests
type CodeunitsHandler struct {
	db     *sql.DB
	dbType database.DBType
}

// NewCodeunitsHandler creates a new codeunits handler
func NewCodeunitsHandler(db *sql.DB) *CodeunitsHandler {
	return NewCodeunitsHandlerWithDBType(db, database.DBTypeSQLite)
}

// NewCodeunitsHandlerWithDBType creates a new codeunits handler with explicit database type
func NewCodeunitsHandlerWithDBType(db *sql.DB, dbType database.DBType) *CodeunitsHandler {
	return &CodeunitsHandler{db: db, dbType: dbType}
}

// RunCodeunit executes a codeunit by ID with the provided record
// POST /api/codeunits/run
func (h *CodeunitsHandler) RunCodeunit(c *fiber.Ctx) error {
	sess := getSession(c)

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	company := sess.GetCompany()
	language := sess.GetLanguage()

	// Set goroutine context for sync codeunit execution
	fcodeunits.SetCurrentContext(sess.GetUserID(), sess.GetUserName(), company, language)
	defer fcodeunits.ClearCurrentContext()

	// Parse request body
	var req struct {
		CodeunitID int                    `json:"codeunit_id"`
		Record     map[string]interface{} `json:"record"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidRequestBody().Message(language)))
	}

	if req.CodeunitID == 0 {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Codeunit ID is required"))
	}

	// Get codeunit factory from registry
	factory, ok := codeunits.Get(req.CodeunitID)
	if !ok {
		return c.Status(404).JSON(apitypes.NewErrorResponse("Codeunit not found"))
	}

	// Create codeunit instance
	codeunit := factory(h.db, company, h.dbType)

	// Get the source table name and create a typed record
	tableName := codeunit.SourceTable()
	tableFactory, ok := tables.GetTableFactory(tableName)
	if !ok {
		return c.Status(500).JSON(apitypes.NewErrorResponse("Source table not found: " + tableName))
	}

	// Create table instance and populate from request
	table := tableFactory()
	table.InitWithDBType(h.db, company, h.dbType)

	// Populate table fields from the record map
	table.FromMap(req.Record)

	// Run the codeunit with the typed record
	result, err := codeunit.Run(table)
	if err != nil {
		return c.Status(500).JSON(apitypes.NewErrorResponse(err.Error()))
	}

	response := map[string]interface{}{
		"success": result.Success,
		"message": result.Message,
		"data":    result.Data,
	}
	if result.Dialog != nil {
		response["dialog"] = result.Dialog
	}
	return c.JSON(apitypes.NewSuccessResponse(response))
}
