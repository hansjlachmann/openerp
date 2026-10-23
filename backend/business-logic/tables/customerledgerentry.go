package tables

import (
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// customerLedgerEntryLogic is the business logic of table 21 Customer Ledger Entry
type customerLedgerEntryLogic struct{}

func init() { gtables.RegisterCustomerLedgerEntryLogic(customerLedgerEntryLogic{}) }

// SetDebitCredit fills Debit/Credit Amount (and LCY) from Amount and Amount (LCY): a positive
// amount is a debit, a negative one a credit (shown positive) — as posting does
func SetDebitCredit(e *gtables.CustomerLedgerEntry) {
	zero := types.ZeroDecimal()
	e.Debit_amount, e.Credit_amount = zero, zero
	e.Debit_amount_lcy, e.Credit_amount_lcy = zero, zero
	if e.Amount.IsPositive() {
		e.Debit_amount = e.Amount
	} else {
		e.Credit_amount = e.Amount.Abs()
	}
	if e.Amount_lcy.IsPositive() {
		e.Debit_amount_lcy = e.Amount_lcy
	} else {
		e.Credit_amount_lcy = e.Amount_lcy.Abs()
	}
}
