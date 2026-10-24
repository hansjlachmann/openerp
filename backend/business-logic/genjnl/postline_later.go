package genjnl

import (
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// Customer/vendor posting and sales tax come in the next steps of this branch

func (p *GenJnlPostLine) postCust(genJnlLine *gtables.GenJournalLine) error {
	return apperrors.JnlCustVendNotYet(jnlLine, "account_type", genJnlLine.Account_type)
}

func (p *GenJnlPostLine) postVend(genJnlLine *gtables.GenJournalLine) error {
	return apperrors.JnlCustVendNotYet(jnlLine, "account_type", genJnlLine.Account_type)
}

func (p *GenJnlPostLine) initTax(genJnlLine *gtables.GenJournalLine) error {
	if genJnlLine.Gen_posting_type == gtables.GenJournalLine_Gen_posting_type.Blank {
		return nil
	}
	return apperrors.JnlTaxNotYet(jnlLine, "tax_calculation_type", genJnlLine.Tax_calculation_type)
}

func (p *GenJnlPostLine) postTax(genJnlLine *gtables.GenJournalLine) error { return nil }
func (p *GenJnlPostLine) initTaxRegister() error                           { return nil }
func (p *GenJnlPostLine) finishTaxRegister()                               {}
func (p *GenJnlPostLine) newTransaction()                                  {}
func (p *GenJnlPostLine) initGainsLosses()                                 {}
func (p *GenJnlPostLine) postGainsLosses() error                           { return nil }
