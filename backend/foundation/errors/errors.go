package errors

import (
	"strings"

	"github.com/hansjlachmann/openerp/backend/foundation/i18n"
)

// ErrorCode represents a unique error identifier for translation lookup
type ErrorCode string

// Error codes - these map to translation keys in errors.yaml
const (
	// Record operations
	ErrDuplicateRecord  ErrorCode = "ERR_DUPLICATE_RECORD"
	ErrRecordNotFound   ErrorCode = "ERR_RECORD_NOT_FOUND"
	ErrValidationFailed ErrorCode = "ERR_VALIDATION_FAILED"
	ErrRequiredField    ErrorCode = "ERR_REQUIRED_FIELD"
	ErrInvalidValue     ErrorCode = "ERR_INVALID_VALUE"
	ErrDeleteFailed     ErrorCode = "ERR_DELETE_FAILED"
	ErrInsertFailed     ErrorCode = "ERR_INSERT_FAILED"
	ErrModifyFailed     ErrorCode = "ERR_MODIFY_FAILED"
	ErrEmptyPrimaryKey  ErrorCode = "ERR_EMPTY_PRIMARY_KEY"
	ErrTableNotFound    ErrorCode = "ERR_TABLE_NOT_FOUND"

	// Session/Auth
	ErrNoActiveSession    ErrorCode = "ERR_NO_ACTIVE_SESSION"
	ErrNotLoggedIn        ErrorCode = "ERR_NOT_LOGGED_IN"
	ErrInvalidCredentials ErrorCode = "ERR_INVALID_CREDENTIALS"
	ErrUserInactive       ErrorCode = "ERR_USER_INACTIVE"
	ErrUserNotFound       ErrorCode = "ERR_USER_NOT_FOUND"
	ErrUserRequired       ErrorCode = "ERR_USER_REQUIRED"
	ErrUsersExist         ErrorCode = "ERR_USERS_EXIST"
	ErrCreateUserFailed   ErrorCode = "ERR_CREATE_USER_FAILED"

	// Request/Input
	ErrInvalidRequestBody  ErrorCode = "ERR_INVALID_REQUEST_BODY"
	ErrInvalidFields       ErrorCode = "ERR_INVALID_FIELDS"
	ErrInvalidFilters      ErrorCode = "ERR_INVALID_FILTERS"
	ErrInvalidPageID       ErrorCode = "ERR_INVALID_PAGE_ID"
	ErrLanguageRequired    ErrorCode = "ERR_LANGUAGE_REQUIRED"
	ErrUnsupportedLanguage ErrorCode = "ERR_UNSUPPORTED_LANGUAGE"

	// Company
	ErrCompanyNotFound      ErrorCode = "ERR_COMPANY_NOT_FOUND"
	ErrCompanyVerifyFailed  ErrorCode = "ERR_COMPANY_VERIFY_FAILED"
	ErrCompanyRequired      ErrorCode = "ERR_COMPANY_REQUIRED"
	ErrCompanyInvalidName   ErrorCode = "ERR_COMPANY_INVALID_NAME"
	ErrCompanyNameLength    ErrorCode = "ERR_COMPANY_NAME_LENGTH"
	ErrCompanyListFailed    ErrorCode = "ERR_COMPANY_LIST_FAILED"
	ErrCompanyAlreadyExists ErrorCode = "ERR_COMPANY_ALREADY_EXISTS"
	ErrCompanyCreateFailed  ErrorCode = "ERR_COMPANY_CREATE_FAILED"

	// Preferences
	ErrPreferenceNotFound     ErrorCode = "ERR_PREFERENCE_NOT_FOUND"
	ErrPreferenceSerialize    ErrorCode = "ERR_PREFERENCE_SERIALIZE"
	ErrPreferenceUpdateFailed ErrorCode = "ERR_PREFERENCE_UPDATE_FAILED"
	ErrPreferenceSaveFailed   ErrorCode = "ERR_PREFERENCE_SAVE_FAILED"
	ErrPreferenceDeleteFailed ErrorCode = "ERR_PREFERENCE_DELETE_FAILED"

	// Menu/Pages
	ErrMenuNotFound  ErrorCode = "ERR_MENU_NOT_FOUND"
	ErrPageNotFound  ErrorCode = "ERR_PAGE_NOT_FOUND"
)

// AppError represents an application error with translation support
type AppError struct {
	Code   ErrorCode
	Params []string // %1, %2, %3 replacements
}

// Error implements the error interface
func (e *AppError) Error() string {
	return e.Message("en-US")
}

// Message returns the translated, formatted error message
func (e *AppError) Message(language string) string {
	ts := i18n.GetInstance()
	key := "errors." + string(e.Code)
	template := ts.Translate(key, language)

	// Replace placeholders %1, %2, %3, etc. with params
	result := template
	for i, param := range e.Params {
		placeholder := "%" + string(rune('1'+i))
		result = strings.ReplaceAll(result, placeholder, param)
	}

	return result
}

// Helper constructors for common errors

// DuplicateRecord creates an error for duplicate record insertion
func DuplicateRecord(tableName, value string) *AppError {
	return &AppError{
		Code:   ErrDuplicateRecord,
		Params: []string{tableName, value},
	}
}

// RecordNotFound creates an error for record not found
func RecordNotFound(tableName, value string) *AppError {
	return &AppError{
		Code:   ErrRecordNotFound,
		Params: []string{tableName, value},
	}
}

// ValidationFailed creates an error for validation failure
func ValidationFailed(fieldName, reason string) *AppError {
	return &AppError{
		Code:   ErrValidationFailed,
		Params: []string{fieldName, reason},
	}
}

// RequiredField creates an error for missing required field
func RequiredField(fieldName string) *AppError {
	return &AppError{
		Code:   ErrRequiredField,
		Params: []string{fieldName},
	}
}

// InvalidValue creates an error for invalid field value
func InvalidValue(fieldName, value string) *AppError {
	return &AppError{
		Code:   ErrInvalidValue,
		Params: []string{fieldName, value},
	}
}

// DeleteFailed creates an error for failed delete operation
func DeleteFailed(tableName, value string) *AppError {
	return &AppError{
		Code:   ErrDeleteFailed,
		Params: []string{tableName, value},
	}
}

// InsertFailed creates an error for failed insert operation
func InsertFailed(tableName string) *AppError {
	return &AppError{
		Code:   ErrInsertFailed,
		Params: []string{tableName},
	}
}

// ModifyFailed creates an error for failed modify operation
func ModifyFailed(tableName, value string) *AppError {
	return &AppError{
		Code:   ErrModifyFailed,
		Params: []string{tableName, value},
	}
}

// NoActiveSession creates an error for missing session
func NoActiveSession() *AppError {
	return &AppError{
		Code:   ErrNoActiveSession,
		Params: []string{},
	}
}

// TableNotFound creates an error for unknown table
func TableNotFound(tableName string) *AppError {
	return &AppError{
		Code:   ErrTableNotFound,
		Params: []string{tableName},
	}
}

// EmptyPrimaryKey creates an error for empty primary key value
func EmptyPrimaryKey(tableName string) *AppError {
	return &AppError{
		Code:   ErrEmptyPrimaryKey,
		Params: []string{tableName},
	}
}

// NotLoggedIn creates an error for no user logged in
func NotLoggedIn() *AppError {
	return &AppError{Code: ErrNotLoggedIn}
}

// InvalidCredentials creates an error for invalid login credentials
func InvalidCredentials() *AppError {
	return &AppError{Code: ErrInvalidCredentials}
}

// UserInactive creates an error for inactive user account
func UserInactive() *AppError {
	return &AppError{Code: ErrUserInactive}
}

// UserNotFound creates an error for user not found
func UserNotFound() *AppError {
	return &AppError{Code: ErrUserNotFound}
}

// UserRequired creates an error for missing user ID/password
func UserRequired() *AppError {
	return &AppError{Code: ErrUserRequired}
}

// UsersExist creates an error when trying to create first user but users already exist
func UsersExist() *AppError {
	return &AppError{Code: ErrUsersExist}
}

// CreateUserFailed creates an error for failed user creation
func CreateUserFailed() *AppError {
	return &AppError{Code: ErrCreateUserFailed}
}

// InvalidRequestBody creates an error for invalid JSON body
func InvalidRequestBody() *AppError {
	return &AppError{Code: ErrInvalidRequestBody}
}

// InvalidFields creates an error for invalid fields parameter
func InvalidFields() *AppError {
	return &AppError{Code: ErrInvalidFields}
}

// InvalidFilters creates an error for invalid filters parameter
func InvalidFilters() *AppError {
	return &AppError{Code: ErrInvalidFilters}
}

// InvalidPageID creates an error for invalid page ID
func InvalidPageID() *AppError {
	return &AppError{Code: ErrInvalidPageID}
}

// LanguageRequired creates an error for missing language
func LanguageRequired() *AppError {
	return &AppError{Code: ErrLanguageRequired}
}

// UnsupportedLanguage creates an error for unsupported language code
func UnsupportedLanguage(languageCode string) *AppError {
	return &AppError{
		Code:   ErrUnsupportedLanguage,
		Params: []string{languageCode},
	}
}

// CompanyNotFound creates an error for company not found
func CompanyNotFound() *AppError {
	return &AppError{Code: ErrCompanyNotFound}
}

// CompanyVerifyFailed creates an error for failed company verification
func CompanyVerifyFailed() *AppError {
	return &AppError{Code: ErrCompanyVerifyFailed}
}

// CompanyRequired creates an error for missing company name
func CompanyRequired() *AppError {
	return &AppError{Code: ErrCompanyRequired}
}

// CompanyInvalidName creates an error for invalid company name characters
func CompanyInvalidName() *AppError {
	return &AppError{Code: ErrCompanyInvalidName}
}

// CompanyNameLength creates an error for invalid company name length
func CompanyNameLength() *AppError {
	return &AppError{Code: ErrCompanyNameLength}
}

// CompanyListFailed creates an error for failed company listing
func CompanyListFailed() *AppError {
	return &AppError{Code: ErrCompanyListFailed}
}

// CompanyAlreadyExists creates an error for company already exists
func CompanyAlreadyExists(companyName string) *AppError {
	return &AppError{
		Code:   ErrCompanyAlreadyExists,
		Params: []string{companyName},
	}
}

// CompanyCreateFailed creates an error for failed company creation
func CompanyCreateFailed() *AppError {
	return &AppError{Code: ErrCompanyCreateFailed}
}

// PreferenceNotFound creates an error for preference not found
func PreferenceNotFound() *AppError {
	return &AppError{Code: ErrPreferenceNotFound}
}

// PreferenceSerialize creates an error for failed preference serialization
func PreferenceSerialize() *AppError {
	return &AppError{Code: ErrPreferenceSerialize}
}

// PreferenceUpdateFailed creates an error for failed preference update
func PreferenceUpdateFailed() *AppError {
	return &AppError{Code: ErrPreferenceUpdateFailed}
}

// PreferenceSaveFailed creates an error for failed preference save
func PreferenceSaveFailed() *AppError {
	return &AppError{Code: ErrPreferenceSaveFailed}
}

// PreferenceDeleteFailed creates an error for failed preference deletion
func PreferenceDeleteFailed() *AppError {
	return &AppError{Code: ErrPreferenceDeleteFailed}
}

// MenuNotFound creates an error for menu not found
func MenuNotFound() *AppError {
	return &AppError{Code: ErrMenuNotFound}
}

// PageNotFound creates an error for page not found
func PageNotFound(pageID string) *AppError {
	return &AppError{
		Code:   ErrPageNotFound,
		Params: []string{pageID},
	}
}
