package tables

import (
	"errors"
	"time"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
	"golang.org/x/crypto/bcrypt"
)

//go:generate go run ../../../tools/tablegen/main.go

// User wraps UserBase and adds trigger methods
type User struct {
	gtables.UserBase
}

// NewUser creates a new User instance
func NewUser() *User {
	u := &User{}
	u.Active = true
	u.Language = types.NewText("en-US")
	return u
}

// Init initializes the record with database context and sets up triggers
func (t *User) Init(db database.Executor, company string) {
	t.UserBase.Init(db, company)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *User) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *User) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *User) OnDelete(db database.Executor, company string) error {
	// Check if user has preferences
	// Note: We could allow deletion and cascade delete preferences, or prevent deletion
	// For now, we allow deletion (preferences will remain orphaned unless cleaned up)
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *User) OnRename() error {
	// TODO: Update related records if needed
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *User) Validate() error {
	if t.User_id.IsEmpty() {
		return errors.New("user_id is required")
	}
	if len(t.User_id) > 50 {
		return errors.New("user_id cannot exceed 50 characters")
	}
	if len(t.User_name) > 100 {
		return errors.New("user_name cannot exceed 100 characters")
	}
	if len(t.Email) > 100 {
		return errors.New("email cannot exceed 100 characters")
	}
	if len(t.Language) > 10 {
		return errors.New("language cannot exceed 10 characters")
	}

	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// SetPassword hashes and stores the password
func (t *User) SetPassword(password string) error {
	if password == "" {
		return errors.New("password cannot be empty")
	}
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	t.Password_hash = types.NewText(string(hashedPassword))
	return nil
}

// CheckPassword verifies if the provided password matches the stored hash
func (t *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(t.Password_hash.String()), []byte(password))
	return err == nil
}

// UpdateLastLogin updates the last login timestamp
func (t *User) UpdateLastLogin() {
	t.Last_login = types.NewDateTimeFromTime(time.Now())
}
