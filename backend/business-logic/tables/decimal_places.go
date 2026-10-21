package tables

import (
	"strconv"
	"strings"

	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/system"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
)

// CheckDecimalPlacesFormat checks a decimal places setting — one number or two numbers
// separated by a colon, each 0–9, the first not above the second ("2", "2:5") — and writes it
// back tidied ("02 : 5" is refused, "02:5" becomes "2:5"). Used by the currencies' decimal
// places fields; it belongs to the General Ledger Setup once that table is migrated.
func CheckDecimalPlacesFormat(table, field string, value *types.Text) error {
	text := value.String()
	part := func(s string) (int, bool) {
		n, err := strconv.Atoi(s)
		return n, err == nil && s != "" && n >= 0 && n <= 9
	}
	min, max, colon := strings.Cut(text, ":")
	a, ok := part(min)
	if !colon {
		if !ok {
			return apperrors.DecimalPlacesFormat(table, field)
		}
		*value = types.NewText(system.Format(a))
		return nil
	}
	b, ok2 := part(max)
	if !ok || !ok2 || a > b {
		return apperrors.DecimalPlacesFormat(table, field)
	}
	*value = types.NewText(system.StrSubstNo("%1:%2", a, b))
	return nil
}
