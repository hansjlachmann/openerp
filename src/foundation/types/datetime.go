package types

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// DateTime represents a date and time (BC/NAV DateTime type)
// Stored in database as ISO 8601 format
type DateTime struct {
	time.Time
}

// NewDateTime creates a new DateTime from year, month, day, hour, minute, second
func NewDateTime(year int, month time.Month, day, hour, min, sec int) DateTime {
	return DateTime{time.Date(year, month, day, hour, min, sec, 0, time.UTC)}
}

// NewDateTimeFromTime creates a DateTime from a time.Time
func NewDateTimeFromTime(t time.Time) DateTime {
	return DateTime{t.UTC()}
}

// NewDateTimeFromString parses a datetime string in ISO 8601 format
func NewDateTimeFromString(value string) (DateTime, error) {
	if value == "" {
		return DateTime{}, nil
	}

	// Try multiple formats
	formats := []string{
		time.RFC3339,                 // "2006-01-02T15:04:05Z07:00"
		time.RFC3339Nano,             // "2006-01-02T15:04:05.999999999Z07:00"
		"2006-01-02 15:04:05",        // SQLite datetime format
		"2006-01-02T15:04:05",        // ISO 8601 without timezone
		"2006-01-02 15:04:05.999999", // SQLite with microseconds
	}

	var t time.Time
	var err error
	for _, format := range formats {
		t, err = time.Parse(format, value)
		if err == nil {
			return DateTime{t.UTC()}, nil
		}
	}

	return DateTime{}, fmt.Errorf("invalid datetime format: %w", err)
}

// MustDateTime creates a DateTime from a string, panics on error
func MustDateTime(value string) DateTime {
	dt, err := NewDateTimeFromString(value)
	if err != nil {
		panic(fmt.Sprintf("invalid datetime value: %s", value))
	}
	return dt
}

// Now returns the current date and time
func Now() DateTime {
	return DateTime{time.Now().UTC()}
}

// ZeroDateTime returns a zero DateTime value
func ZeroDateTime() DateTime {
	return DateTime{time.Time{}}
}

// String returns the datetime in ISO 8601 format
func (dt DateTime) String() string {
	if dt.IsZero() {
		return ""
	}
	return dt.Time.Format(time.RFC3339)
}

// Format returns the datetime formatted according to the layout
func (dt DateTime) Format(layout string) string {
	return dt.Time.Format(layout)
}

// IsZero returns true if the datetime is the zero value
func (dt DateTime) IsZero() bool {
	return dt.Time.IsZero()
}

// Equal checks if two datetimes are equal
func (dt DateTime) Equal(other DateTime) bool {
	return dt.Time.Equal(other.Time)
}

// Before checks if dt is before other
func (dt DateTime) Before(other DateTime) bool {
	return dt.Time.Before(other.Time)
}

// After checks if dt is after other
func (dt DateTime) After(other DateTime) bool {
	return dt.Time.After(other.Time)
}

// Add adds a duration to the datetime
func (dt DateTime) Add(d time.Duration) DateTime {
	return DateTime{dt.Time.Add(d)}
}

// AddDays adds the specified number of days
func (dt DateTime) AddDays(days int) DateTime {
	return DateTime{dt.Time.AddDate(0, 0, days)}
}

// AddHours adds the specified number of hours
func (dt DateTime) AddHours(hours int) DateTime {
	return DateTime{dt.Time.Add(time.Duration(hours) * time.Hour)}
}

// AddMinutes adds the specified number of minutes
func (dt DateTime) AddMinutes(minutes int) DateTime {
	return DateTime{dt.Time.Add(time.Duration(minutes) * time.Minute)}
}

// Sub returns the duration between two datetimes
func (dt DateTime) Sub(other DateTime) time.Duration {
	return dt.Time.Sub(other.Time)
}

// Date returns the date component as a Date type
func (dt DateTime) Date() Date {
	return NewDateFromTime(dt.Time)
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON/form input
func (dt *DateTime) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		dt.Time = time.Time{}
		return nil
	}

	parsed, err := NewDateTimeFromString(string(text))
	if err != nil {
		return err
	}

	dt.Time = parsed.Time
	return nil
}

// MarshalText implements encoding.TextMarshaler for JSON output
func (dt DateTime) MarshalText() ([]byte, error) {
	if dt.IsZero() {
		return []byte(""), nil
	}
	return []byte(dt.String()), nil
}

// Scan implements sql.Scanner for database scanning
func (dt *DateTime) Scan(value interface{}) error {
	if value == nil {
		dt.Time = time.Time{}
		return nil
	}

	switch v := value.(type) {
	case string:
		parsed, err := NewDateTimeFromString(v)
		if err != nil {
			return err
		}
		dt.Time = parsed.Time
	case []byte:
		parsed, err := NewDateTimeFromString(string(v))
		if err != nil {
			return err
		}
		dt.Time = parsed.Time
	case time.Time:
		dt.Time = v.UTC()
	default:
		return fmt.Errorf("unsupported type for DateTime: %T", v)
	}

	return nil
}

// Value implements driver.Valuer for database storage
// Stores as TEXT in ISO 8601 format
func (dt DateTime) Value() (driver.Value, error) {
	if dt.IsZero() {
		return nil, nil
	}
	return dt.String(), nil
}
