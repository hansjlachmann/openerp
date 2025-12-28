package types

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Date represents a date without time component (BC/NAV Date type)
// Stored in database as "YYYY-MM-DD" format
type Date struct {
	time.Time
}

// NewDate creates a new Date from year, month, day
func NewDate(year int, month time.Month, day int) Date {
	return Date{time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

// NewDateFromTime creates a Date from a time.Time (strips time component)
func NewDateFromTime(t time.Time) Date {
	return Date{time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)}
}

// NewDateFromString parses a date string in "YYYY-MM-DD" format
func NewDateFromString(value string) (Date, error) {
	if value == "" {
		return Date{}, nil
	}

	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return Date{}, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}

	return Date{time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)}, nil
}

// MustDate creates a Date from a string, panics on error
func MustDate(value string) Date {
	d, err := NewDateFromString(value)
	if err != nil {
		panic(fmt.Sprintf("invalid date value: %s", value))
	}
	return d
}

// Today returns today's date
func Today() Date {
	now := time.Now()
	return Date{time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)}
}

// ZeroDate returns a zero Date value (0001-01-01)
func ZeroDate() Date {
	return Date{time.Time{}}
}

// String returns the date in "YYYY-MM-DD" format (ISO 8601)
func (d Date) String() string {
	if d.IsZero() {
		return ""
	}
	return d.Time.Format("2006-01-02")
}

// Format returns the date formatted according to the layout
func (d Date) Format(layout string) string {
	return d.Time.Format(layout)
}

// IsZero returns true if the date is the zero value
func (d Date) IsZero() bool {
	return d.Time.IsZero()
}

// Equal checks if two dates are the same day
func (d Date) Equal(other Date) bool {
	return d.Year() == other.Year() &&
		d.Month() == other.Month() &&
		d.Day() == other.Day()
}

// Before checks if d is before other
func (d Date) Before(other Date) bool {
	return d.Time.Before(other.Time)
}

// After checks if d is after other
func (d Date) After(other Date) bool {
	return d.Time.After(other.Time)
}

// AddDays adds the specified number of days to the date
func (d Date) AddDays(days int) Date {
	newDate := d.Time.AddDate(0, 0, days)
	return Date{newDate}
}

// AddMonths adds the specified number of months to the date
func (d Date) AddMonths(months int) Date {
	newDate := d.Time.AddDate(0, months, 0)
	return Date{newDate}
}

// AddYears adds the specified number of years to the date
func (d Date) AddYears(years int) Date {
	newDate := d.Time.AddDate(years, 0, 0)
	return Date{newDate}
}

// DaysBetween returns the number of days between two dates
func (d Date) DaysBetween(other Date) int {
	duration := other.Time.Sub(d.Time)
	return int(duration.Hours() / 24)
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON/form input
func (d *Date) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		d.Time = time.Time{}
		return nil
	}

	parsed, err := NewDateFromString(string(text))
	if err != nil {
		return err
	}

	d.Time = parsed.Time
	return nil
}

// MarshalText implements encoding.TextMarshaler for JSON output
func (d Date) MarshalText() ([]byte, error) {
	if d.IsZero() {
		return []byte(""), nil
	}
	return []byte(d.String()), nil
}

// Scan implements sql.Scanner for database scanning
func (d *Date) Scan(value interface{}) error {
	if value == nil {
		d.Time = time.Time{}
		return nil
	}

	switch v := value.(type) {
	case string:
		parsed, err := NewDateFromString(v)
		if err != nil {
			return err
		}
		d.Time = parsed.Time
	case []byte:
		parsed, err := NewDateFromString(string(v))
		if err != nil {
			return err
		}
		d.Time = parsed.Time
	case time.Time:
		d.Time = time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, time.UTC)
	default:
		return fmt.Errorf("unsupported type for Date: %T", v)
	}

	return nil
}

// Value implements driver.Valuer for database storage
// Stores as TEXT in "YYYY-MM-DD" format
func (d Date) Value() (driver.Value, error) {
	if d.IsZero() {
		return nil, nil
	}
	return d.String(), nil
}
