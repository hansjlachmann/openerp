package reports

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hansjlachmann/openerp/backend/foundation/i18n"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
)

// numberFormat: the thousands and decimal separators of a language on printouts
type numberFormat struct {
	thousands string
	decimal   string
	date      string // Go layout of a date
}

var numberFormats = map[string]numberFormat{
	"en-US": {",", ".", "01/02/2006"},
	"nb-NO": {" ", ",", "02.01.2006"},
	"da-DK": {".", ",", "02.01.2006"},
}

// formatOf is the language's format (en-US for languages without one)
func formatOf(language string) numberFormat {
	f, ok := numberFormats[language]
	if !ok {
		return numberFormats["en-US"]
	}
	return f
}

// formatDecimal prints a decimal in the language with decimal places "min:max" (blank: 0:5),
// rounded half away from zero at max
func formatDecimal(d types.Decimal, places string, language string) string {
	min, max := 0, 5
	minText, maxText, ok := strings.Cut(places, ":")
	if ok {
		min, _ = strconv.Atoi(minText)
		max, _ = strconv.Atoi(maxText)
	}
	text := d.Round(int32(max)).StringFixed(int32(max))
	sign := ""
	if strings.HasPrefix(text, "-") {
		sign, text = "-", text[1:]
	}
	intPart, frac, _ := strings.Cut(text, ".")
	frac = strings.TrimRight(frac, "0")
	for len(frac) < min {
		frac += "0"
	}
	if strings.Trim(intPart+frac, "0") == "" {
		sign = ""
	}
	f := formatOf(language)
	var grouped strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			grouped.WriteString(f.thousands)
		}
		grouped.WriteRune(c)
	}
	if frac == "" {
		return sign + grouped.String()
	}
	return sign + grouped.String() + f.decimal + frac
}

// formatDate prints a date in the language (blank for a blank date)
func formatDate(d types.Date, language string) string {
	if d.IsZero() {
		return ""
	}
	return d.Time.Format(formatOf(language).date)
}

// formatValue prints a report value: texts as they are, numbers and dates in the language,
// Yes/No
func formatValue(v interface{}, places string, language string) string {
	switch value := v.(type) {
	case nil:
		return ""
	case string:
		return value
	case types.Code:
		return value.String()
	case types.Text:
		return value.String()
	case types.Decimal:
		if places == "" {
			places = "2:2"
		}
		return formatDecimal(value, places, language)
	case types.Date:
		return formatDate(value, language)
	case int:
		return strconv.Itoa(value)
	case bool:
		key := "terms.no"
		if value {
			key = "terms.yes"
		}
		return i18n.GetInstance().CommonTranslation(key, language)
	}
	lines := linesOf(v)
	if lines != nil {
		return strings.Join(lines, "\n")
	}
	return fmt.Sprint(v)
}

// isSet: a value a section's or element's print_if takes as set — a text that is not blank, a
// number that is not 0, true, lines or a picture
func isSet(v interface{}) bool {
	switch value := v.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(value) != ""
	case types.Code:
		return value != ""
	case types.Text:
		return strings.TrimSpace(value.String()) != ""
	case types.Decimal:
		return !value.IsZero()
	case types.Date:
		return !value.IsZero()
	case int:
		return value != 0
	case bool:
		return value
	case []byte:
		return len(value) > 0
	}
	lines := linesOf(v)
	if lines != nil {
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				return true
			}
		}
		return false
	}
	return true
}

// isZero: a value blank_zero leaves out
func isZero(v interface{}) bool {
	switch value := v.(type) {
	case types.Decimal:
		return value.IsZero()
	case int:
		return value == 0
	}
	return false
}
