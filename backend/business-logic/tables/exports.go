package tables

// This file re-exports types, constants, and namespaces from the generated package
// for backwards compatibility with existing code that imports tables package.

import (
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// Table ID constants
const (
	CustomerTableID            = gtables.CustomerTableID
	PaymentTermsTableID        = gtables.PaymentTermsTableID
	CustomerLedgerEntryTableID = gtables.CustomerLedgerEntryTableID
	UserTableID                = gtables.UserTableID
	UserPreferencesTableID     = gtables.UserPreferencesTableID
	SMTPSetupTableID           = gtables.SMTPSetupTableID
)

// Table name constants
const (
	CustomerTableName            = gtables.CustomerTableName
	PaymentTermsTableName        = gtables.PaymentTermsTableName
	CustomerLedgerEntryTableName = gtables.CustomerLedgerEntryTableName
	UserTableName                = gtables.UserTableName
	UserPreferencesTableName     = gtables.UserPreferencesTableName
	SMTPSetupTableName           = gtables.SMTPSetupTableName
)

// Option type aliases
type (
	CustomerStatus                         = gtables.CustomerStatus
	CustomerLedgerEntryDocument_type       = gtables.CustomerLedgerEntryDocument_type
	CustomerLedgerEntryBal_account_type    = gtables.CustomerLedgerEntryBal_account_type
	CustomerLedgerEntryApplies_to_doc_type = gtables.CustomerLedgerEntryApplies_to_doc_type
)

// Option namespace re-exports
var (
	Customer_Status                         = gtables.Customer_Status
	CustomerLedgerEntry_Document_type       = gtables.CustomerLedgerEntry_Document_type
	CustomerLedgerEntry_Bal_account_type    = gtables.CustomerLedgerEntry_Bal_account_type
	CustomerLedgerEntry_Applies_to_doc_type = gtables.CustomerLedgerEntry_Applies_to_doc_type
)
