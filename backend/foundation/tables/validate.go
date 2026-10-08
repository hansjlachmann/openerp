package tables

import (
	"fmt"
	"unicode/utf8"

	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
)

// Field checks for the tables' Validate triggers. The errors name the field, so the user
// sees its caption in their own language ("Name cannot exceed 50 characters").

// CheckRequired fails when a field that must have a value is empty.
func CheckRequired(table, field string, empty bool) error {
	if empty {
		return apperrors.FieldRequired(table, field)
	}
	return nil
}

// CheckMaxLength fails when value has more than max characters (characters, not bytes:
// "Ørsta" is 5 long, like the database's VARCHAR counts it).
func CheckMaxLength(table, field, value string, max int) error {
	if utf8.RuneCountInString(value) > max {
		return apperrors.FieldTooLong(table, field, max)
	}
	return nil
}

// CheckRange fails when value is outside min..max.
func CheckRange[T ~int | ~int64 | ~float64](table, field string, value, min, max T) error {
	if value < min || value > max {
		return apperrors.FieldOutOfRange(table, field, fmt.Sprint(min), fmt.Sprint(max))
	}
	return nil
}
