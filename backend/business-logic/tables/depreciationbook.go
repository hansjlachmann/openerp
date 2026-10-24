package tables

import (
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/system"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// depreciationBookLogic is the business logic of table 5611 Depreciation Book.
//
// Not migrated yet (their tables come with posting): OnInsert creates the book's FA Posting
// Type Setup lines (5604); OnDelete deletes them and the FA Journal Setup lines (5605).
type depreciationBookLogic struct{}

func init() { gtables.RegisterDepreciationBookLogic(depreciationBookLogic{}) }

// OnModify sets Last Date Modified
func (depreciationBookLogic) OnModify(rec *gtables.DepreciationBook) error {
	rec.Last_date_modified = system.Today()
	return nil
}

// OnRename sets Last Date Modified
func (depreciationBookLogic) OnRename(rec *gtables.DepreciationBook) error {
	rec.Last_date_modified = system.Today()
	return nil
}

// OnDelete: not while fixed assets use the book, nor while it is the insurance book with
// coverage entries
func (depreciationBookLogic) OnDelete(rec *gtables.DepreciationBook) error {
	faSetup := gtables.GetFASetup(rec)
	fADeprBook := gtables.NewFADepreciationBook(rec)
	fADeprBook.SetCurrentKey("depreciation_book_code")
	fADeprBook.SetRange("depreciation_book_code", rec.Code)
	if fADeprBook.FindFirst() {
		return apperrors.DeprBookInUse()
	}
	insCoverageLedgEntry := gtables.NewInsCoverageLedgerEntry(rec)
	if insCoverageLedgEntry.FindFirst() && faSetup.Insurance_depr_book == rec.Code {
		return apperrors.DeprBookInsurance(gtables.FASetupTableName, "insurance_depr_book", rec.Code.String())
	}
	return nil
}

// OnValidate_Use_fa_exch_rate_in_duplic: without it there is no default exchange rate
func (depreciationBookLogic) OnValidate_Use_fa_exch_rate_in_duplic(rec *gtables.DepreciationBook) error {
	if !rec.Use_fa_exch_rate_in_duplic {
		rec.Default_exchange_rate = types.ZeroDecimal()
	}
	return nil
}

// OnValidate_Default_exchange_rate: only with Use FA Exch. Rate in Duplic.
func (depreciationBookLogic) OnValidate_Default_exchange_rate(rec *gtables.DepreciationBook) error {
	if rec.Default_exchange_rate.IsPositive() {
		return rec.TestField("use_fa_exch_rate_in_duplic", true)
	}
	return nil
}
