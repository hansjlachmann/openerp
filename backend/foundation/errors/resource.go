package errors

// Resources (table 156 and the tables around it)
const (
	ErrResCodeWhenTypeAll ErrorCode = "ERR_RES_CODE_WHEN_TYPE_ALL"
)

// ResCodeWhenTypeAll: a price or cost for all resources has no code ("Code cannot be specified
// when Type is All")
func ResCodeWhenTypeAll(table, field, field2 string, value any) *AppError {
	return &AppError{Code: ErrResCodeWhenTypeAll, Table: table, Field: field, Field2: field2, Value: value}
}
