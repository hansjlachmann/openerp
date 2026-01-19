package filters

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ParseBCFilter parses a Business Central/NAV style filter expression into SQL WHERE clause
// Supports: *, |, .., <, >, <=, >=, <>
// Examples:
//   - "1000" -> field = '1000'
//   - "*son" -> field LIKE '%son'
//   - "A*" -> field LIKE 'A%'
//   - "1000|2000|3000" -> field IN ('1000', '2000', '3000')
//   - "1000..2000" -> field BETWEEN '1000' AND '2000'
//   - ">1000" -> field > '1000'
//   - "<1000|>5000" -> (field < '1000' OR field > '5000')
func ParseBCFilter(field, expression string) (string, []interface{}, error) {
	if expression == "" {
		return "", nil, nil
	}

	expression = strings.TrimSpace(expression)

	// Handle OR operator (|) - split and process each part
	if strings.Contains(expression, "|") {
		parts := strings.Split(expression, "|")
		var conditions []string
		var args []interface{}

		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			cond, partArgs, err := parseSingleFilter(field, part)
			if err != nil {
				return "", nil, err
			}

			conditions = append(conditions, cond)
			args = append(args, partArgs...)
		}

		if len(conditions) == 0 {
			return "", nil, nil
		}

		return "(" + strings.Join(conditions, " OR ") + ")", args, nil
	}

	// Single filter (no OR)
	return parseSingleFilter(field, expression)
}

func parseSingleFilter(field, expression string) (string, []interface{}, error) {
	expression = strings.TrimSpace(expression)

	// Handle range operator (..)
	if strings.Contains(expression, "..") {
		parts := strings.Split(expression, "..")
		if len(parts) != 2 {
			return "", nil, fmt.Errorf("invalid range expression: %s", expression)
		}

		from := strings.TrimSpace(parts[0])
		to := strings.TrimSpace(parts[1])

		return fmt.Sprintf("%s BETWEEN ? AND ?", field), []interface{}{from, to}, nil
	}

	// Handle comparison operators
	if strings.HasPrefix(expression, ">=") {
		value := strings.TrimSpace(expression[2:])
		return fmt.Sprintf("%s >= ?", field), []interface{}{value}, nil
	}
	if strings.HasPrefix(expression, "<=") {
		value := strings.TrimSpace(expression[2:])
		return fmt.Sprintf("%s <= ?", field), []interface{}{value}, nil
	}
	if strings.HasPrefix(expression, "<>") {
		value := strings.TrimSpace(expression[2:])
		return fmt.Sprintf("%s <> ?", field), []interface{}{value}, nil
	}
	if strings.HasPrefix(expression, ">") {
		value := strings.TrimSpace(expression[1:])
		return fmt.Sprintf("%s > ?", field), []interface{}{value}, nil
	}
	if strings.HasPrefix(expression, "<") {
		value := strings.TrimSpace(expression[1:])
		return fmt.Sprintf("%s < ?", field), []interface{}{value}, nil
	}

	// Handle wildcards (* and ?)
	if strings.Contains(expression, "*") || strings.Contains(expression, "?") {
		// Convert BC wildcards to SQL LIKE pattern
		pattern := expression
		pattern = strings.ReplaceAll(pattern, "*", "%")
		pattern = strings.ReplaceAll(pattern, "?", "_")

		return fmt.Sprintf("%s LIKE ?", field), []interface{}{pattern}, nil
	}

	// Exact match
	return fmt.Sprintf("%s = ?", field), []interface{}{expression}, nil
}

// BuildFilterClause builds a complete WHERE clause from multiple filters
func BuildFilterClause(filters []FilterExpression) (string, []interface{}, error) {
	if len(filters) == 0 {
		return "", nil, nil
	}

	var conditions []string
	var args []interface{}

	for _, filter := range filters {
		if filter.Expression == "" {
			continue
		}

		cond, filterArgs, err := ParseBCFilter(filter.Field, filter.Expression)
		if err != nil {
			return "", nil, fmt.Errorf("error parsing filter for field %s: %w", filter.Field, err)
		}

		if cond != "" {
			conditions = append(conditions, cond)
			args = append(args, filterArgs...)
		}
	}

	if len(conditions) == 0 {
		return "", nil, nil
	}

	return strings.Join(conditions, " AND "), args, nil
}

// FilterExpression represents a single field filter
type FilterExpression struct {
	Field      string
	Expression string
}

// IsNumeric checks if a string is numeric (for proper comparison handling)
func IsNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// SanitizeFieldName ensures the field name is safe for SQL (prevents injection)
func SanitizeFieldName(field string) string {
	// Only allow alphanumeric, underscore, and dot (for table.field)
	re := regexp.MustCompile(`[^a-zA-Z0-9_.]`)
	return re.ReplaceAllString(field, "")
}
