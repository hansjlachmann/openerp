package handlers

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	apitypes "github.com/hansjlachmann/openerp/backend/api/types"
	"github.com/hansjlachmann/openerp/backend/business-logic/codeunits"
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/session"

	fcodeunits "github.com/hansjlachmann/openerp/backend/foundation/codeunits"
)

// JobsHandler handles job-related API requests (codeunits with progress)
type JobsHandler struct {
	db     *sql.DB
	dbType database.DBType
}

// NewJobsHandler creates a new jobs handler
func NewJobsHandler(db *sql.DB) *JobsHandler {
	return NewJobsHandlerWithDBType(db, database.DBTypeSQLite)
}

// NewJobsHandlerWithDBType creates a new jobs handler with explicit database type
func NewJobsHandlerWithDBType(db *sql.DB, dbType database.DBType) *JobsHandler {
	return &JobsHandler{db: db, dbType: dbType}
}

// StartJob starts a codeunit as a background job with progress tracking
// POST /api/jobs/start
func (h *JobsHandler) StartJob(c *fiber.Ctx) error {
	sess := session.GetCurrent()

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	// Capture session context for use in codeunit
	company := sess.GetCompany()
	language := sess.GetLanguage()
	userID := sess.GetUserID()
	userName := sess.GetUserName()

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
	table.FromMap(req.Record)

	// Check if codeunit uses progress - if not, run synchronously
	if !codeunit.UsesProgress() {
		// Set context for sync execution (same goroutine)
		fcodeunits.SetCurrentContext(userID, userName, company, language)
		defer fcodeunits.ClearCurrentContext()

		// Run synchronously and return result directly
		result, err := codeunit.Run(table)
		if err != nil {
			return c.Status(500).JSON(apitypes.NewErrorResponse(err.Error()))
		}

		// Return the result with dialog info if present
		response := map[string]interface{}{
			"success": result.Success,
			"message": result.Message,
		}
		if result.Data != nil {
			response["data"] = result.Data
		}
		if result.Dialog != nil {
			response["dialog"] = result.Dialog
		}
		return c.JSON(apitypes.NewSuccessResponse(response))
	}

	// Codeunit uses progress - run async with SSE
	dialog := fcodeunits.OpenDialog("Processing...")
	jobID := dialog.JobID()

	// Run the codeunit in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				dialog.CloseWithError(fmt.Errorf("panic: %v", r))
			}
		}()

		// Set context for async execution (new goroutine)
		fcodeunits.SetCurrentContext(userID, userName, company, language)
		defer fcodeunits.ClearCurrentContext()

		// Set dialog in context for the codeunit to use
		fcodeunits.SetCurrentDialog(dialog)
		defer fcodeunits.ClearCurrentDialog()

		result, err := codeunit.Run(table)
		if err != nil {
			dialog.CloseWithError(err)
			return
		}

		// Send final result through dialog
		if result.Data != nil {
			// Close with data (e.g., PDF base64)
			message := result.Message
			if result.Dialog != nil {
				message = result.Dialog.Message
			}
			dialog.CloseWithData(result.Data, message)
		} else if result.Dialog != nil {
			dialog.UpdateWithMessage(0, 100, result.Dialog.Message)
			dialog.Close()
		} else {
			dialog.Close()
		}
	}()

	// Return job ID immediately
	return c.JSON(apitypes.NewSuccessResponse(map[string]interface{}{
		"job_id": jobID,
	}))
}

// GetJobEvents streams job progress events via SSE
// GET /api/jobs/:id/events
func (h *JobsHandler) GetJobEvents(c *fiber.Ctx) error {
	jobID := c.Params("id")
	if jobID == "" {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Job ID is required"))
	}

	// Get the job from registry
	dialog, ok := fcodeunits.GetJobRegistry().Get(jobID)
	if !ok {
		return c.Status(404).JSON(apitypes.NewErrorResponse("Job not found"))
	}

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	// Stream events
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		events := dialog.Events()
		ctx := dialog.Context()

		// Send initial connected event
		fmt.Fprintf(w, "event: connected\ndata: {\"job_id\":\"%s\"}\n\n", jobID)
		w.Flush()

		// Keep-alive ticker
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case event, ok := <-events:
				if !ok {
					// Channel closed
					fmt.Fprintf(w, "event: close\ndata: {}\n\n")
					w.Flush()
					return
				}

				// Send progress event - use JSON marshal to include Data field
				eventData := map[string]interface{}{
					"job_id":     event.JobID,
					"field":      event.Field,
					"value":      event.Value,
					"message":    event.Message,
					"completed":  event.Completed,
					"error":      event.Error,
					"timestamp":  event.Timestamp,
					"event_type": event.EventType,
					"confirm_id": event.ConfirmID,
				}
				if event.Data != nil {
					eventData["data"] = event.Data
				}
				jsonData, _ := json.Marshal(eventData)
				fmt.Fprintf(w, "event: progress\ndata: %s\n\n", string(jsonData))
				w.Flush()

				if event.Completed {
					return
				}

			case <-ticker.C:
				// Send keep-alive
				fmt.Fprintf(w, ": keepalive\n\n")
				w.Flush()

			case <-ctx.Done():
				// Job cancelled or completed
				fmt.Fprintf(w, "event: close\ndata: {}\n\n")
				w.Flush()
				return
			}
		}
	})

	return nil
}

// RespondToConfirm handles confirm dialog responses from the frontend
// POST /api/jobs/:id/confirm
func (h *JobsHandler) RespondToConfirm(c *fiber.Ctx) error {
	jobID := c.Params("id")
	if jobID == "" {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Job ID is required"))
	}

	var req struct {
		ConfirmID string `json:"confirm_id"`
		Response  bool   `json:"response"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Invalid request body"))
	}

	if req.ConfirmID == "" {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Confirm ID is required"))
	}

	// Get the job from registry
	dialog, ok := fcodeunits.GetJobRegistry().Get(jobID)
	if !ok {
		return c.Status(404).JSON(apitypes.NewErrorResponse("Job not found"))
	}

	// Send the response
	if !dialog.RespondToConfirm(req.ConfirmID, req.Response) {
		return c.Status(400).JSON(apitypes.NewErrorResponse("No pending confirm with that ID"))
	}

	return c.JSON(apitypes.NewSuccessResponse(map[string]interface{}{
		"acknowledged": true,
	}))
}
