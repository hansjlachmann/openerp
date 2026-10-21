package tables

import (
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/system"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// currencyLogic is the business logic of table 4 Currency.
// Add a trigger when it is needed: the available triggers are listed at the
// top of generated/tables/currency_gen.go. The field rules of the definition
// (required, length, range) are checked after OnInsert and OnModify.
//
// Not migrated yet (their tables come later): OnDelete deletes the currency's exchange rates
// (Currency Exchange Rate), InitRoundingPrecision takes the precisions from the General
// Ledger Setup, the G/L account fields and their checks.
type currencyLogic struct{}

func init() { gtables.RegisterCurrencyLogic(currencyLogic{}) }

// OnInsert runs before a new record is inserted
func (currencyLogic) OnInsert(rec *gtables.Currency) error {
	return nil
}

// OnModify runs before a record is modified
func (currencyLogic) OnModify(rec *gtables.Currency) error {
	rec.Last_date_modified = system.Today()
	return nil
}

// OnDelete runs before a record is deleted
func (currencyLogic) OnDelete(rec *gtables.Currency) error {
	return nil
}

// OnRename runs before a primary key field is changed (rec.OldValue has the old value)
func (currencyLogic) OnRename(rec *gtables.Currency) error {
	rec.Last_date_modified = system.Today()
	return nil
}

// OnValidate_Invoice_rounding_precision: the invoice rounding precision must be a multiple of
// the amount rounding precision
func (currencyLogic) OnValidate_Invoice_rounding_precision(rec *gtables.Currency) error {
	if !rec.Amount_rounding_precision.IsZero() &&
		!rec.Invoice_rounding_precision.Equal(system.Round(rec.Invoice_rounding_precision, rec.Amount_rounding_precision)) {
		return apperrors.MustBeRoundedTo(gtables.CurrencyTableName, "invoice_rounding_precision",
			system.Format(rec.Amount_rounding_precision))
	}
	return nil
}

// OnValidate_Amount_rounding_precision rounds the invoice rounding precision to the new
// amount rounding precision
func (currencyLogic) OnValidate_Amount_rounding_precision(rec *gtables.Currency) error {
	if !rec.Amount_rounding_precision.IsZero() {
		rec.Invoice_rounding_precision = system.Round(rec.Invoice_rounding_precision, rec.Amount_rounding_precision)
	}
	return nil
}

// OnValidate_Amount_decimal_places checks and tidies the format ("2:2")
func (currencyLogic) OnValidate_Amount_decimal_places(rec *gtables.Currency) error {
	return CheckDecimalPlacesFormat(gtables.CurrencyTableName, "amount_decimal_places", &rec.Amount_decimal_places)
}

// OnValidate_Unit_amount_decimal_places checks and tidies the format ("2:5")
func (currencyLogic) OnValidate_Unit_amount_decimal_places(rec *gtables.Currency) error {
	return CheckDecimalPlacesFormat(gtables.CurrencyTableName, "unit_amount_decimal_places", &rec.Unit_amount_decimal_places)
}
