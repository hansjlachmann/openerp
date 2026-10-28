// Package salespost is codeunit 80 Sales-Post. For now its SumSalesLines, the totals of a sales
// document the statistics show; the posting itself comes with the sales posting step.
package salespost

import (
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/system"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// QtyType is the quantity SumSalesLines totals ('General,Invoicing,Shipping'); quotes have only
// General (the lines' Quantity)
type QtyType int

const (
	General QtyType = iota
	Invoicing
	Shipping
)

// SalesTotals is what SumSalesLines returns: the totals of the lines in the document's currency
// (TotalSalesLine) and in LCY (TotalSalesLineLCY, with the cost in Unit Cost (LCY)), the tax,
// the profit in LCY and its percent
type SalesTotals struct {
	TotalSalesLine    gtables.SalesLine
	TotalSalesLineLCY gtables.SalesLine
	TaxAmount         types.Decimal
	ProfitLCY         types.Decimal
	ProfitPct         types.Decimal
}

// sumSales holds what SumSalesLines2 adds up (the codeunit's global variables)
type sumSales struct {
	salesHeader       *gtables.SalesHeader
	currency          *gtables.Currency
	totalSalesLine    gtables.SalesLine
	totalSalesLineLCY gtables.SalesLine
	orgTotalSalesLine gtables.SalesLine
}

// SumSalesLines (SumSalesLines): the document's totals — each line's share of its quantity
// (General: all of it), rounded so the rounding differences fall on the total, also converted
// to LCY at the document's factor; the profit is the LCY amount less the cost. Invoice rounding
// comes with posting.
func SumSalesLines(salesHeader *gtables.SalesHeader, qtyType QtyType) (SalesTotals, error) {
	s := &sumSales{salesHeader: salesHeader}
	err := s.sumSalesLines2(qtyType)
	if err != nil {
		return SalesTotals{}, err
	}
	var totals SalesTotals
	totals.ProfitLCY = s.totalSalesLineLCY.Amount.Sub(s.totalSalesLineLCY.Unit_cost_lcy)
	if s.totalSalesLineLCY.Amount.IsZero() {
		totals.ProfitPct = types.Decimal{}
	} else {
		totals.ProfitPct = system.Round(totals.ProfitLCY.Div(s.totalSalesLineLCY.Amount).Mul(types.NewDecimalFromInt(100)), types.MustDecimal("0.1"))
	}
	totals.TaxAmount = s.totalSalesLine.Amount_including_tax.Sub(s.totalSalesLine.Amount)
	totals.TotalSalesLine = s.totalSalesLine
	totals.TotalSalesLineLCY = s.totalSalesLineLCY
	return totals, nil
}

func (s *sumSales) sumSalesLines2(qtyType QtyType) error {
	s.currency = gtables.NewCurrency(s.salesHeader)
	if s.salesHeader.Currency_code == "" {
		tables.InitRoundingPrecision(s.currency)
	} else {
		err := s.currency.GetOrError(s.salesHeader.Currency_code)
		if err != nil {
			return err
		}
		err = s.currency.TestField("amount_rounding_precision")
		if err != nil {
			return err
		}
	}
	oldSalesLine := gtables.NewSalesLine(s.salesHeader)
	oldSalesLine.SetRange("document_type", s.salesHeader.Document_type)
	oldSalesLine.SetRange("document_no", s.salesHeader.No)
	for found := oldSalesLine.FindSet(); found; found = oldSalesLine.Next() {
		salesLine := *oldSalesLine
		var salesLineQty types.Decimal
		switch qtyType {
		case General:
			salesLineQty = salesLine.Quantity
		}
		if !salesLineQty.IsZero() {
			s.divideAmount(&salesLine, salesLineQty)
			if !salesLine.Amount.IsZero() {
				if s.totalSalesLine.Amount.IsZero() {
					s.totalSalesLine.Tax_percent = salesLine.Tax_percent
				} else {
					if !s.totalSalesLine.Tax_percent.Equal(salesLine.Tax_percent) {
						s.totalSalesLine.Tax_percent = types.Decimal{}
					}
				}
			}
			err := s.roundAmount(&salesLine, salesLineQty)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// divideAmount (DivideAmount): the line's amounts for a share of its quantity
func (s *sumSales) divideAmount(salesLine *gtables.SalesLine, salesLineQty types.Decimal) {
	share := func(value types.Decimal) types.Decimal {
		return value.Mul(salesLineQty).Div(salesLine.Quantity)
	}
	salesLine.Amount = share(salesLine.Amount)
	salesLine.Tax_base_amount = share(salesLine.Tax_base_amount)
	salesLine.Amount_including_tax = share(salesLine.Amount_including_tax)
	salesLine.Line_discount_amount = share(salesLine.Line_discount_amount)
	salesLine.Inv_discount_amount = share(salesLine.Inv_discount_amount)
}

// roundAmount (RoundAmount): the line rounded so the running totals stay exact, added to the
// totals with its quantity, weights, volume and parcels; in LCY with its cost
func (s *sumSales) roundAmount(salesLine *gtables.SalesLine, salesLineQty types.Decimal) error {
	precision := s.currency.Amount_rounding_precision
	incrAmount(&s.orgTotalSalesLine, salesLine)
	noTax := salesLine.Amount.Equal(salesLine.Amount_including_tax)
	salesLine.Amount_including_tax = system.Round(s.orgTotalSalesLine.Amount_including_tax, precision).Sub(s.totalSalesLine.Amount_including_tax)
	if noTax {
		salesLine.Amount = salesLine.Amount_including_tax
		salesLine.Tax_base_amount = salesLine.Amount_including_tax
	} else {
		salesLine.Amount = system.Round(s.orgTotalSalesLine.Amount, precision).Sub(s.totalSalesLine.Amount)
		salesLine.Tax_base_amount = system.Round(s.orgTotalSalesLine.Tax_base_amount, precision).Sub(s.totalSalesLine.Tax_base_amount)
	}
	salesLine.Line_discount_amount = system.Round(s.orgTotalSalesLine.Line_discount_amount, precision).Sub(s.totalSalesLine.Line_discount_amount)
	salesLine.Inv_discount_amount = system.Round(s.orgTotalSalesLine.Inv_discount_amount, precision).Sub(s.totalSalesLine.Inv_discount_amount)
	incrAmount(&s.totalSalesLine, salesLine)
	s.totalSalesLine.Net_weight = s.totalSalesLine.Net_weight.Add(system.Round(salesLineQty.Mul(salesLine.Net_weight), types.MustDecimal("0.00001")))
	s.totalSalesLine.Gross_weight = s.totalSalesLine.Gross_weight.Add(system.Round(salesLineQty.Mul(salesLine.Gross_weight), types.MustDecimal("0.00001")))
	s.totalSalesLine.Unit_volume = s.totalSalesLine.Unit_volume.Add(system.Round(salesLineQty.Mul(salesLine.Unit_volume), types.MustDecimal("0.00001")))
	s.totalSalesLine.Quantity = s.totalSalesLine.Quantity.Add(salesLineQty)
	if salesLine.Units_per_parcel.IsPositive() {
		s.totalSalesLine.Units_per_parcel = s.totalSalesLine.Units_per_parcel.Add(system.Round(salesLineQty.Div(salesLine.Units_per_parcel), types.NewDecimalFromInt(1), ">"))
	}
	if s.salesHeader.Currency_code != "" {
		var useDate types.Date
		if s.salesHeader.Document_type == gtables.SalesHeader_Document_type.Quote {
			useDate = system.WorkDate()
		} else {
			useDate = s.salesHeader.Posting_date
		}
		currExchRate := tables.ExchangeRates(s.salesHeader)
		toLCY := func(totalFCY, totalLCY types.Decimal) (types.Decimal, error) {
			amount, err := currExchRate.ExchangeAmtFCYToLCY(useDate, s.salesHeader.Currency_code, totalFCY, s.salesHeader.Currency_factor)
			if err != nil {
				return types.Decimal{}, err
			}
			return system.Round(amount).Sub(totalLCY), nil
		}
		noTax = salesLine.Amount.Equal(salesLine.Amount_including_tax)
		var err error
		salesLine.Amount_including_tax, err = toLCY(s.totalSalesLine.Amount_including_tax, s.totalSalesLineLCY.Amount_including_tax)
		if err != nil {
			return err
		}
		if noTax {
			salesLine.Amount = salesLine.Amount_including_tax
		} else {
			salesLine.Amount, err = toLCY(s.totalSalesLine.Amount, s.totalSalesLineLCY.Amount)
			if err != nil {
				return err
			}
		}
		salesLine.Line_discount_amount, err = toLCY(s.totalSalesLine.Line_discount_amount, s.totalSalesLineLCY.Line_discount_amount)
		if err != nil {
			return err
		}
		salesLine.Inv_discount_amount, err = toLCY(s.totalSalesLine.Inv_discount_amount, s.totalSalesLineLCY.Inv_discount_amount)
		if err != nil {
			return err
		}
	}
	incrAmount(&s.totalSalesLineLCY, salesLine)
	s.totalSalesLineLCY.Unit_cost_lcy = s.totalSalesLineLCY.Unit_cost_lcy.Add(system.Round(salesLineQty.Mul(salesLine.Unit_cost_lcy)))
	return nil
}

// incrAmount (IncrAmount): a line's amounts added to a total
func incrAmount(totalSalesLine *gtables.SalesLine, salesLine *gtables.SalesLine) {
	totalSalesLine.Amount = totalSalesLine.Amount.Add(salesLine.Amount)
	totalSalesLine.Tax_base_amount = totalSalesLine.Tax_base_amount.Add(salesLine.Tax_base_amount)
	totalSalesLine.Amount_including_tax = totalSalesLine.Amount_including_tax.Add(salesLine.Amount_including_tax)
	totalSalesLine.Line_discount_amount = totalSalesLine.Line_discount_amount.Add(salesLine.Line_discount_amount)
	totalSalesLine.Inv_discount_amount = totalSalesLine.Inv_discount_amount.Add(salesLine.Inv_discount_amount)
}
