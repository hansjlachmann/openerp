// Package system holds the classic ERP system functions for business logic in Go (table
// triggers, codeunits): text, date, number and dialog functions with the classic names, so
// business logic reads like classic ERP code — system.CopyStr(…), system.CalcDate(…). Where a
// classic function does not fit Go, the easiest Go way is taken: positions are 1-based and
// count characters (not bytes), functions that can fail return an error, and values are
// formatted and read in the Go-native, language-independent formats (no session language).
// Record functions (TestField, CalcSums, TransferFields, …) are methods of the generated
// records.
package system

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// StrLen returns the number of characters of s
func StrLen(s string) int {
	return len([]rune(s))
}

// CopyStr returns length characters of s from position (1-based); without length the rest
// of s. A position before the start counts from 1; past the end gives "".
func CopyStr(s string, position int, length ...int) string {
	runes := []rune(s)
	if position < 1 {
		position = 1
	}
	start := position - 1
	if start >= len(runes) {
		return ""
	}
	end := len(runes)
	if len(length) > 0 {
		if length[0] <= 0 {
			return ""
		}
		end = min(start+length[0], len(runes))
	}
	return string(runes[start:end])
}

// StrPos returns the position (1-based) of the first occurrence of sub in s, 0 if there is
// none
func StrPos(s, sub string) int {
	i := strings.Index(s, sub)
	if i < 0 || sub == "" {
		return 0
	}
	return len([]rune(s[:i])) + 1
}

// UpperCase returns s in upper case
func UpperCase(s string) string { return strings.ToUpper(s) }

// LowerCase returns s in lower case
func LowerCase(s string) string { return strings.ToLower(s) }

// PadStr returns s padded to length characters with fill (default a space) on the right, or
// cut to length. A negative length pads on the left (right-aligned).
func PadStr(s string, length int, fill ...string) string {
	pad := " "
	if len(fill) > 0 && fill[0] != "" {
		pad = string([]rune(fill[0])[0])
	}
	left := length < 0
	if left {
		length = -length
	}
	runes := []rune(s)
	if len(runes) >= length {
		return string(runes[:length])
	}
	padding := strings.Repeat(pad, length-len(runes))
	if left {
		return padding + s
	}
	return s + padding
}

// DelChr deletes the characters of which from s: where "=" (default) everywhere, "<" at the
// start, ">" at the end — combinable ("<>"). which defaults to a space.
func DelChr(s string, args ...string) string {
	where, which := "=", " "
	if len(args) > 0 && args[0] != "" {
		where = args[0]
	}
	if len(args) > 1 {
		which = args[1]
	}
	if which == "" {
		return s
	}
	if strings.Contains(where, "=") {
		return strings.Map(func(r rune) rune {
			if strings.ContainsRune(which, r) {
				return -1
			}
			return r
		}, s)
	}
	if strings.Contains(where, "<") {
		s = strings.TrimLeft(s, which)
	}
	if strings.Contains(where, ">") {
		s = strings.TrimRight(s, which)
	}
	return s
}

// SelectStr returns the n-th (1-based) item of a comma-separated string, "" when there is
// no such item
func SelectStr(n int, s string) string {
	items := strings.Split(s, ",")
	if n < 1 || n > len(items) {
		return ""
	}
	return items[n-1]
}

// IncStr increases the last number in s by one, keeping its width ("INV0099" → "INV0100",
// "A9" → "A10"); "" when s has no number
func IncStr(s string) string {
	runes := []rune(s)
	end := len(runes)
	for end > 0 && !unicode.IsDigit(runes[end-1]) {
		end--
	}
	if end == 0 {
		return ""
	}
	start := end
	for start > 0 && unicode.IsDigit(runes[start-1]) {
		start--
	}
	digits := string(runes[start:end])
	n, err := strconv.ParseUint(digits, 10, 64)
	if err != nil {
		return ""
	}
	next := strconv.FormatUint(n+1, 10)
	if len(next) < len(digits) {
		next = strings.Repeat("0", len(digits)-len(next)) + next
	}
	return string(runes[:start]) + next + string(runes[end:])
}

// StrSubstNo replaces %1 … %9 (and #1, #2 … followed by any number of #) in s with the
// values, formatted with Format. A placeholder without a value becomes "".
func StrSubstNo(s string, values ...any) string {
	var b strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if (r == '%' || r == '#') && i+1 < len(runes) && runes[i+1] >= '1' && runes[i+1] <= '9' {
			j := i + 1
			for j < len(runes) && unicode.IsDigit(runes[j]) {
				j++
			}
			n, _ := strconv.Atoi(string(runes[i+1 : j]))
			if r == '#' {
				for j < len(runes) && runes[j] == '#' {
					j++
				}
			}
			if n >= 1 && n <= len(values) {
				b.WriteString(Format(values[n-1]))
			}
			i = j - 1
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ConvertStr replaces every character of s found in from with the character at the same
// position in to ("a-b", "-", "_" → "a_b"); s is returned unchanged when from and to do not
// have the same length
func ConvertStr(s, from, to string) string {
	f, t := []rune(from), []rune(to)
	if len(f) != len(t) || len(f) == 0 {
		return s
	}
	return strings.Map(func(r rune) rune {
		for i, c := range f {
			if c == r {
				return t[i]
			}
		}
		return r
	}, s)
}

// InsStr inserts sub into s before position (1-based); past the end it is appended
func InsStr(s, sub string, position int) string {
	runes := []rune(s)
	i := min(max(position-1, 0), len(runes))
	return string(runes[:i]) + sub + string(runes[i:])
}

// DelStr deletes length characters (default: the rest) of s from position (1-based)
func DelStr(s string, position int, length ...int) string {
	runes := []rune(s)
	start := max(position-1, 0)
	if start >= len(runes) {
		return s
	}
	end := len(runes)
	if len(length) > 0 {
		end = min(start+max(length[0], 0), len(runes))
	}
	return string(runes[:start]) + string(runes[end:])
}

// StrCheckSum returns the check digit of a number string: the weighted sum of its digits
// (weights default 1 each; a shorter weight string is padded with 1) taken modulo modulus
// (default 10), subtracted from modulus — (modulus - sum % modulus) % modulus. Digits only.
func StrCheckSum(s string, weights string, modulus ...int) (int, error) {
	mod := 10
	if len(modulus) > 0 && modulus[0] > 0 {
		mod = modulus[0]
	}
	if len(weights) > len(s) {
		return 0, fmt.Errorf("StrCheckSum: the weight string is longer than %q", s)
	}
	weights += strings.Repeat("1", len(s)-len(weights))
	sum := 0
	for i := range len(s) {
		d, w := s[i], weights[i]
		if d < '0' || d > '9' || w < '0' || w > '9' {
			return 0, fmt.Errorf("StrCheckSum: %q and %q must be digits", s, weights)
		}
		sum += int(d-'0') * int(w-'0')
	}
	return (mod - sum%mod) % mod, nil
}
