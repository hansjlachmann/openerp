package handlers

import (
	"github.com/gofiber/fiber/v2"
	apitypes "github.com/hansjlachmann/openerp/src/api/types"
	"github.com/hansjlachmann/openerp/src/foundation/session"
)

// SessionHandler handles session-related API requests
type SessionHandler struct {
	// Can add dependencies here if needed
}

// NewSessionHandler creates a new session handler
func NewSessionHandler() *SessionHandler {
	return &SessionHandler{}
}

// GetSession returns the current session information
// GET /api/session
func (h *SessionHandler) GetSession(c *fiber.Ctx) error {
	// Get current session (in real app, this would come from authentication)
	sess := session.GetCurrent()

	if sess == nil {
		// No active session - return empty session
		response := apitypes.NewSuccessResponse(&apitypes.SessionResponse{
			Database:     "",
			Company:      "",
			UserID:       "",
			UserName:     "",
			UserFullName: "",
			Language:     "en-US",
		})
		return c.JSON(response)
	}

	// Return session data
	dbPath := ""
	if db := sess.GetDatabase(); db != nil {
		dbPath = db.GetDatabasePath()
	}

	sessionData := &apitypes.SessionResponse{
		Database:     dbPath,
		Company:      sess.GetCompany(),
		UserID:       sess.GetUserID(),
		UserName:     sess.GetUserName(),
		UserFullName: sess.GetUserName(), // Session doesn't have full name, use username
		Language:     sess.GetLanguage(),
	}

	response := apitypes.NewSuccessResponse(sessionData)
	return c.JSON(response)
}
