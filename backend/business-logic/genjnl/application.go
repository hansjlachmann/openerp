package genjnl

import (
	"strconv"

	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// gainsLosses is the gains-and-losses posting buffer of one currency (classic table 343): the
// exchange-rate differences of the applications of a line, posted to the currency's accounts
type gainsLosses struct {
	realGains, realLosses, unrealGains, unrealLosses types.Decimal
}

// resetApplicationTotals: a customer or vendor line starts its application totals
func (p *GenJnlPostLine) resetApplicationTotals() {
	p.totalPmtDiscLCY = types.Decimal{}
	p.totalApplnRoundingLCY = types.Decimal{}
	p.realExchRateGains = types.Decimal{}
	p.realExchRateLosses = types.Decimal{}
	p.unrealExchRateGains = types.Decimal{}
	p.unrealExchRateLosses = types.Decimal{}
}

// applicationLCY is what the line's application posts elsewhere than on the receivables or
// payables account: exchange-rate gains and losses, payment discount, application rounding
func (p *GenJnlPostLine) applicationLCY() types.Decimal {
	return p.realExchRateGains.Add(p.realExchRateLosses).Add(p.unrealExchRateGains).Add(p.unrealExchRateLosses).
		Add(p.totalPmtDiscLCY).Add(p.totalApplnRoundingLCY)
}

// initGainsLosses: each line starts with an empty gains-and-losses buffer
func (p *GenJnlPostLine) initGainsLosses() {
	p.gainsLossesBuffer = map[types.Code]*gainsLosses{}
	p.gainsLossesOrder = nil
}

func (p *GenJnlPostLine) gainsLossesOf(currencyCode types.Code) *gainsLosses {
	buffer, ok := p.gainsLossesBuffer[currencyCode]
	if !ok {
		buffer = &gainsLosses{}
		p.gainsLossesBuffer[currencyCode] = buffer
		p.gainsLossesOrder = append(p.gainsLossesOrder, currencyCode)
	}
	return buffer
}

// addGainsLosses records an application's exchange-rate differences: the realized one in the
// application's currency, the unrealized one in the entry's
func (p *GenJnlPostLine) addGainsLosses(currencyCode, entryCurrencyCode types.Code, deltaRealAmount, deltaUnrealAmount types.Decimal) {
	buffer := p.gainsLossesOf(currencyCode)
	if deltaRealAmount.IsPositive() {
		p.realExchRateGains = p.realExchRateGains.Sub(deltaRealAmount)
		buffer.realGains = buffer.realGains.Sub(deltaRealAmount)
	} else {
		p.realExchRateLosses = p.realExchRateLosses.Sub(deltaRealAmount)
		buffer.realLosses = buffer.realLosses.Sub(deltaRealAmount)
	}
	buffer = p.gainsLossesOf(entryCurrencyCode)
	if deltaUnrealAmount.IsNegative() {
		p.unrealExchRateGains = p.unrealExchRateGains.Sub(deltaUnrealAmount)
		buffer.unrealGains = buffer.unrealGains.Sub(deltaUnrealAmount)
	} else {
		p.unrealExchRateLosses = p.unrealExchRateLosses.Sub(deltaUnrealAmount)
		buffer.unrealLosses = buffer.unrealLosses.Sub(deltaUnrealAmount)
	}
}

// postGainsLosses posts the line's exchange-rate gains and losses per currency to the
// currency's gains and losses accounts — all as unrealized when a currency uses the same
// accounts for both (PostCurrencyGainsLosses)
func (p *GenJnlPostLine) postGainsLosses() error {
	for _, currencyCode := range p.gainsLossesOrder {
		buffer := p.gainsLossesBuffer[currencyCode]
		if buffer.realGains.IsZero() && buffer.realLosses.IsZero() && buffer.unrealGains.IsZero() && buffer.unrealLosses.IsZero() {
			continue
		}
		currency := gtables.NewCurrency(p.src)
		err := currency.GetOrError(currencyCode)
		if err != nil {
			return err
		}
		if currency.Realized_gains_acc == currency.Unrealized_gains_acc && currency.Realized_losses_acc == currency.Unrealized_losses_acc {
			buffer.unrealGains = buffer.unrealGains.Add(buffer.realGains)
			buffer.unrealLosses = buffer.unrealLosses.Add(buffer.realLosses)
			buffer.realGains = types.Decimal{}
			buffer.realLosses = types.Decimal{}
		}
		postings := []struct {
			amount  types.Decimal
			field   string
			account types.Code
		}{
			{buffer.unrealGains, "unrealized_gains_acc", currency.Unrealized_gains_acc},
			{buffer.unrealLosses, "unrealized_losses_acc", currency.Unrealized_losses_acc},
			{buffer.realGains, "realized_gains_acc", currency.Realized_gains_acc},
			{buffer.realLosses, "realized_losses_acc", currency.Realized_losses_acc},
		}
		for _, posting := range postings {
			if posting.amount.IsZero() {
				continue
			}
			err = currency.TestField(posting.field)
			if err != nil {
				return err
			}
			_, err = p.initGLEntry(posting.account, posting.amount, true)
			if err != nil {
				return err
			}
			err = p.insertGLEntry()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// testPostingType: a customer line cannot post a purchase, a vendor line not a sale
func (p *GenJnlPostLine) testPostingType() error {
	switch {
	case p.custPosting && p.purchPostingType:
		return apperrors.JnlCustPurchaseType()
	case p.vendPosting && p.salesPostingType:
		return apperrors.JnlVendSalesType()
	}
	return nil
}

// newTransaction: a new transaction starts without customer, vendor or sales/purchase postings
func (p *GenJnlPostLine) newTransaction() {
	p.custPosting = false
	p.vendPosting = false
	p.salesPostingType = false
	p.purchPostingType = false
	p.firstNewTaxEntryNo = p.nextTaxEntryNo
}

// applnRoundingPrecision is the application rounding precision of a currency (the local
// currency's from the General Ledger Setup)
func (p *GenJnlPostLine) applnRoundingPrecision(currencyCode types.Code) (types.Decimal, error) {
	if currencyCode == "" {
		return p.glSetup.Appln_rounding_precision, nil
	}
	currency3 := gtables.NewCurrency(p.src)
	err := currency3.GetOrError(currencyCode)
	if err != nil {
		return types.Decimal{}, err
	}
	return currency3.Appln_rounding_precision, nil
}

// CheckCurrency: the entries of one application are in one currency (application between
// currencies is not supported yet)
func CheckCurrency(applnCurrencyCode, compareCurrencyCode types.Code) error {
	if applnCurrencyCode == compareCurrencyCode {
		return nil
	}
	return apperrors.JnlApplnSameCurrency()
}

func itoa(n int) string { return strconv.Itoa(n) }
