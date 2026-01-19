package types

import (
	"database/sql/driver"
	"fmt"

	"github.com/shopspring/decimal"
)

// Decimal is a custom type for exact decimal arithmetic (BC/NAV Decimal type)
// Uses shopspring/decimal internally for precision financial calculations
type Decimal struct {
	decimal.Decimal
}

// NewDecimal creates a new Decimal from a float64
func NewDecimal(value float64) Decimal {
	return Decimal{decimal.NewFromFloat(value)}
}

// NewDecimalFromString creates a new Decimal from a string
// Returns error if the string is not a valid decimal
func NewDecimalFromString(value string) (Decimal, error) {
	d, err := decimal.NewFromString(value)
	if err != nil {
		return Decimal{}, err
	}
	return Decimal{d}, nil
}

// NewDecimalFromInt creates a new Decimal from an integer
func NewDecimalFromInt(value int64) Decimal {
	return Decimal{decimal.NewFromInt(value)}
}

// MustDecimal creates a Decimal from a string, panics on error
// Use for constants where you know the value is valid
func MustDecimal(value string) Decimal {
	d, err := decimal.NewFromString(value)
	if err != nil {
		panic(fmt.Sprintf("invalid decimal value: %s", value))
	}
	return Decimal{d}
}

// Zero returns a zero Decimal value
func ZeroDecimal() Decimal {
	return Decimal{decimal.Zero}
}

// String returns the string representation with full precision
func (d Decimal) String() string {
	return d.Decimal.String()
}

// StringFixed returns the string representation with fixed decimal places
// This matches BC/NAV's behavior for displaying decimals
func (d Decimal) StringFixed(places int32) string {
	return d.Decimal.StringFixed(places)
}

// Float64 returns the float64 representation (may lose precision)
func (d Decimal) Float64() float64 {
	f, _ := d.Decimal.Float64()
	return f
}

// IsZero returns true if the value is zero
func (d Decimal) IsZero() bool {
	return d.Decimal.IsZero()
}

// IsPositive returns true if the value is positive
func (d Decimal) IsPositive() bool {
	return d.Decimal.IsPositive()
}

// IsNegative returns true if the value is negative
func (d Decimal) IsNegative() bool {
	return d.Decimal.IsNegative()
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON/form input
func (d *Decimal) UnmarshalText(text []byte) error {
	dec, err := decimal.NewFromString(string(text))
	if err != nil {
		return err
	}
	d.Decimal = dec
	return nil
}

// MarshalText implements encoding.TextMarshaler for JSON output
func (d Decimal) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// Scan implements sql.Scanner for database scanning
// Handles TEXT storage of decimal values
func (d *Decimal) Scan(value interface{}) error {
	if value == nil {
		d.Decimal = decimal.Zero
		return nil
	}

	switch v := value.(type) {
	case string:
		dec, err := decimal.NewFromString(v)
		if err != nil {
			return fmt.Errorf("failed to scan decimal from string: %w", err)
		}
		d.Decimal = dec
	case []byte:
		dec, err := decimal.NewFromString(string(v))
		if err != nil {
			return fmt.Errorf("failed to scan decimal from bytes: %w", err)
		}
		d.Decimal = dec
	case int64:
		d.Decimal = decimal.NewFromInt(v)
	case float64:
		d.Decimal = decimal.NewFromFloat(v)
	default:
		return fmt.Errorf("unsupported type for Decimal: %T", v)
	}

	return nil
}

// Value implements driver.Valuer for database storage
// Stores as TEXT to preserve exact decimal representation
func (d Decimal) Value() (driver.Value, error) {
	return d.String(), nil
}

// Add performs addition (BC/NAV style: d + other)
func (d Decimal) Add(other Decimal) Decimal {
	return Decimal{d.Decimal.Add(other.Decimal)}
}

// Sub performs subtraction (BC/NAV style: d - other)
func (d Decimal) Sub(other Decimal) Decimal {
	return Decimal{d.Decimal.Sub(other.Decimal)}
}

// Mul performs multiplication (BC/NAV style: d * other)
func (d Decimal) Mul(other Decimal) Decimal {
	return Decimal{d.Decimal.Mul(other.Decimal)}
}

// Div performs division (BC/NAV style: d / other)
func (d Decimal) Div(other Decimal) Decimal {
	return Decimal{d.Decimal.Div(other.Decimal)}
}

// Round rounds to the specified number of decimal places
func (d Decimal) Round(places int32) Decimal {
	return Decimal{d.Decimal.Round(places)}
}

// Abs returns the absolute value
func (d Decimal) Abs() Decimal {
	return Decimal{d.Decimal.Abs()}
}

// Cmp compares two decimals
// Returns: -1 if d < other, 0 if d == other, 1 if d > other
func (d Decimal) Cmp(other Decimal) int {
	return d.Decimal.Cmp(other.Decimal)
}

// Equal checks if two decimals are equal
func (d Decimal) Equal(other Decimal) bool {
	return d.Decimal.Equal(other.Decimal)
}

// GreaterThan checks if d > other
func (d Decimal) GreaterThan(other Decimal) bool {
	return d.Decimal.GreaterThan(other.Decimal)
}

// LessThan checks if d < other
func (d Decimal) LessThan(other Decimal) bool {
	return d.Decimal.LessThan(other.Decimal)
}
