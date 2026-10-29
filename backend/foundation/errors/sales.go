package errors

// Sales documents
const (
	ErrSalesQtyLessThan         ErrorCode = "ERR_SALES_QTY_LESS_THAN"
	ErrSalesCannotShipMore      ErrorCode = "ERR_SALES_CANNOT_SHIP_MORE"
	ErrSalesCannotShipMoreBase  ErrorCode = "ERR_SALES_CANNOT_SHIP_MORE_BASE"
	ErrSalesCannotInvoiceMore   ErrorCode = "ERR_SALES_CANNOT_INVOICE_MORE"
	ErrSalesCannotInvoiceMoreBs ErrorCode = "ERR_SALES_CANNOT_INVOICE_MORE_BASE"
)

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
