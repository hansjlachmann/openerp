package genjnl

import (
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// Sales tax comes in the next step of this branch

func (p *GenJnlPostLine) initTax(genJnlLine *gtables.GenJournalLine) error {
	if genJnlLine.Gen_posting_type == gtables.GenJournalLine_Gen_posting_type.Blank {
		return nil
	}
	return apperrors.JnlTaxNotYet(jnlLine, "tax_calculation_type", genJnlLine.Tax_calculation_type)
}

func (p *GenJnlPostLine) postTax(genJnlLine *gtables.GenJournalLine) error { return nil }
func (p *GenJnlPostLine) initTaxRegister() error                           { return nil }
func (p *GenJnlPostLine) finishTaxRegister()                               {}
