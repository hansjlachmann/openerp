package types

import (
	"database/sql/driver"
	"strings"
)

// Code is a custom type that automatically converts to uppercase
// This matches Business Central's Code datatype behavior
type Code string

// NewCode creates a new Code value with automatic uppercase conversion
func NewCode(s string) Code {
	return Code(strings.ToUpper(strings.TrimSpace(s)))
}

// Set updates the Code value with automatic uppercase conversion
func (c *Code) Set(s string) {
	*c = Code(strings.ToUpper(strings.TrimSpace(s)))
}

// String returns the string representation
func (c Code) String() string {
	return string(c)
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON/form input
func (c *Code) UnmarshalText(text []byte) error {
	*c = Code(strings.ToUpper(string(text)))
	return nil
}

// MarshalText implements encoding.TextMarshaler for JSON output
func (c Code) MarshalText() ([]byte, error) {
	return []byte(c), nil
}

// Scan implements sql.Scanner for database scanning
func (c *Code) Scan(value interface{}) error {
	if value == nil {
		*c = ""
		return nil
	}

	switch v := value.(type) {
	case string:
		*c = Code(strings.ToUpper(v))
	case []byte:
		*c = Code(strings.ToUpper(string(v)))
	default:
		*c = ""
	}

	return nil
}

// Value implements driver.Valuer for database storage
func (c Code) Value() (driver.Value, error) {
	return string(c), nil
}

// IsEmpty returns true if the Code is empty
func (c Code) IsEmpty() bool {
	return c == ""
}
