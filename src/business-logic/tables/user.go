package tables

import (
	"errors"
	"time"

	"github.com/hansjlachmann/openerp/src/foundation/database"
	"github.com/hansjlachmann/openerp/src/foundation/types"
	"golang.org/x/crypto/bcrypt"
)

//go:generate go run ../../../tools/tablegen/main.go

// NewUser creates a new User instance
func NewUser() *User {
	return &User{
		Active:   true,
		Language: types.NewCode("en-US"),
	}
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
// Field Validation Hooks
// ========================================
// These methods are called by auto-generated OnValidate triggers in user_gen.go
// Add your custom field validation logic here

// CustomValidate_User_id - Custom validation for user_id field
func (t *User) CustomValidate_User_id() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for user_id:
	// if len(t.User_id) < 3 {
	//     return errors.New("user_id must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_User_name - Custom validation for user_name field
func (t *User) CustomValidate_User_name() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for user_name:
	// if len(t.User_name) < 3 {
	//     return errors.New("user_name must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Email - Custom validation for email field
func (t *User) CustomValidate_Email() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for email:
	// if len(t.Email) < 3 {
	//     return errors.New("email must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Password_hash - Custom validation for password_hash field
func (t *User) CustomValidate_Password_hash() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for password_hash:
	// if len(t.Password_hash) < 3 {
	//     return errors.New("password_hash must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Language - Custom validation for language field
func (t *User) CustomValidate_Language() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for language:
	// if len(t.Language) < 3 {
	//     return errors.New("language must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Active - Custom validation for active field
func (t *User) CustomValidate_Active() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for active:
	// if len(t.Active) < 3 {
	//     return errors.New("active must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Created_at - Custom validation for created_at field
func (t *User) CustomValidate_Created_at() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for created_at:
	// if len(t.Created_at) < 3 {
	//     return errors.New("created_at must be at least 3 characters")
	// }

	return nil
}

// CustomValidate_Last_login - Custom validation for last_login field
func (t *User) CustomValidate_Last_login() error {
	// *** ADD YOUR CUSTOM VALIDATION LOGIC HERE ***
	// Example for last_login:
	// if len(t.Last_login) < 3 {
	//     return errors.New("last_login must be at least 3 characters")
	// }

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
