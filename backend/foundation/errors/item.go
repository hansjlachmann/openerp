package errors

// Items (table 27 and the tables around it)
const (
	ErrItemUoMQtyNotPositive    ErrorCode = "ERR_ITEM_UOM_QTY_NOT_POSITIVE"
	ErrItemPriceInclTaxSalesTax ErrorCode = "ERR_ITEM_PRICE_INCL_TAX_SALES_TAX"
	ErrItemOpenEntriesChange    ErrorCode = "ERR_ITEM_OPEN_ENTRIES_CHANGE"
	ErrItemHasEntriesChange     ErrorCode = "ERR_ITEM_HAS_ENTRIES_CHANGE"
)

// ItemUoMQtyNotPositive: Qty. per Unit of Measure must be greater than 0
func ItemUoMQtyNotPositive(table, field string) *AppError {
	return &AppError{Code: ErrItemUoMQtyNotPositive, Table: table, Field: field}
}

// ItemPriceInclTaxSalesTax: prices including tax cannot be calculated for a tax posting setup
// with Tax Calculation Type Sales Tax (value)
func ItemPriceInclTaxSalesTax(table, field string, value any) *AppError {
	return &AppError{Code: ErrItemPriceInclTaxSalesTax, Table: table, Field: field, Value: value}
}

// ItemOpenEntriesChange: a field of an item (no.) cannot change while it has open ledger entries
func ItemOpenEntriesChange(table, field, no string) *AppError {
	return &AppError{Code: ErrItemOpenEntriesChange, Table: table, Field: field, Params: []string{no}}
}

// ItemHasEntriesChange: a field of an item cannot change once it has ledger entries
func ItemHasEntriesChange(table, field string) *AppError {
	return &AppError{Code: ErrItemHasEntriesChange, Table: table, Field: field}
}
