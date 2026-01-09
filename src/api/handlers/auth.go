package handlers

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
	apitypes "github.com/hansjlachmann/openerp/src/api/types"
	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/session"
	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// AuthHandler handles authentication API requests
type AuthHandler struct {
	db *sql.DB
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{db: db}
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
	user.Init(h.db, company)

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
	user.Init(h.db, company)

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
	user.Init(h.db, company)
	count := user.Count()

	if count > 0 {
		return c.Status(403).JSON(apitypes.NewErrorResponse("Users already exist. Use the user management interface."))
	}

	// Create the initial user
	now := time.Now()
	user.User_id = types.NewCode(requestBody.UserID)
	user.User_name = types.NewText(requestBody.UserName)
	user.Email = types.NewText(requestBody.Email)
	user.Language = types.NewCode("en-US")
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
