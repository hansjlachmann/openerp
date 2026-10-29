package errors

// Sales documents
const (
	ErrSalesQtyLessThan         ErrorCode = "ERR_SALES_QTY_LESS_THAN"
	ErrSalesCannotShipMore      ErrorCode = "ERR_SALES_CANNOT_SHIP_MORE"
	ErrSalesCannotShipMoreBase  ErrorCode = "ERR_SALES_CANNOT_SHIP_MORE_BASE"
	ErrSalesCannotInvoiceMore   ErrorCode = "ERR_SALES_CANNOT_INVOICE_MORE"
	ErrSalesCannotInvoiceMoreBs ErrorCode = "ERR_SALES_CANNOT_INVOICE_MORE_BASE"
	ErrSalesShipOrInvoice       ErrorCode = "ERR_SALES_SHIP_OR_INVOICE"
	ErrSalesNothingToPost       ErrorCode = "ERR_SALES_NOTHING_TO_POST"
	ErrSalesCannotPostType      ErrorCode = "ERR_SALES_CANNOT_POST_TYPE"
)

// SalesShipOrInvoice: an order is posted with neither Ship nor Invoice ("Please enter Yes in
// Ship and/or Invoice.")
func SalesShipOrInvoice(table, field, field2 string) *AppError {
	return &AppError{Code: ErrSalesShipOrInvoice, Table: table, Field: field, Field2: field2}
}

// SalesNothingToPost: a document has no line with something to ship or invoice
func SalesNothingToPost() *AppError {
	return &AppError{Code: ErrSalesNothingToPost}
}

// SalesCannotPostType: a document of a type that is not posted (a quote, a blanket order) —
// "Document Type Quote cannot be posted."
func SalesCannotPostType(table, field string, value any) *AppError {
	return &AppError{Code: ErrSalesCannotPostType, Table: table, Field: field, Value: value}
}

// SalesQtyLessThan: a line's quantity is below what was shipped or invoiced ("Quantity must not
// be less than Quantity Shipped.")
func SalesQtyLessThan(table, field, field2 string) *AppError {
	return &AppError{Code: ErrSalesQtyLessThan, Table: table, Field: field, Field2: field2}
}

// SalesCannotShipMore: Qty. to Ship above the outstanding quantity ("You cannot ship more than
// %1 units."; base: in base units)
func SalesCannotShipMore(qty string, base bool) *AppError {
	if base {
		return &AppError{Code: ErrSalesCannotShipMoreBase, Params: []string{qty}}
	}
	return &AppError{Code: ErrSalesCannotShipMore, Params: []string{qty}}
}

// SalesCannotInvoiceMore: Qty. to Invoice above what can be invoiced ("You cannot invoice more
// than %1 units."; base: in base units)
func SalesCannotInvoiceMore(qty string, base bool) *AppError {
	if base {
		return &AppError{Code: ErrSalesCannotInvoiceMoreBs, Params: []string{qty}}
	}
	return &AppError{Code: ErrSalesCannotInvoiceMore, Params: []string{qty}}
}
