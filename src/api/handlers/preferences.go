package handlers

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	apitypes "github.com/hansjlachmann/openerp/src/api/types"
	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/session"
	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// PreferencesHandler handles user preferences API requests
type PreferencesHandler struct {
	db *sql.DB
}

// NewPreferencesHandler creates a new preferences handler
func NewPreferencesHandler(db *sql.DB) *PreferencesHandler {
	return &PreferencesHandler{db: db}
}

// GetPreferences returns user preferences for a specific page and type
// GET /api/preferences/:page_id/:type
func (h *PreferencesHandler) GetPreferences(c *fiber.Ctx) error {
	pageID := c.Params("page_id")
	preferenceType := c.Params("type")
	sess := session.GetCurrent()

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("No active session"))
	}

	userID := sess.GetUserID()
	if userID == "" {
		return c.Status(400).JSON(apitypes.NewErrorResponse("No user logged in"))
	}

	// Query preferences for this user, page, and type
	var prefs tables.UserPreferences
	prefs.Init(h.db, "") // No company context

	// Set filters
	prefs.SetFilter("user_id", userID)
	prefs.SetFilter("page_id", pageID) // SetFilter handles type conversion
	prefs.SetFilter("preference_type", preferenceType)

	var results []map[string]interface{}

	if prefs.FindSet() {
		for {
			results = append(results, map[string]interface{}{
				"preference_name": prefs.Preference_name.String(),
				"preference_data": prefs.Preference_data.String(),
				"created_at":      prefs.Created_at.Time,
				"updated_at":      prefs.Updated_at.Time,
			})
			if !prefs.Next() {
				break
			}
		}
	}

	response := apitypes.NewSuccessResponse(results)
	return c.JSON(response)
}

// SavePreference saves or updates a user preference
// POST /api/preferences/:page_id/:type
func (h *PreferencesHandler) SavePreference(c *fiber.Ctx) error {
	pageID := c.Params("page_id")
	preferenceType := c.Params("type")
	sess := session.GetCurrent()

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("No active session"))
	}

	userID := sess.GetUserID()
	if userID == "" {
		return c.Status(400).JSON(apitypes.NewErrorResponse("No user logged in"))
	}

	// Parse request body
	var requestBody struct {
		PreferenceName string                 `json:"preference_name"`
		PreferenceData map[string]interface{} `json:"preference_data"`
	}

	if err := c.BodyParser(&requestBody); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Invalid request body"))
	}

	// Convert preference data to JSON string
	dataJSON, err := json.Marshal(requestBody.PreferenceData)
	if err != nil {
		return c.Status(500).JSON(apitypes.NewErrorResponse("Failed to serialize preference data"))
	}

	// Convert page_id string to int
	pageIDInt, err := strconv.Atoi(pageID)
	if err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Invalid page_id"))
	}

	// Check if preference already exists
	var pref tables.UserPreferences
	pref.Init(h.db, "")

	exists := pref.Get(
		types.NewCode(userID),
		pageIDInt,
		types.NewCode(preferenceType),
		types.NewCode(requestBody.PreferenceName),
	)
	now := types.NewDateTimeFromTime(time.Now())

	if exists {
		// Update existing preference
		pref.Preference_data = types.NewText(string(dataJSON))
		pref.Updated_at = now

		if !pref.Modify(true) {
			return c.Status(500).JSON(apitypes.NewErrorResponse("Failed to update preference"))
		}
	} else {
		// Insert new preference
		pref.Preference_data = types.NewText(string(dataJSON))
		pref.Created_at = now
		pref.Updated_at = now

		if !pref.Insert(true) {
			return c.Status(500).JSON(apitypes.NewErrorResponse("Failed to save preference"))
		}
	}

	response := apitypes.NewSuccessResponse(map[string]interface{}{
		"message": "Preference saved successfully",
	})
	return c.JSON(response)
}

// DeletePreference deletes a user preference
// DELETE /api/preferences/:page_id/:type/:name
func (h *PreferencesHandler) DeletePreference(c *fiber.Ctx) error {
	pageID := c.Params("page_id")
	preferenceType := c.Params("type")
	preferenceName := c.Params("name")
	sess := session.GetCurrent()

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("No active session"))
	}

	userID := sess.GetUserID()
	if userID == "" {
		return c.Status(400).JSON(apitypes.NewErrorResponse("No user logged in"))
	}

	// Convert page_id string to int
	pageIDInt, err := strconv.Atoi(pageID)
	if err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Invalid page_id"))
	}

	// Find and delete the preference
	var pref tables.UserPreferences
	pref.Init(h.db, "")

	if !pref.Get(
		types.NewCode(userID),
		pageIDInt,
		types.NewCode(preferenceType),
		types.NewCode(preferenceName),
	) {
		return c.Status(404).JSON(apitypes.NewErrorResponse("Preference not found"))
	}

	if !pref.Delete(true) {
		return c.Status(500).JSON(apitypes.NewErrorResponse("Failed to delete preference"))
	}

	response := apitypes.NewSuccessResponse(map[string]interface{}{
		"message": "Preference deleted successfully",
	})
	return c.JSON(response)
}
