package session

import (
	"bufio"
	"database/sql"

	"github.com/hansjlachmann/openerp/src/foundation/database"
)

// Session represents a user session (similar to BC/NAV SESSION variable)
// Contains database connection, current company, and other session state
type Session struct {
	DB       *database.Database // Database connection
	Company  string             // Current company context
	Scanner  *bufio.Scanner     // Input scanner (for CLI)
	UserID   string             // Current user ID/username
	UserName string             // Current user full name
	Language string             // Application language (e.g., "en-US", "de-DE")
	// Future extensions:
	// DateFormat string
	// Permissions map[string]bool
}

// Global current session (like BC/NAV SESSION global variable)
var currentSession *Session

// NewSession creates a new session instance
func NewSession(db *database.Database, company string, scanner *bufio.Scanner) *Session {
	return &Session{
		DB:      db,
		Company: company,
		Scanner: scanner,
	}
}

// GetConnection returns the underlying SQL database connection
func (s *Session) GetConnection() *sql.DB {
	return s.DB.GetConnection()
}

// GetCompany returns the current company context
func (s *Session) GetCompany() string {
	return s.Company
}

// SetCompany changes the current company context
func (s *Session) SetCompany(company string) {
	s.Company = company
}

// GetDatabase returns the database instance
func (s *Session) GetDatabase() *database.Database {
	return s.DB
}

// GetScanner returns the input scanner
func (s *Session) GetScanner() *bufio.Scanner {
	return s.Scanner
}

// GetUserID returns the current user ID
func (s *Session) GetUserID() string {
	return s.UserID
}

// GetUserName returns the current user full name
func (s *Session) GetUserName() string {
	return s.UserName
}

// GetLanguage returns the current application language
func (s *Session) GetLanguage() string {
	return s.Language
}

// SetUser sets the current user information
func (s *Session) SetUser(userID, userName, language string) {
	s.UserID = userID
	s.UserName = userName
	s.Language = language
}

// ========================================
// Global Session Management
// ========================================

// SetCurrent sets the global current session
func SetCurrent(s *Session) {
	currentSession = s
}

// GetCurrent returns the global current session
func GetCurrent() *Session {
	return currentSession
}

// ClearCurrent clears the global current session
func ClearCurrent() {
	currentSession = nil
}
