package types

import (
	"database/sql/driver"
)

// Text is a custom type that preserves case sensitivity
// This matches Business Central's Text datatype behavior
type Text string

// NewText creates a new Text value
func NewText(s string) Text {
	return Text(s)
}

// Set updates the Text value
func (t *Text) Set(s string) {
	*t = Text(s)
}

// String returns the string representation
func (t Text) String() string {
	return string(t)
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON/form input
func (t *Text) UnmarshalText(text []byte) error {
	*t = Text(text)
	return nil
}

// MarshalText implements encoding.TextMarshaler for JSON output
func (t Text) MarshalText() ([]byte, error) {
	return []byte(t), nil
}

// Scan implements sql.Scanner for database scanning
func (t *Text) Scan(value interface{}) error {
	if value == nil {
		*t = ""
		return nil
	}

	switch v := value.(type) {
	case string:
		*t = Text(v)
	case []byte:
		*t = Text(string(v))
	default:
		*t = ""
	}

	return nil
}

// Value implements driver.Valuer for database storage
func (t Text) Value() (driver.Value, error) {
	return string(t), nil
}

// IsEmpty returns true if the Text is empty
func (t Text) IsEmpty() bool {
	return t == ""
}
