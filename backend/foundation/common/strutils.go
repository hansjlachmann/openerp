package common

import (
	"fmt"
	"strings"
	"unicode"
)

// IncStr increments a string (BC/NAV style)
// Examples:
//   IncStr("001") -> "002"
//   IncStr("099") -> "100"
//   IncStr("ABC") -> "ABD"
//   IncStr("A99") -> "B00"
//   IncStr("") -> "1"
func IncStr(s string) string {
	if s == "" {
		return "1"
	}

	// Convert string to runes for proper Unicode handling
	runes := []rune(s)

	// Start from the rightmost character
	for i := len(runes) - 1; i >= 0; i-- {
		r := runes[i]

		if unicode.IsDigit(r) {
			// Increment digit
			if r == '9' {
				runes[i] = '0'
				// Continue to carry over to next position
				if i == 0 {
					// Need to prepend '1'
					return "1" + string(runes)
				}
			} else {
				runes[i] = r + 1
				return string(runes)
			}
		} else if unicode.IsLetter(r) {
			// Increment letter
			if unicode.IsUpper(r) {
				if r == 'Z' {
					runes[i] = 'A'
					// Continue to carry over to next position
					if i == 0 {
						// Need to prepend 'A'
						return "A" + string(runes)
					}
				} else {
					runes[i] = r + 1
					return string(runes)
				}
			} else if unicode.IsLower(r) {
				if r == 'z' {
					runes[i] = 'a'
					// Continue to carry over to next position
					if i == 0 {
						// Need to prepend 'a'
						return "a" + string(runes)
					}
				} else {
					runes[i] = r + 1
					return string(runes)
				}
			}
		} else {
			// Non-alphanumeric character, skip to next position
			continue
		}
	}

	// If we get here, all positions carried over
	return string(runes)
}

// CopyStr copies a substring from a string (BC/NAV style)
// Parameters:
//   str: source string
//   position: starting position (1-based, BC/NAV style)
//   length: number of characters to copy (optional)
//
// Examples:
//   CopyStr("Hello World", 1, 5) -> "Hello"
//   CopyStr("Hello World", 7) -> "World"
//   CopyStr("Hello World", 1) -> "Hello World"
//   CopyStr("Hello", 10, 5) -> ""
func CopyStr(str string, position int, length ...int) string {
	if str == "" || position < 1 {
		return ""
	}

	// Convert to runes for proper Unicode handling
	runes := []rune(str)

	// Convert from 1-based to 0-based index
	startIdx := position - 1

	// Check if start position is beyond string length
	if startIdx >= len(runes) {
		return ""
	}

	// If no length specified, copy to end of string
	if len(length) == 0 {
		return string(runes[startIdx:])
	}

	// Calculate end position
	copyLen := length[0]
	if copyLen <= 0 {
		return ""
	}

	endIdx := startIdx + copyLen
	if endIdx > len(runes) {
		endIdx = len(runes)
	}

	return string(runes[startIdx:endIdx])
}

// PadStr pads a string to a specified length (BC/NAV style)
// Parameters:
//   str: source string
//   length: desired total length
//   padChar: character to use for padding (default: space)
//   leftPad: if true, pad on left; if false, pad on right (default: false)
func PadStr(str string, length int, padChar string, leftPad bool) string {
	if padChar == "" {
		padChar = " "
	}

	// Get first rune of padChar
	padRune := []rune(padChar)[0]
	runes := []rune(str)

	if len(runes) >= length {
		return str
	}

	padding := strings.Repeat(string(padRune), length-len(runes))

	if leftPad {
		return padding + str
	}
	return str + padding
}

// StrLen returns the length of a string in characters (BC/NAV style)
// Properly handles Unicode characters
func StrLen(str string) int {
	return len([]rune(str))
}

// DelChr deletes characters from a string (BC/NAV style)
// Parameters:
//   str: source string
//   where: where to delete ('=' exact match, '<' leading, '>' trailing, '<>' both)
//   which: characters to delete
//
// Examples:
//   DelChr("  Hello  ", '<>', ' ') -> "Hello"
//   DelChr("ABC123", '=', '123') -> "ABC"
func DelChr(str string, where string, which string) string {
	if str == "" {
		return ""
	}

	switch where {
	case "<":
		// Delete leading characters
		return strings.TrimLeft(str, which)
	case ">":
		// Delete trailing characters
		return strings.TrimRight(str, which)
	case "<>":
		// Delete both leading and trailing
		return strings.Trim(str, which)
	case "=":
		// Delete all occurrences
		for _, ch := range which {
			str = strings.ReplaceAll(str, string(ch), "")
		}
		return str
	default:
		return str
	}
}

// UpperCase converts string to uppercase (BC/NAV style)
func UpperCase(str string) string {
	return strings.ToUpper(str)
}

// LowerCase converts string to lowercase (BC/NAV style)
func LowerCase(str string) string {
	return strings.ToLower(str)
}

// StrPos finds position of substring (BC/NAV style)
// Returns 1-based position, 0 if not found
func StrPos(str string, substr string) int {
	idx := strings.Index(str, substr)
	if idx == -1 {
		return 0
	}
	return idx + 1 // Convert to 1-based
}

// ConvertStr replaces characters in a string (BC/NAV style)
// Parameters:
//   str: source string
//   fromChars: characters to replace
//   toChars: replacement characters
//
// Example:
//   ConvertStr("Hello", "el", "ip") -> "Hippo"
func ConvertStr(str string, fromChars string, toChars string) string {
	if len(fromChars) != len(toChars) {
		return str
	}

	result := str
	fromRunes := []rune(fromChars)
	toRunes := []rune(toChars)

	for i := 0; i < len(fromRunes); i++ {
		result = strings.ReplaceAll(result, string(fromRunes[i]), string(toRunes[i]))
	}

	return result
}

// InsStr inserts a substring into a string (BC/NAV style)
// Parameters:
//   str: source string
//   substr: substring to insert
//   position: 1-based position where to insert
//
// Example:
//   InsStr("Hello World", "Beautiful ", 7) -> "Hello Beautiful World"
func InsStr(str string, substr string, position int) string {
	if position < 1 {
		return str
	}

	runes := []rune(str)

	// Convert to 0-based index
	idx := position - 1

	if idx >= len(runes) {
		// Append at end
		return str + substr
	}

	return string(runes[:idx]) + substr + string(runes[idx:])
}

// DelStr deletes a substring from a string (BC/NAV style)
// Parameters:
//   str: source string
//   position: 1-based starting position
//   length: number of characters to delete
//
// Example:
//   DelStr("Hello World", 7, 6) -> "Hello"
func DelStr(str string, position int, length int) string {
	if position < 1 || length <= 0 {
		return str
	}

	runes := []rune(str)

	// Convert to 0-based index
	startIdx := position - 1

	if startIdx >= len(runes) {
		return str
	}

	endIdx := startIdx + length
	if endIdx > len(runes) {
		endIdx = len(runes)
	}

	return string(runes[:startIdx]) + string(runes[endIdx:])
}

// Format formats a string with arguments (similar to fmt.Sprintf)
// This is a convenience wrapper for BC/NAV style string formatting
func Format(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
