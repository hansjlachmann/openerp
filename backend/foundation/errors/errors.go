package errors

import (
	"strings"

	"github.com/hansjlachmann/openerp/backend/foundation/i18n"
)

// ErrorCode represents a unique error identifier for translation lookup
type ErrorCode string

// Error codes - these map to translation keys in errors.yaml
const (
	ErrDuplicateRecord  ErrorCode = "ERR_DUPLICATE_RECORD"
	ErrRecordNotFound   ErrorCode = "ERR_RECORD_NOT_FOUND"
	ErrValidationFailed ErrorCode = "ERR_VALIDATION_FAILED"
	ErrRequiredField    ErrorCode = "ERR_REQUIRED_FIELD"
	ErrInvalidValue     ErrorCode = "ERR_INVALID_VALUE"
	ErrDeleteFailed     ErrorCode = "ERR_DELETE_FAILED"
	ErrInsertFailed     ErrorCode = "ERR_INSERT_FAILED"
	ErrModifyFailed     ErrorCode = "ERR_MODIFY_FAILED"
	ErrNoActiveSession  ErrorCode = "ERR_NO_ACTIVE_SESSION"
	ErrTableNotFound    ErrorCode = "ERR_TABLE_NOT_FOUND"
	ErrEmptyPrimaryKey  ErrorCode = "ERR_EMPTY_PRIMARY_KEY"
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
