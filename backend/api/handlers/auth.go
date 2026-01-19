package handlers

import (
	"database/sql"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	apitypes "github.com/hansjlachmann/openerp/backend/api/types"
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/session"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
)

// AuthHandler handles authentication API requests
type AuthHandler struct {
	db     *sql.DB
	dbType database.DBType
}

// NewAuthHandler creates a new auth handler (defaults to SQLite)
func NewAuthHandler(db *sql.DB) *AuthHandler {
	return NewAuthHandlerWithDBType(db, database.DBTypeSQLite)
}

// NewAuthHandlerWithDBType creates a new auth handler with explicit database type
func NewAuthHandlerWithDBType(db *sql.DB, dbType database.DBType) *AuthHandler {
	return &AuthHandler{db: db, dbType: dbType}
}

// Login authenticates a user and creates a session
// POST /api/auth/login
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var requestBody struct {
		UserID   string `json:"user_id"`
		Password string `json:"password"`
		Company  string `json:"company"`
	}

	if err := c.BodyParser(&requestBody); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Invalid request body"))
	}

	if requestBody.UserID == "" || requestBody.Password == "" {
		return c.Status(400).JSON(apitypes.NewErrorResponse("User ID and password are required"))
	}

	// Determine company - use provided company, or default to "cronus"
	company := requestBody.Company
	if company == "" {
		company = "cronus"
	}

	// Verify company exists
	var companyCheck string
	err := h.db.QueryRow(`SELECT name FROM "Company" WHERE name = $1`, company).Scan(&companyCheck)
	if err == sql.ErrNoRows {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Company does not exist"))
	}
	if err != nil {
		return c.Status(500).JSON(apitypes.NewErrorResponse("Failed to verify company"))
	}

	sess := session.GetCurrent()

	var user tables.User
	user.InitWithDBType(h.db, company, h.dbType)

	if !user.Get(types.NewCode(requestBody.UserID)) {
		return c.Status(401).JSON(apitypes.NewErrorResponse("Invalid credentials"))
	}

	// Check if user is active
	if !user.Active {
		return c.Status(401).JSON(apitypes.NewErrorResponse("User account is inactive"))
	}

	// Verify password
	if !user.CheckPassword(requestBody.Password) {
		return c.Status(401).JSON(apitypes.NewErrorResponse("Invalid credentials"))
	}

	// Update last login
	user.UpdateLastLogin()
	if !user.Modify(false) {
		// Don't fail login if last login update fails
		// Just log it (in production, use proper logging)
	}

	// Create/update session
	if sess == nil {
		// For API, we don't have a pre-existing session, so we'll just return user info
		// The frontend will store this and send it with future requests
		// In a production system, you'd want to use JWT tokens or session cookies
	} else {
		sess.SetUser(
			user.User_id.String(),
			user.User_name.String(),
			user.Language.String(),
		)
		sess.SetCompany(company)
	}

	response := apitypes.NewSuccessResponse(map[string]interface{}{
		"user_id":   user.User_id.String(),
		"user_name": user.User_name.String(),
		"email":     user.Email.String(),
		"language":  user.Language.String(),
		"company":   company,
		"message":   "Login successful",
	})
	return c.JSON(response)
}

// Logout ends the current user session
// POST /api/auth/logout
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	sess := session.GetCurrent()
	if sess != nil {
		sess.SetUser("", "", "")
	}

	response := apitypes.NewSuccessResponse(map[string]interface{}{
		"message": "Logout successful",
	})
	return c.JSON(response)
}

// GetCurrentUser returns the currently logged in user
// GET /api/auth/user
func (h *AuthHandler) GetCurrentUser(c *fiber.Ctx) error {
	sess := session.GetCurrent()
	if sess == nil {
		return c.Status(401).JSON(apitypes.NewErrorResponse("No active session"))
	}

	userID := sess.GetUserID()
	if userID == "" {
		return c.Status(401).JSON(apitypes.NewErrorResponse("Not logged in"))
	}

	// Get full user details
	company := sess.GetCompany()
	if company == "" {
		company = "cronus"
	}

	var user tables.User
	user.InitWithDBType(h.db, company, h.dbType)

	if !user.Get(types.NewCode(userID)) {
		return c.Status(404).JSON(apitypes.NewErrorResponse("User not found"))
	}

	response := apitypes.NewSuccessResponse(map[string]interface{}{
		"user_id":    user.User_id.String(),
		"user_name":  user.User_name.String(),
		"email":      user.Email.String(),
		"language":   user.Language.String(),
		"active":     user.Active,
		"created_at": user.Created_at.Time,
		"last_login": user.Last_login.Time,
	})
	return c.JSON(response)
}

// CreateInitialUser creates the first admin user (for setup)
// POST /api/auth/init
func (h *AuthHandler) CreateInitialUser(c *fiber.Ctx) error {
	var requestBody struct {
		UserID   string `json:"user_id"`
		UserName string `json:"user_name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&requestBody); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Invalid request body"))
	}

	// Check if any users exist
	// Use default company "cronus" for user storage
	company := "cronus"
	var user tables.User
	user.InitWithDBType(h.db, company, h.dbType)
	count := user.Count()

	if count > 0 {
		return c.Status(403).JSON(apitypes.NewErrorResponse("Users already exist. Use the user management interface."))
	}

	// Create the initial user
	now := time.Now()
	user.User_id = types.NewCode(requestBody.UserID)
	user.User_name = types.NewText(requestBody.UserName)
	user.Email = types.NewText(requestBody.Email)
	user.Language = types.NewText("en-US")
	user.Active = true
	user.Created_at = types.NewDateTimeFromTime(now)
	user.Last_login = types.NewDateTimeFromTime(now) // Initialize to avoid NULL

	if err := user.SetPassword(requestBody.Password); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(err.Error()))
	}

	if !user.Insert(true) {
		return c.Status(500).JSON(apitypes.NewErrorResponse("Failed to create user"))
	}

	response := apitypes.NewSuccessResponse(map[string]interface{}{
		"message":   "Initial user created successfully",
		"user_id":   user.User_id.String(),
		"user_name": user.User_name.String(),
	})
	return c.JSON(response)
}

// ListCompanies returns all available companies
// GET /api/auth/companies
func (h *AuthHandler) ListCompanies(c *fiber.Ctx) error {
	rows, err := h.db.Query(`SELECT name FROM "Company" ORDER BY name`)
	if err != nil {
		return c.Status(500).JSON(apitypes.NewErrorResponse("Failed to list companies"))
	}
	defer rows.Close()

	var companies []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return c.Status(500).JSON(apitypes.NewErrorResponse("Failed to read company name"))
		}
		companies = append(companies, name)
	}

	response := apitypes.NewSuccessResponse(companies)
	return c.JSON(response)
}

// SetLanguage changes the current session language
// POST /api/auth/language
func (h *AuthHandler) SetLanguage(c *fiber.Ctx) error {
	var requestBody struct {
		Language string `json:"language"`
		Persist  bool   `json:"persist"` // If true, also update user record
	}

	if err := c.BodyParser(&requestBody); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Invalid request body"))
	}

	if requestBody.Language == "" {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Language is required"))
	}

	// Validate language code (must be one of supported languages)
	supportedLanguages := []string{"en-US", "nb-NO"}
	valid := false
	for _, lang := range supportedLanguages {
		if lang == requestBody.Language {
			valid = true
			break
		}
	}
	if !valid {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Unsupported language: " + requestBody.Language))
	}

	sess := session.GetCurrent()
	if sess == nil {
		return c.Status(401).JSON(apitypes.NewErrorResponse("No active session"))
	}

	// Update session language
	sess.SetUser(sess.GetUserID(), sess.GetUserName(), requestBody.Language)

	// Optionally persist to user record
	if requestBody.Persist {
		userID := sess.GetUserID()
		company := sess.GetCompany()
		if userID != "" && company != "" {
			var user tables.User
			user.InitWithDBType(h.db, company, h.dbType)
			if user.Get(types.NewCode(userID)) {
				user.Language = types.NewText(requestBody.Language)
				user.Modify(false) // Don't run triggers
			}
		}
	}

	response := apitypes.NewSuccessResponse(map[string]interface{}{
		"language": requestBody.Language,
		"message":  "Language updated successfully",
	})
	return c.JSON(response)
}

// GetLanguages returns supported languages
// GET /api/auth/languages
func (h *AuthHandler) GetLanguages(c *fiber.Ctx) error {
	languages := []map[string]string{
		{"code": "en-US", "name": "English (US)"},
		{"code": "nb-NO", "name": "Norsk (Bokmål)"},
	}

	response := apitypes.NewSuccessResponse(languages)
	return c.JSON(response)
}

// CreateCompany creates a new company
// POST /api/auth/companies
func (h *AuthHandler) CreateCompany(c *fiber.Ctx) error {
	var requestBody struct {
		Name string `json:"name"`
	}

	if err := c.BodyParser(&requestBody); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Invalid request body"))
	}

	if requestBody.Name == "" {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Company name is required"))
	}

	// Validate company name (alphanumeric, underscores, hyphens only)
	name := strings.ToLower(strings.TrimSpace(requestBody.Name))
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return c.Status(400).JSON(apitypes.NewErrorResponse("Company name can only contain letters, numbers, underscores, and hyphens"))
		}
	}

	if len(name) < 2 || len(name) > 50 {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Company name must be between 2 and 50 characters"))
	}

	// Check if company already exists
	var existingName string
	err := h.db.QueryRow(`SELECT name FROM "Company" WHERE name = $1`, name).Scan(&existingName)
	if err == nil {
		return c.Status(409).JSON(apitypes.NewErrorResponse("Company '" + name + "' already exists"))
	}

	// Insert company
	_, err = h.db.Exec(`INSERT INTO "Company" (name) VALUES ($1)`, name)
	if err != nil {
		return c.Status(500).JSON(apitypes.NewErrorResponse("Failed to create company: " + err.Error()))
	}

	response := apitypes.NewSuccessResponse(map[string]interface{}{
		"name":    name,
		"message": "Company created successfully",
	})
	return c.JSON(response)
}
