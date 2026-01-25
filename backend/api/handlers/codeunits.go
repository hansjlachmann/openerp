package handlers

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	apitypes "github.com/hansjlachmann/openerp/backend/api/types"
	"github.com/hansjlachmann/openerp/backend/business-logic/codeunits"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/session"
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

// GenerateCustLedgerEntries generates random customer ledger entries
// POST /api/codeunits/generate-cust-ledger-entries
func (h *CodeunitsHandler) GenerateCustLedgerEntries(c *fiber.Ctx) error {
	sess := session.GetCurrent()

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	company := sess.GetCompany()
	language := sess.GetLanguage()

	// Parse request body
	var req struct {
		CustomerNo string `json:"customer_no"`
		Count      int    `json:"count"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidRequestBody().Message(language)))
	}

	if req.CustomerNo == "" {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Customer number is required"))
	}

	// Default count
	if req.Count <= 0 {
		req.Count = 5
	}

	// Create and run codeunit
	codeunit := codeunits.NewCustLedgerEntryGenerate(h.db, company, h.dbType)
	inserted, err := codeunit.Run(req.CustomerNo, req.Count)
	if err != nil {
		return c.Status(500).JSON(apitypes.NewErrorResponse(err.Error()))
	}

	return c.JSON(apitypes.NewSuccessResponse(map[string]interface{}{
		"inserted":    inserted,
		"customer_no": req.CustomerNo,
		"message":     "Ledger entries generated successfully",
	}))
}
