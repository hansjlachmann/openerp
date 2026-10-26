package tables

import (
	"github.com/hansjlachmann/openerp/backend/business-logic/noseries"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/functions"
	"github.com/hansjlachmann/openerp/backend/foundation/system"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// itemLogic is the business logic of table 27 Item.
//
// Not migrated yet (their tables come later): OnDelete's checks of outstanding purchase, sales
// and production order lines and its deletion of extended texts; the
// Costing Method's check of purchase order lines; the Standard Cost's confirmation (no dialogs
// in table triggers yet); the additional reporting currency (Add.-Curr. Average Cost).
type itemLogic struct{}

func init() {
	gtables.RegisterItemLogic(itemLogic{})
	// the No. field's "…" (Item Card): pick another number series
	functions.Register("table", gtables.ItemTableName, "AssistEdit", functions.ForAssistEdit(AssistEditItem), functions.WithAssistEdit())
}

// OnInsert: an item without No. gets the next number of the Item Nos. series of the Inventory
// Setup
func (itemLogic) OnInsert(rec *gtables.Item) error {
	if rec.No == "" {
		invtSetup := gtables.GetInventorySetup(rec)
		err := invtSetup.TestField("item_nos")
		if err != nil {
			return err
		}
		noSeriesMgt := noseries.New(rec)
		xRec := rec.XRec()
		err = noSeriesMgt.InitSeries(invtSetup.Item_nos, xRec.No_series, types.Date{}, &rec.No, &rec.No_series)
		if err != nil {
			return err
		}
		rec.Sales_qty_disc_code = rec.No
	}
	return nil
}

// OnModify sets Last Date Modified
func (itemLogic) OnModify(rec *gtables.Item) error {
	rec.Last_date_modified = system.Today()
	return nil
}

// OnRename sets Last Date Modified
func (itemLogic) OnRename(rec *gtables.Item) error {
	rec.Last_date_modified = system.Today()
	return nil
}

// OnDelete deletes the item's comment lines, vendors, prices, translations, units of measure,
// variants and BOM components
func (itemLogic) OnDelete(rec *gtables.Item) error {
	DeleteComments(rec, gtables.CommentLine_Table_name.Item, rec.No)

	itemVend := gtables.NewItemVendor(rec)
	itemVend.SetRange("item_no", rec.No)
	itemVend.DeleteAll()

	itemPrice := gtables.NewItemPrice(rec)
	itemPrice.SetRange("item_no", rec.No)
	itemPrice.DeleteAll()

	itemTranslation := gtables.NewItemTranslation(rec)
	itemTranslation.SetRange("item_no", rec.No)
	itemTranslation.DeleteAll()

	itemUnitOfMeasure := gtables.NewItemUnitOfMeasure(rec)
	itemUnitOfMeasure.SetRange("item_no", rec.No)
	itemUnitOfMeasure.DeleteAll()

	itemVariant := gtables.NewItemVariant(rec)
	itemVariant.SetRange("item_no", rec.No)
	itemVariant.DeleteAll()

	bomComp := gtables.NewBOMComponent(rec)
	bomComp.SetRange("parent_item_no", rec.No)
	bomComp.DeleteAll()
	return nil
}

// OnValidate_No: a number typed by hand must be allowed by the Item Nos. series (Manual Nos.)
// and takes the item out of its series; a new item saved with the number its series just
// issued (AssistEdit) keeps that series. Sales Qty. Disc. Code follows the No.
func (itemLogic) OnValidate_No(rec *gtables.Item) error {
	xRec := rec.XRec()
	if rec.No != xRec.No {
		invtSetup := gtables.GetInventorySetup(rec)
		noSeriesMgt := noseries.New(rec)
		series, issued := noSeriesMgt.IssuedBy(invtSetup.Item_nos, rec.No)
		if issued && xRec.No == "" {
			rec.No_series = series
		} else {
			err := noSeriesMgt.TestManual(invtSetup.Item_nos)
			if err != nil {
				return err
			}
			rec.No_series = ""
		}
	}
	rec.Sales_qty_disc_code = rec.No
	return nil
}

// OnValidate_Description: Search Description follows the description while it was blank or the
// old description
func (itemLogic) OnValidate_Description(rec *gtables.Item) error {
	xRec := rec.XRec()
	if string(rec.Search_description) == system.UpperCase(xRec.Description.String()) || rec.Search_description == "" {
		rec.Search_description = types.NewCode(system.CopyStr(rec.Description.String(), 1, 50))
	}
	return nil
}

// OnValidate_Base_unit_of_measure: the sales and purchase units follow; the base unit is one of
// the item's units of measure with quantity 1
func (itemLogic) OnValidate_Base_unit_of_measure(rec *gtables.Item) error {
	rec.Sales_unit_of_measure = rec.Base_unit_of_measure
	rec.Purch_unit_of_measure = rec.Base_unit_of_measure
	if rec.Base_unit_of_measure != "" {
		itemUnitOfMeasure := gtables.NewItemUnitOfMeasure(rec)
		err := itemUnitOfMeasure.GetOrError(map[string]interface{}{"item_no": rec.No, "code": rec.Base_unit_of_measure})
		if err != nil {
			return err
		}
		err = itemUnitOfMeasure.TestField("qty_per_unit_of_measure", types.NewDecimalFromInt(1))
		if err != nil {
			return err
		}
	}
	return nil
}

// OnValidate_Unit_price: the price/profit calculation again
func (itemLogic) OnValidate_Unit_price(rec *gtables.Item) error {
	return validateItemPriceProfit(rec)
}

// OnValidate_Price_profit_calculation: Profit % from the price and cost, or the price from the
// cost and Profit % — with prices including tax, the tax of the item's tax posting setup for
// prices
func (itemLogic) OnValidate_Price_profit_calculation(rec *gtables.Item) error {
	taxPostingSetup := gtables.NewTaxPostingSetup(rec)
	if rec.Price_includes_tax && rec.Price_profit_calculation < gtables.Item_Price_profit_calculation.NoRelationship {
		if !taxPostingSetup.Get(map[string]interface{}{"tax_bus_posting_group": rec.Tax_bus_posting_gr_price, "tax_prod_posting_group": rec.Tax_prod_posting_group}) {
			taxPostingSetup.Init()
		}
		switch taxPostingSetup.Tax_calculation_type {
		case gtables.TaxPostingSetup_Tax_calculation_type.ReverseChargeVAT:
			taxPostingSetup.Vat_percent = types.Decimal{}
		case gtables.TaxPostingSetup_Tax_calculation_type.SalesTax:
			return apperrors.ItemPriceInclTaxSalesTax(gtables.TaxPostingSetupTableName, "tax_calculation_type", taxPostingSetup.Tax_calculation_type)
		}
	} else {
		taxPostingSetup.Init()
		taxPostingSetup.Vat_percent = types.Decimal{}
	}

	hundred := types.NewDecimalFromInt(100)
	one := types.NewDecimalFromInt(1)
	taxFactor := one.Add(taxPostingSetup.Vat_percent.Div(hundred))
	switch rec.Price_profit_calculation {
	case gtables.Item_Price_profit_calculation.ProfitPriceCost:
		if !rec.Unit_price.IsZero() {
			rec.Profit_percent = system.Round(hundred.Mul(one.Sub(rec.Unit_cost.Div(rec.Unit_price.Div(taxFactor)))), types.MustDecimal("0.00001"))
		} else {
			rec.Profit_percent = types.Decimal{}
		}
	case gtables.Item_Price_profit_calculation.PriceCostProfit:
		if rec.Profit_percent.LessThan(hundred) {
			rec.Unit_price = system.Round(rec.Unit_cost.Div(one.Sub(rec.Profit_percent.Div(hundred))).Mul(taxFactor), types.MustDecimal("0.00001"))
		}
	}
	return nil
}

func validateItemPriceProfit(rec *gtables.Item) error {
	return rec.Validate("price_profit_calculation", rec.Price_profit_calculation)
}

// OnValidate_Profit_percent: the price/profit calculation again
func (itemLogic) OnValidate_Profit_percent(rec *gtables.Item) error {
	return validateItemPriceProfit(rec)
}

// OnValidate_Costing_method: the unit cost follows the costing method
func (itemLogic) OnValidate_Costing_method(rec *gtables.Item) error {
	xRec := rec.XRec()
	if rec.Costing_method == xRec.Costing_method {
		return nil
	}
	return updateItemUnitCost(rec)
}

// OnValidate_Standard_cost: the unit cost of a standard cost item follows
func (itemLogic) OnValidate_Standard_cost(rec *gtables.Item) error {
	return updateItemUnitCost(rec)
}

// OnValidate_Average_cost: the unit cost of the other costing methods follows
func (itemLogic) OnValidate_Average_cost(rec *gtables.Item) error {
	return updateItemUnitCost(rec)
}

// OnValidate_Indirect_cost_percent: the unit cost again
func (itemLogic) OnValidate_Indirect_cost_percent(rec *gtables.Item) error {
	return updateItemUnitCost(rec)
}

// OnValidate_Price_includes_tax: the price/profit calculation again
func (itemLogic) OnValidate_Price_includes_tax(rec *gtables.Item) error {
	return validateItemPriceProfit(rec)
}

// OnValidate_Tax_bus_posting_gr_price: the price/profit calculation again
func (itemLogic) OnValidate_Tax_bus_posting_gr_price(rec *gtables.Item) error {
	return validateItemPriceProfit(rec)
}

// OnValidate_Gen_prod_posting_group fills in the group's default Tax Prod. Posting Group
func (itemLogic) OnValidate_Gen_prod_posting_group(rec *gtables.Item) error {
	xRec := rec.XRec()
	if xRec.Gen_prod_posting_group != rec.Gen_prod_posting_group {
		genProdPostingGroup := gtables.NewGenProductPostingGroup(rec)
		autoInsert, err := ValidateTaxProdPostingGroup(genProdPostingGroup, rec.Gen_prod_posting_group)
		if err != nil {
			return err
		}
		if autoInsert {
			err = rec.Validate("tax_prod_posting_group", genProdPostingGroup.Def_tax_prod_posting_group)
			if err != nil {
				return err
			}
		}
	}
	return validateItemPriceProfit(rec)
}

// OnValidate_Tax_prod_posting_group: the price/profit calculation again
func (itemLogic) OnValidate_Tax_prod_posting_group(rec *gtables.Item) error {
	return validateItemPriceProfit(rec)
}

// updateItemUnitCost (UpdateUnitCost): a standard cost item costs its standard cost (no indirect
// cost), the others their average cost
func updateItemUnitCost(rec *gtables.Item) error {
	if rec.Costing_method == gtables.Item_Costing_method.Standard {
		rec.Unit_cost = rec.Standard_cost
		rec.Indirect_cost_percent = types.Decimal{}
	} else {
		rec.Unit_cost = rec.Average_cost
	}
	return validateItemPriceProfit(rec)
}

// AssistEditItem lets the user pick another number series for the item — the Item Nos. series
// or one related to it — and gives the item that series' next number (AssistEdit of the No.
// field); false when the user cancelled
func AssistEditItem(rec *gtables.Item) (bool, error) {
	invtSetup := gtables.GetInventorySetup(rec)
	err := invtSetup.TestField("item_nos")
	if err != nil {
		return false, err
	}
	noSeriesMgt := noseries.New(rec)
	xRec := rec.XRec()
	noSeries := rec.No_series
	picked, err := noSeriesMgt.SelectSeries(invtSetup.Item_nos, xRec.No_series, &noSeries)
	if err != nil {
		return false, err
	}
	if picked {
		err = noSeriesMgt.SetSeries(&rec.No)
		if err != nil {
			return false, err
		}
		rec.No_series = noSeries
		rec.Sales_qty_disc_code = rec.No
	}
	return picked, nil
}
