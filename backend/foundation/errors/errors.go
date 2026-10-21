package errors

import (
	"strconv"
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
	// Field validation (table triggers): %1 is the field's caption
	ErrFieldTooLong       ErrorCode = "ERR_FIELD_TOO_LONG"
	ErrFieldTooShort      ErrorCode = "ERR_FIELD_TOO_SHORT"
	ErrFieldOutOfRange    ErrorCode = "ERR_FIELD_OUT_OF_RANGE"
	ErrFieldBelowMin      ErrorCode = "ERR_FIELD_BELOW_MIN"
	ErrFieldAboveMax      ErrorCode = "ERR_FIELD_ABOVE_MAX"
	ErrFieldFormat        ErrorCode = "ERR_FIELD_FORMAT"
	ErrRelatedNotFound    ErrorCode = "ERR_RELATED_NOT_FOUND"
	ErrRelatedInactive    ErrorCode = "ERR_RELATED_INACTIVE"
	ErrFieldMustBe        ErrorCode = "ERR_FIELD_MUST_BE"
	ErrFieldError         ErrorCode = "ERR_FIELD_ERROR"
	ErrEmailInvalid       ErrorCode = "ERR_EMAIL_INVALID"
	ErrDateFormulaInvalid ErrorCode = "ERR_DATE_FORMULA_INVALID"
	ErrEmailDuplicate     ErrorCode = "ERR_EMAIL_DUPLICATE"
	ErrThemeWrongMode     ErrorCode = "ERR_THEME_WRONG_MODE"
	ErrFilterInvalid      ErrorCode = "ERR_FILTER_INVALID"
	ErrDeleteTooMany      ErrorCode = "ERR_DELETE_TOO_MANY"
	ErrDeleteRecordFailed ErrorCode = "ERR_DELETE_RECORD_FAILED"
	ErrCopyTooMany        ErrorCode = "ERR_COPY_TOO_MANY"
	ErrSelectionTooMany   ErrorCode = "ERR_SELECTION_TOO_MANY"
	ErrInconsistent       ErrorCode = "ERR_INCONSISTENT"
	ErrFieldNotEditable   ErrorCode = "ERR_FIELD_NOT_EDITABLE"
	ErrMustBeRoundedTo    ErrorCode = "ERR_MUST_BE_ROUNDED_TO"
	ErrDecimalPlaces      ErrorCode = "ERR_DECIMAL_PLACES_FORMAT"
	ErrSameCurrency       ErrorCode = "ERR_SAME_CURRENCY"
	ErrNoExchangeRate     ErrorCode = "ERR_NO_EXCHANGE_RATE"
	ErrFieldMustNotBe     ErrorCode = "ERR_FIELD_MUST_NOT_BE"
	ErrEndTotalNoBegin    ErrorCode = "ERR_END_TOTAL_NO_BEGIN"
	ErrPasteRecordFailed  ErrorCode = "ERR_PASTE_RECORD_FAILED"
	ErrDeleteFailed       ErrorCode = "ERR_DELETE_FAILED"
	ErrInsertFailed       ErrorCode = "ERR_INSERT_FAILED"
	ErrModifyFailed       ErrorCode = "ERR_MODIFY_FAILED"
	ErrEmptyPrimaryKey    ErrorCode = "ERR_EMPTY_PRIMARY_KEY"
	ErrTableNotFound      ErrorCode = "ERR_TABLE_NOT_FOUND"

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
	ErrInvalidSortField    ErrorCode = "ERR_INVALID_SORT_FIELD"
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
	ErrCompanyAccessDenied  ErrorCode = "ERR_COMPANY_ACCESS_DENIED"

	// Preferences
	ErrPreferenceNotFound     ErrorCode = "ERR_PREFERENCE_NOT_FOUND"
	ErrPreferenceSerialize    ErrorCode = "ERR_PREFERENCE_SERIALIZE"
	ErrPreferenceUpdateFailed ErrorCode = "ERR_PREFERENCE_UPDATE_FAILED"
	ErrPreferenceSaveFailed   ErrorCode = "ERR_PREFERENCE_SAVE_FAILED"
	ErrPreferenceDeleteFailed ErrorCode = "ERR_PREFERENCE_DELETE_FAILED"

	// Permissions
	ErrPermissionDenied ErrorCode = "ERR_PERMISSION_DENIED"

	// Menu/Pages
	ErrMenuNotFound ErrorCode = "ERR_MENU_NOT_FOUND"
	ErrPageNotFound ErrorCode = "ERR_PAGE_NOT_FOUND"
)

// AppError represents an application error with translation support
type AppError struct {
	Code   ErrorCode
	Params []string // %1, %2, %3 replacements
	// Table and Field name a table field the error is about: its caption in the message's
	// language becomes %1 (Params follow as %2, %3, ...)
	Table, Field string
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

	params := e.Params
	if e.Field != "" {
		params = append([]string{ts.FieldCaption(e.Table, e.Field, language)}, params...)
	}

	// Replace placeholders %1, %2, %3, etc. with params
	result := template
	for i, param := range params {
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

// FieldRequired: a table field that must have a value is empty
func FieldRequired(table, field string) *AppError {
	return &AppError{Code: ErrRequiredField, Table: table, Field: field}
}

// FieldTooLong: a table field's value has more than max characters
func FieldTooLong(table, field string, max int) *AppError {
	return &AppError{Code: ErrFieldTooLong, Table: table, Field: field, Params: []string{strconv.Itoa(max)}}
}

// FieldTooShort: a table field's value has fewer than min characters
func FieldTooShort(table, field string, min int) *AppError {
	return &AppError{Code: ErrFieldTooShort, Table: table, Field: field, Params: []string{strconv.Itoa(min)}}
}

// FieldBelowMin: a table field's value is below its minimum (a range with only a lower bound)
func FieldBelowMin(table, field, min string) *AppError {
	return &AppError{Code: ErrFieldBelowMin, Table: table, Field: field, Params: []string{min}}
}

// FieldAboveMax: a table field's value is above its maximum (a range with only an upper bound)
func FieldAboveMax(table, field, max string) *AppError {
	return &AppError{Code: ErrFieldAboveMax, Table: table, Field: field, Params: []string{max}}
}

// FieldOutOfRange: a table field's value is outside min..max
func FieldOutOfRange(table, field, min, max string) *AppError {
	return &AppError{Code: ErrFieldOutOfRange, Table: table, Field: field, Params: []string{min, max}}
}

// FieldFormat: a table field's value does not have the expected format (e.g. "xx-XX")
func FieldFormat(table, field, format string) *AppError {
	return &AppError{Code: ErrFieldFormat, Table: table, Field: field, Params: []string{format}}
}

// RelatedNotFound: a table field refers to a record (value) that does not exist
func RelatedNotFound(table, field, value string) *AppError {
	return &AppError{Code: ErrRelatedNotFound, Table: table, Field: field, Params: []string{value}}
}

// FieldMustBe: a table field must have a certain value (TestField with a value)
func FieldMustBe(table, field, value string) *AppError {
	return &AppError{Code: ErrFieldMustBe, Table: table, Field: field, Params: []string{value}}
}

// FieldErrorText: an error about a table field with a translated text (FieldError)
func FieldErrorText(table, field, text string) *AppError {
	return &AppError{Code: ErrFieldError, Table: table, Field: field, Params: []string{text}}
}

// RelatedInactive: a table field refers to a record (value) that may not be used
func RelatedInactive(table, field, value string) *AppError {
	return &AppError{Code: ErrRelatedInactive, Table: table, Field: field, Params: []string{value}}
}

// DateFormulaInvalid: a date formula field holds a value (formula) that is not a date formula
func DateFormulaInvalid(table, field, formula string) *AppError {
	return &AppError{Code: ErrDateFormulaInvalid, Table: table, Field: field, Params: []string{formula}}
}

// EmailInvalid: a table field's e-mail address list holds an item (address) that is not
// an e-mail address
func EmailInvalid(table, field, address string) *AppError {
	return &AppError{Code: ErrEmailInvalid, Table: table, Field: field, Params: []string{address}}
}

// ThemeWrongMode: a theme slot (light/dark theme) holds a theme made for the other mode
func ThemeWrongMode(table, field, theme string) *AppError {
	return &AppError{Code: ErrThemeWrongMode, Table: table, Field: field, Params: []string{theme}}
}

// EmailDuplicate: a table field's e-mail address list holds an address twice
func EmailDuplicate(table, field, address string) *AppError {
	return &AppError{Code: ErrEmailDuplicate, Table: table, Field: field, Params: []string{address}}
}

// FilterInvalid: a filter expression the user typed for a field is not valid for its type
func FilterInvalid(table, field, expr string) *AppError {
	return &AppError{Code: ErrFilterInvalid, Table: table, Field: field, Params: []string{expr}}
}

// DeleteTooMany: a bulk delete selected more records than one delete may remove
func DeleteTooMany(max int) *AppError {
	return &AppError{Code: ErrDeleteTooMany, Params: []string{strconv.Itoa(max)}}
}

// DeleteRecordFailed: a record of a bulk delete could not be deleted (reason: its trigger's
// message); nothing was deleted
func DeleteRecordFailed(tableCaption, key, reason string) *AppError {
	return &AppError{Code: ErrDeleteRecordFailed, Params: []string{tableCaption, key, sentence(reason)}}
}

// MustBeRoundedTo: a precision field is not a multiple of another precision
func MustBeRoundedTo(table, field, precision string) *AppError {
	return &AppError{Code: ErrMustBeRoundedTo, Table: table, Field: field, Params: []string{precision}}
}

// DecimalPlacesFormat: a decimal places setting is not "n" or "min:max" (0–9)
func DecimalPlacesFormat(table, field string) *AppError {
	return &AppError{Code: ErrDecimalPlaces, Table: table, Field: field}
}

// SameCurrency: an exchange rate's currency and relational currency are the same
func SameCurrency(table, field string) *AppError {
	return &AppError{Code: ErrSameCurrency, Table: table, Field: field}
}

// NoExchangeRate: a currency has no exchange rate on or before a date
func NoExchangeRate(currency, date string) *AppError {
	return &AppError{Code: ErrNoExchangeRate, Params: []string{currency, date}}
}

// FieldMustNotBe: a table field has a value that does not allow the change (FieldError without
// text: "Account Type must not be Posting")
func FieldMustNotBe(table, field, value string) *AppError {
	return &AppError{Code: ErrFieldMustNotBe, Table: table, Field: field, Params: []string{value}}
}

// EndTotalWithoutBeginTotal: indenting the chart of accounts found an End-Total account
// without a Begin-Total before it
func EndTotalWithoutBeginTotal(account string) *AppError {
	return &AppError{Code: ErrEndTotalNoBegin, Params: []string{account}}
}

// FieldNotEditable: a page tried to change a read-only field (editable: false)
func FieldNotEditable(table, field string) *AppError {
	return &AppError{Code: ErrFieldNotEditable, Table: table, Field: field}
}

// Inconsistent: a transaction cannot be committed while tables are inconsistent (Consistent)
func Inconsistent(tables string) *AppError {
	return &AppError{Code: ErrInconsistent, Params: []string{tables}}
}

// SelectionTooMany: more records are selected than a codeunit may be run on (max)
func SelectionTooMany(max int) *AppError {
	return &AppError{Code: ErrSelectionTooMany, Params: []string{strconv.Itoa(max)}}
}

// CopyTooMany: a copy or paste holds more records than one copy/paste may (max)
func CopyTooMany(max int) *AppError {
	return &AppError{Code: ErrCopyTooMany, Params: []string{strconv.Itoa(max)}}
}

// PasteRecordFailed: record n of total of a paste could not be inserted (reason: the
// insert's message, naming the record); nothing was pasted
func PasteRecordFailed(n, total int, reason string) *AppError {
	return &AppError{Code: ErrPasteRecordFailed, Params: []string{strconv.Itoa(n), strconv.Itoa(total), sentence(reason)}}
}

// sentence ends a message that is followed by another sentence with a full stop
func sentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasSuffix(s, ".") || strings.HasSuffix(s, "!") || strings.HasSuffix(s, "?") {
		return s
	}
	return s + "."
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

// InvalidSortField creates an error for a sort_by field that is not a column of the table
func InvalidSortField() *AppError {
	return &AppError{Code: ErrInvalidSortField}
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

// CompanyAccessDenied creates an error when user is not allowed to access a company
func CompanyAccessDenied(companyName string) *AppError {
	return &AppError{
		Code:   ErrCompanyAccessDenied,
		Params: []string{companyName},
	}
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

// PermissionDenied creates an error for permission denied on a table operation
func PermissionDenied(operation, tableName string) *AppError {
	return &AppError{
		Code:   ErrPermissionDenied,
		Params: []string{operation, tableName},
	}
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
