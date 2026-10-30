package errors

// Purchase documents
const (
	ErrPurchResetVendorLines      ErrorCode = "ERR_PURCH_RESET_VENDOR_LINES"
	ErrPurchQtyLessThan           ErrorCode = "ERR_PURCH_QTY_LESS_THAN"
	ErrPurchCannotReceiveMore     ErrorCode = "ERR_PURCH_CANNOT_RECEIVE_MORE"
	ErrPurchCannotReceiveMoreBase ErrorCode = "ERR_PURCH_CANNOT_RECEIVE_MORE_BASE"
	ErrPurchCannotInvoiceMore     ErrorCode = "ERR_PURCH_CANNOT_INVOICE_MORE"
	ErrPurchCannotInvoiceMoreBase ErrorCode = "ERR_PURCH_CANNOT_INVOICE_MORE_BASE"
	ErrPurchIndirectCostStandard  ErrorCode = "ERR_PURCH_INDIRECT_COST_STANDARD"
)

// PurchResetVendorLines: Buy-from Vendor No. cleared on a document that has lines ("You cannot
// reset Buy-from Vendor No. because the document still has one or more lines.")
func PurchResetVendorLines(table, field string) *AppError {
	return &AppError{Code: ErrPurchResetVendorLines, Table: table, Field: field}
}

// PurchQtyLessThan: a line's quantity is below what was received or invoiced ("Quantity must
// not be less than Quantity Received.")
func PurchQtyLessThan(table, field, field2 string) *AppError {
	return &AppError{Code: ErrPurchQtyLessThan, Table: table, Field: field, Field2: field2}
}

// PurchCannotReceiveMore: Qty. to Receive above the outstanding quantity ("You cannot receive
// more than %1 units."; base: in base units)
func PurchCannotReceiveMore(qty string, base bool) *AppError {
	if base {
		return &AppError{Code: ErrPurchCannotReceiveMoreBase, Params: []string{qty}}
	}
	return &AppError{Code: ErrPurchCannotReceiveMore, Params: []string{qty}}
}

// PurchCannotInvoiceMore: Qty. to Invoice above what can be invoiced ("You cannot invoice more
// than %1 units."; base: in base units)
func PurchCannotInvoiceMore(qty string, base bool) *AppError {
	if base {
		return &AppError{Code: ErrPurchCannotInvoiceMoreBase, Params: []string{qty}}
	}
	return &AppError{Code: ErrPurchCannotInvoiceMore, Params: []string{qty}}
}

// PurchIndirectCostStandard: Indirect Cost % changed on an item with Costing Method Standard
// ("You cannot change Indirect Cost % when Costing Method is Standard."); the line's field is
// %1, the item's Costing Method caption and value (in the user's language) %2 and %3
func PurchIndirectCostStandard(table, field, costingMethodCaption, costingMethod string) *AppError {
	return &AppError{Code: ErrPurchIndirectCostStandard, Table: table, Field: field, Params: []string{costingMethodCaption, costingMethod}}
}
