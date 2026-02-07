package handlers

import (
	"database/sql"
	"fmt"
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

	// Check company access before proceeding (use canonical user ID from DB, not raw input)
	allowed, _ := h.userHasCompanyAccess(user.User_id.String(), company)
	if !allowed {
		return c.Status(403).JSON(apitypes.NewErrorResponse(apperrors.CompanyAccessDenied(company).Message("en-US")))
	}

	// Update last login (ignore failure - non-critical)
	user.UpdateLastLogin()
	_ = user.Modify(false)

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

		// Load permissions for this user/company
		hasRoles, isSuper, perms := h.loadUserPermissions(user.User_id.String(), company)
		sess.SetPermissions(hasRoles, isSuper, perms)
	}

	ts := i18n.GetInstance()
	userLang := user.Language.String()
	if userLang == "" {
		userLang = "en-US"
	}

	// Look up the translation_key from the Language table
	translationKey := "en-US" // Default
	var lang tables.Language
	lang.InitWithDBType(h.db, company, h.dbType)
	if lang.Get(types.NewCode(userLang)) {
		if !lang.Translation_key.IsEmpty() {
			translationKey = lang.Translation_key.String()
		}
	}

	userMenu := user.Menu.String()
	if userMenu == "" {
		userMenu = "admin" // Default menu
	}

	response := apitypes.NewSuccessResponse(map[string]interface{}{
		"user_id":         user.User_id.String(),
		"user_name":       user.User_name.String(),
		"email":           user.Email.String(),
		"language":        userLang,
		"translation_key": translationKey,
		"menu":            userMenu,
		"company":         company,
		"message":         ts.Message("MSG_LOGIN_SUCCESS", translationKey),
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

// ListCompanies returns available companies for the current user.
// If the user is logged in and has role memberships, only returns companies they have access to.
// If not logged in (login screen) or user has no roles, returns all companies.
// GET /api/auth/companies
func (h *AuthHandler) ListCompanies(c *fiber.Ctx) error {
	// Get all companies first
	rows, err := h.db.Query(`SELECT name FROM "Company" ORDER BY name`)
	if err != nil {
		return c.Status(500).JSON(apitypes.NewErrorResponse(apperrors.CompanyListFailed().Message("en-US")))
	}
	defer rows.Close()

	var allCompanies []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return c.Status(500).JSON(apitypes.NewErrorResponse(apperrors.CompanyListFailed().Message("en-US")))
		}
		allCompanies = append(allCompanies, name)
	}

	// If user is logged in, filter by allowed companies
	sess := session.GetCurrent()
	if sess != nil && sess.GetUserID() != "" {
		allowedCompanies, hasRoles := h.getUserAllowedCompanies(sess.GetUserID())
		if hasRoles && allowedCompanies != nil {
			// Filter to only allowed companies
			filtered := make([]string, 0, len(allowedCompanies))
			allowedSet := make(map[string]bool, len(allowedCompanies))
			for _, c := range allowedCompanies {
				allowedSet[strings.ToLower(c)] = true
			}
			for _, c := range allCompanies {
				if allowedSet[strings.ToLower(c)] {
					filtered = append(filtered, c)
				}
			}
			response := apitypes.NewSuccessResponse(filtered)
			return c.JSON(response)
		}
		// hasRoles=true but allowedCompanies=nil means blank company (all access)
		// hasRoles=false means no roles configured, show all
	}

	response := apitypes.NewSuccessResponse(allCompanies)
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
	supportedLanguages := []string{"en-US", "nb-NO", "da-DK"}
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

// SetCompany changes the current session company
// POST /api/auth/company
func (h *AuthHandler) SetCompany(c *fiber.Ctx) error {
	var requestBody struct {
		Company string `json:"company"`
	}

	if err := c.BodyParser(&requestBody); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidRequestBody().Message("en-US")))
	}

	if requestBody.Company == "" {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.CompanyRequired().Message("en-US")))
	}

	// Verify company exists
	var companyCheck string
	err := h.db.QueryRow(`SELECT name FROM "Company" WHERE name = $1`, requestBody.Company).Scan(&companyCheck)
	if err == sql.ErrNoRows {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.CompanyNotFound().Message("en-US")))
	}
	if err != nil {
		return c.Status(500).JSON(apitypes.NewErrorResponse(apperrors.CompanyVerifyFailed().Message("en-US")))
	}

	sess := session.GetCurrent()
	if sess == nil {
		return c.Status(401).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	// Check company access
	userID := sess.GetUserID()
	if userID != "" {
		allowed, _ := h.userHasCompanyAccess(userID, requestBody.Company)
		if !allowed {
			language := getLanguageOrDefault(sess)
			return c.Status(403).JSON(apitypes.NewErrorResponse(apperrors.CompanyAccessDenied(requestBody.Company).Message(language)))
		}
	}

	// Update session company
	sess.SetCompany(requestBody.Company)

	// Reload permissions for the new company context
	if userID != "" {
		hasRoles, isSuper, perms := h.loadUserPermissions(userID, requestBody.Company)
		sess.SetPermissions(hasRoles, isSuper, perms)
	}

	ts := i18n.GetInstance()
	language := getLanguageOrDefault(sess)
	response := apitypes.NewSuccessResponse(map[string]interface{}{
		"company": requestBody.Company,
		"message": ts.Message("MSG_COMPANY_CHANGED", language),
	})
	return c.JSON(response)
}

// GetLanguages returns supported languages
// GET /api/auth/languages
func (h *AuthHandler) GetLanguages(c *fiber.Ctx) error {
	languages := []map[string]string{
		{"code": "en-US", "name": "English (US)"},
		{"code": "da-DK", "name": "Dansk"},
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

// getUserAllowedCompanies returns the list of companies a user is allowed to access.
// If the user has no User_Member records at all, returns (nil, false) — meaning the permission
// system is not configured for this user and all companies are allowed.
// If the user has memberships, returns the distinct company names (blank company = all companies).
func (h *AuthHandler) getUserAllowedCompanies(userID string) ([]string, bool) {
	var query string
	if h.dbType == database.DBTypePostgres {
		query = `SELECT DISTINCT company FROM "User_Member" WHERE user_id = $1`
	} else {
		query = `SELECT DISTINCT company FROM "User_Member" WHERE user_id = ?`
	}

	rows, err := h.db.Query(query, userID)
	if err != nil {
		return nil, false
	}
	defer rows.Close()

	var companies []string
	hasBlank := false
	for rows.Next() {
		var company string
		if err := rows.Scan(&company); err != nil {
			continue
		}
		if company == "" {
			hasBlank = true
		} else {
			companies = append(companies, company)
		}
	}

	// No memberships at all — permission system not configured for this user
	if len(companies) == 0 && !hasBlank {
		return nil, false
	}

	// Blank company means access to all companies
	if hasBlank {
		return nil, true
	}

	return companies, true
}

// userHasCompanyAccess checks if a user can access a specific company.
// Returns (allowed, hasRoles). If hasRoles is false, all companies are allowed.
func (h *AuthHandler) userHasCompanyAccess(userID, company string) (bool, bool) {
	allowedCompanies, hasRoles := h.getUserAllowedCompanies(userID)
	if !hasRoles {
		return true, false // No roles = all access
	}
	if allowedCompanies == nil {
		return true, true // Blank company = all companies
	}
	for _, c := range allowedCompanies {
		if strings.EqualFold(c, company) {
			return true, true
		}
	}
	return false, true
}

// loadUserPermissions loads and merges all permissions for a user in a company.
// Queries User_Member for the user's roles (matching the company or blank company = all companies),
// then queries Permission for each role, merging with OR logic across roles.
// loadUserPermissions returns (hasRoles, isSuper, permissions).
func (h *AuthHandler) loadUserPermissions(userID, company string) (bool, bool, map[string]session.TablePermission) {
	permissions := make(map[string]session.TablePermission)

	// Query User_Member for this user's roles
	// Include memberships where company matches OR company is blank (all companies)
	var query string
	var args []interface{}
	if h.dbType == database.DBTypePostgres {
		query = `SELECT role_id FROM "User_Member" WHERE user_id = $1 AND (company = $2 OR company = '')`
		args = []interface{}{userID, company}
	} else {
		query = `SELECT role_id FROM "User_Member" WHERE user_id = ? AND (company = ? OR company = '')`
		args = []interface{}{userID, company}
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		fmt.Printf("Warning: Failed to load user roles: %v\n", err)
		return false, false, permissions
	}
	defer rows.Close()

	var roleIDs []string
	isSuper := false
	for rows.Next() {
		var roleID string
		if err := rows.Scan(&roleID); err != nil {
			continue
		}
		roleIDs = append(roleIDs, roleID)
		if strings.ToUpper(roleID) == "SUPER" {
			isSuper = true
		}
	}

	// No roles assigned = permission system not configured for this user
	if len(roleIDs) == 0 {
		return false, false, permissions
	}

	// SUPER bypasses all checks — no need to load individual permissions
	if isSuper {
		return true, true, nil
	}

	// Query Permission table for all assigned roles
	placeholders := make([]string, len(roleIDs))
	permArgs := make([]interface{}, len(roleIDs))
	for i, id := range roleIDs {
		if h.dbType == database.DBTypePostgres {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		} else {
			placeholders[i] = "?"
		}
		permArgs[i] = id
	}

	permQuery := fmt.Sprintf(
		`SELECT role_id, table_name, can_read, can_insert, can_modify, can_delete FROM "Permission" WHERE role_id IN (%s)`,
		strings.Join(placeholders, ", "),
	)

	permRows, err := h.db.Query(permQuery, permArgs...)
	if err != nil {
		fmt.Printf("Warning: Failed to load permissions: %v\n", err)
		return true, false, permissions
	}
	defer permRows.Close()

	// Merge permissions with OR logic across roles
	for permRows.Next() {
		var roleID, tableName string
		var canRead, canInsert, canModify, canDelete bool
		if err := permRows.Scan(&roleID, &tableName, &canRead, &canInsert, &canModify, &canDelete); err != nil {
			continue
		}

		existing := permissions[tableName]
		existing.Read = existing.Read || canRead
		existing.Insert = existing.Insert || canInsert
		existing.Modify = existing.Modify || canModify
		existing.Delete = existing.Delete || canDelete
		permissions[tableName] = existing
	}

	return true, false, permissions
}
