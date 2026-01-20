package handlers

import (
	"database/sql"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	apitypes "github.com/hansjlachmann/openerp/backend/api/types"
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/i18n"
	"github.com/hansjlachmann/openerp/backend/foundation/session"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
)

// getLanguageOrDefault returns the session language or "en-US" as default
func getLanguageOrDefault(sess *session.Session) string {
	if sess != nil {
		lang := sess.GetLanguage()
		if lang != "" {
			return lang
		}
	}
	return "en-US"
}

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
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidRequestBody().Message("en-US")))
	}

	if requestBody.UserID == "" || requestBody.Password == "" {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.UserRequired().Message("en-US")))
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
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.CompanyNotFound().Message("en-US")))
	}
	if err != nil {
		return c.Status(500).JSON(apitypes.NewErrorResponse(apperrors.CompanyVerifyFailed().Message("en-US")))
	}

	sess := session.GetCurrent()

	var user tables.User
	user.InitWithDBType(h.db, company, h.dbType)

	if !user.Get(types.NewCode(requestBody.UserID)) {
		return c.Status(401).JSON(apitypes.NewErrorResponse(apperrors.InvalidCredentials().Message("en-US")))
	}

	// Check if user is active
	if !user.Active {
		return c.Status(401).JSON(apitypes.NewErrorResponse(apperrors.UserInactive().Message("en-US")))
	}

	// Verify password
	if !user.CheckPassword(requestBody.Password) {
		return c.Status(401).JSON(apitypes.NewErrorResponse(apperrors.InvalidCredentials().Message("en-US")))
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
			user.Menu.String(),
		)
		sess.SetCompany(company)
	}

	ts := i18n.GetInstance()
	userLang := user.Language.String()
	if userLang == "" {
		userLang = "en-US"
	}

	userMenu := user.Menu.String()
	if userMenu == "" {
		userMenu = "admin" // Default menu
	}

	response := apitypes.NewSuccessResponse(map[string]interface{}{
		"user_id":   user.User_id.String(),
		"user_name": user.User_name.String(),
		"email":     user.Email.String(),
		"language":  userLang,
		"menu":      userMenu,
		"company":   company,
		"message":   ts.Message("MSG_LOGIN_SUCCESS", userLang),
	})
	return c.JSON(response)
}

// Logout ends the current user session
// POST /api/auth/logout
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	sess := session.GetCurrent()
	language := getLanguageOrDefault(sess)
	if sess != nil {
		sess.SetUser("", "", "", "")
	}

	ts := i18n.GetInstance()
	response := apitypes.NewSuccessResponse(map[string]interface{}{
		"message": ts.Message("MSG_LOGOUT_SUCCESS", language),
	})
	return c.JSON(response)
}

// GetCurrentUser returns the currently logged in user
// GET /api/auth/user
func (h *AuthHandler) GetCurrentUser(c *fiber.Ctx) error {
	sess := session.GetCurrent()
	if sess == nil {
		return c.Status(401).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	language := getLanguageOrDefault(sess)
	userID := sess.GetUserID()
	if userID == "" {
		return c.Status(401).JSON(apitypes.NewErrorResponse(apperrors.NotLoggedIn().Message(language)))
	}

	// Get full user details
	company := sess.GetCompany()
	if company == "" {
		company = "cronus"
	}

	var user tables.User
	user.InitWithDBType(h.db, company, h.dbType)

	if !user.Get(types.NewCode(userID)) {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.UserNotFound().Message(language)))
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
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidRequestBody().Message("en-US")))
	}

	// Check if any users exist
	// Use default company "cronus" for user storage
	company := "cronus"
	var user tables.User
	user.InitWithDBType(h.db, company, h.dbType)
	count := user.Count()

	if count > 0 {
		return c.Status(403).JSON(apitypes.NewErrorResponse(apperrors.UsersExist().Message("en-US")))
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
		return c.Status(500).JSON(apitypes.NewErrorResponse(apperrors.CreateUserFailed().Message("en-US")))
	}

	ts := i18n.GetInstance()
	response := apitypes.NewSuccessResponse(map[string]interface{}{
		"message":   ts.Message("MSG_USER_CREATED", "en-US"),
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
		return c.Status(500).JSON(apitypes.NewErrorResponse(apperrors.CompanyListFailed().Message("en-US")))
	}
	defer rows.Close()

	var companies []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return c.Status(500).JSON(apitypes.NewErrorResponse(apperrors.CompanyListFailed().Message("en-US")))
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
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidRequestBody().Message("en-US")))
	}

	if requestBody.Language == "" {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.LanguageRequired().Message("en-US")))
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
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.UnsupportedLanguage(requestBody.Language).Message("en-US")))
	}

	sess := session.GetCurrent()
	if sess == nil {
		return c.Status(401).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	// Update session language (keep existing menu)
	sess.SetUser(sess.GetUserID(), sess.GetUserName(), requestBody.Language, sess.GetMenu())

	// Optionally persist to user record
	if requestBody.Persist {
		userID := sess.GetUserID()
		company := sess.GetCompany()
		if userID != "" && company != "" {
			var user tables.User
			user.InitWithDBType(h.db, company, h.dbType)
			if user.Get(types.NewCode(userID)) {
				user.Language = types.NewCode(requestBody.Language)
				user.Modify(false) // Don't run triggers
			}
		}
	}

	ts := i18n.GetInstance()
	response := apitypes.NewSuccessResponse(map[string]interface{}{
		"language": requestBody.Language,
		"message":  ts.Message("MSG_LANGUAGE_UPDATED", requestBody.Language),
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
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidRequestBody().Message("en-US")))
	}

	if requestBody.Name == "" {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.CompanyRequired().Message("en-US")))
	}

	// Validate company name (alphanumeric, underscores, hyphens only)
	name := strings.ToLower(strings.TrimSpace(requestBody.Name))
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.CompanyInvalidName().Message("en-US")))
		}
	}

	if len(name) < 2 || len(name) > 50 {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.CompanyNameLength().Message("en-US")))
	}

	// Check if company already exists
	var existingName string
	err := h.db.QueryRow(`SELECT name FROM "Company" WHERE name = $1`, name).Scan(&existingName)
	if err == nil {
		return c.Status(409).JSON(apitypes.NewErrorResponse(apperrors.CompanyAlreadyExists(name).Message("en-US")))
	}

	// Insert company
	_, err = h.db.Exec(`INSERT INTO "Company" (name) VALUES ($1)`, name)
	if err != nil {
		return c.Status(500).JSON(apitypes.NewErrorResponse(apperrors.CompanyCreateFailed().Message("en-US")))
	}

	ts := i18n.GetInstance()
	response := apitypes.NewSuccessResponse(map[string]interface{}{
		"name":    name,
		"message": ts.Message("MSG_COMPANY_CREATED", "en-US"),
	})
	return c.JSON(response)
}
