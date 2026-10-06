// Package flowfilter turns BC/NAV FlowFilter expressions into SQL conditions.
//
// A FlowFilter field (NAV FieldClass FlowFilter) is not stored: it holds a filter the user
// sets, e.g. a Date Filter "2026-01-01..2026-03-31", and FlowFields apply it to a field of
// their source table (Posting Date = FIELD(Date Filter)). Clause builds the SQL condition
// for one column from such an expression.
//
// Syntax (BC): alternatives separated by '|'; each alternative is a value, a range "a..b",
// an open range "..b" or "a..", or "<>value". Text and Code accept '*' wildcards. Booleans
// accept Yes/No, true/false, 1/0. Dates are ISO (YYYY-MM-DD) — the frontend converts the
// user's local date format before sending.
package flowfilter

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Kind is the data type of a FlowFilter field.
type Kind string

const (
	KindDate Kind = "date"
	KindBool Kind = "bool"
	KindInt  Kind = "int"
	KindText Kind = "text"
	KindCode Kind = "code"
)

// Clause returns the SQL condition (with ? placeholders) and its arguments that apply expr
// to column. An empty expression means no filter: "" and no arguments.
func Clause(column string, kind Kind, expr string) (string, []interface{}, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", nil, nil
	}
	var ors []string
	var args []interface{}
	for _, part := range strings.Split(expr, "|") {
		part = strings.TrimSpace(part)
		if part == "" {
			return "", nil, fmt.Errorf("empty value in filter %q", expr)
		}
		cond, a, err := alternative(column, kind, part)
		if err != nil {
			return "", nil, err
		}
		ors = append(ors, cond)
		args = append(args, a...)
	}
	if len(ors) == 1 {
		return ors[0], args, nil
	}
	return "(" + strings.Join(ors, " OR ") + ")", args, nil
}

// Validate reports whether expr is a valid filter for kind.
func Validate(kind Kind, expr string) error {
	_, _, err := Clause("x", kind, expr)
	return err
}

func alternative(column string, kind Kind, part string) (string, []interface{}, error) {
	if rest, ok := strings.CutPrefix(part, "<>"); ok {
		v, err := value(kind, strings.TrimSpace(rest))
		if err != nil {
			return "", nil, err
		}
		return column + " <> ?", []interface{}{v}, nil
	}

	if kind != KindBool {
		if from, to, isRange := strings.Cut(part, ".."); isRange {
			from, to = strings.TrimSpace(from), strings.TrimSpace(to)
			switch {
			case from == "" && to == "":
				return "", nil, fmt.Errorf("range %q has no start or end", part)
			case from == "":
				v, err := value(kind, to)
				if err != nil {
					return "", nil, err
				}
				return column + " <= ?", []interface{}{v}, nil
			case to == "":
				v, err := value(kind, from)
				if err != nil {
					return "", nil, err
				}
				return column + " >= ?", []interface{}{v}, nil
			default:
				a, err := value(kind, from)
				if err != nil {
					return "", nil, err
				}
				b, err := value(kind, to)
				if err != nil {
					return "", nil, err
				}
				return column + " BETWEEN ? AND ?", []interface{}{a, b}, nil
			}
		}
	}

	if (kind == KindText || kind == KindCode) && strings.Contains(part, "*") {
		pattern := part
		if kind == KindCode {
			pattern = strings.ToUpper(pattern)
		}
		escaped := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(pattern)
		return column + ` LIKE ? ESCAPE '\'`, []interface{}{strings.ReplaceAll(escaped, "*", "%")}, nil
	}

	v, err := value(kind, part)
	if err != nil {
		return "", nil, err
	}
	return column + " = ?", []interface{}{v}, nil
}

// value parses one filter value of the given kind into its SQL argument.
func value(kind Kind, s string) (interface{}, error) {
	if s == "" {
		return nil, fmt.Errorf("missing value")
	}
	switch kind {
	case KindDate:
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			return nil, fmt.Errorf("%q is not a date", s)
		}
		return d.Format("2006-01-02"), nil
	case KindInt:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%q is not a whole number", s)
		}
		return n, nil
	case KindBool:
		switch strings.ToLower(s) {
		case "yes", "true", "1", "ja":
			return true, nil
		case "no", "false", "0", "nei", "nej":
			return false, nil
		}
		return nil, fmt.Errorf("%q is not Yes or No", s)
	case KindCode:
		return strings.ToUpper(s), nil
	default:
		return s, nil
	}
}
