package errors

// Journal and posting errors (general journal lines, templates, batches, posting codeunits)
const (
	ErrJnlAccountTypes        ErrorCode = "ERR_JNL_ACCOUNT_TYPES"
	ErrJnlOptionNotAllowed    ErrorCode = "ERR_JNL_OPTION_NOT_ALLOWED"
	ErrJnlOptionInternal      ErrorCode = "ERR_JNL_OPTION_INTERNAL"
	ErrJnlBankAccountRequired ErrorCode = "ERR_JNL_BANK_ACCOUNT_REQUIRED"
	ErrJnlWithoutField        ErrorCode = "ERR_JNL_WITHOUT_FIELD"
	ErrJnlRecurringOnlyField  ErrorCode = "ERR_JNL_RECURRING_ONLY_FIELD"
	ErrJnlMustNotBe           ErrorCode = "ERR_JNL_MUST_NOT_BE"
	ErrJnlFixedAssetNotYet    ErrorCode = "ERR_JNL_FIXED_ASSET_NOT_YET"
	ErrJnlTaxNotYet           ErrorCode = "ERR_JNL_TAX_NOT_YET"
	ErrJnlTemplateTypeNoPage  ErrorCode = "ERR_JNL_TEMPLATE_TYPE_NO_PAGE"
)

// JnlAccountTypes: Account Type or Bal. Account Type must be G/L Account or Bank Account (a
// line cannot post customer/vendor to customer/vendor)
func JnlAccountTypes(table, field, field2 string) *AppError {
	return &AppError{Code: ErrJnlAccountTypes, Table: table, Field: field, Field2: field2}
}

// JnlOptionNotAllowed: a field cannot have an option value ("Bal. Account Type cannot be
// Employee.")
func JnlOptionNotAllowed(table, field string, value any) *AppError {
	return &AppError{Code: ErrJnlOptionNotAllowed, Table: table, Field: field, Value: value}
}

// JnlOptionInternal: an option value only the system sets (Gen. Posting Type Settlement)
func JnlOptionInternal(table, field string, value any) *AppError {
	return &AppError{Code: ErrJnlOptionInternal, Table: table, Field: field, Value: value}
}

// JnlBankAccountRequired: Account Type or Bal. Account Type must be a bank account
func JnlBankAccountRequired(table, field, field2 string) *AppError {
	return &AppError{Code: ErrJnlBankAccountRequired, Table: table, Field: field, Field2: field2}
}

// JnlWithoutField: a field cannot be filled in without another (Currency Factor without
// Currency Code)
func JnlWithoutField(table, field, field2 string) *AppError {
	return &AppError{Code: ErrJnlWithoutField, Table: table, Field: field, Field2: field2}
}

// JnlRecurringOnlyField: on recurring journals only one numbering field can be filled in
func JnlRecurringOnlyField(table, field string) *AppError {
	return &AppError{Code: ErrJnlRecurringOnlyField, Table: table, Field: field}
}

// JnlMustNotBe: a field must not have a value (Posting No. Series equal to No. Series)
func JnlMustNotBe(table, field, value string) *AppError {
	return &AppError{Code: ErrJnlMustNotBe, Table: table, Field: field, Params: []string{value}}
}

// JnlFixedAssetNotYet: fixed asset lines cannot be posted until fixed asset posting is built
func JnlFixedAssetNotYet(table, field string) *AppError {
	return &AppError{Code: ErrJnlFixedAssetNotYet, Table: table, Field: field}
}

// JnlTaxNotYet: a tax calculation type that cannot be posted (only Sales Tax)
func JnlTaxNotYet(table, field string, value any) *AppError {
	return &AppError{Code: ErrJnlTaxNotYet, Table: table, Field: field, Value: value}
}

// JnlTemplateTypeNoPage: a journal template type without a journal page yet (Assets, Payroll
// Accrual)
func JnlTemplateTypeNoPage(table, field string, value any) *AppError {
	return &AppError{Code: ErrJnlTemplateTypeNoPage, Table: table, Field: field, Value: value}
}
