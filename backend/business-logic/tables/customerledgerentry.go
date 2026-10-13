package tables

import (
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// customerLedgerEntryLogic is the business logic of table 21 Customer Ledger Entry
type customerLedgerEntryLogic struct{}

func init() { gtables.RegisterCustomerLedgerEntryLogic(customerLedgerEntryLogic{}) }

// OnValidate_Customer_no: the customer must exist
func (customerLedgerEntryLogic) OnValidate_Customer_no(rec *gtables.CustomerLedgerEntry) error {
	return checkCustomerExists(rec, "customer_no", rec.Customer_no)
}

// OnValidate_Sell_to_customer_no: the sell-to customer must exist
func (customerLedgerEntryLogic) OnValidate_Sell_to_customer_no(rec *gtables.CustomerLedgerEntry) error {
	return checkCustomerExists(rec, "sell_to_customer_no", rec.Sell_to_customer_no)
}

func checkCustomerExists(rec *gtables.CustomerLedgerEntry, field string, no types.Code) error {
	if no.IsEmpty() {
		return nil
	}
	var customer gtables.Customer
	customer.InitWithDBType(rec.GetDB(), rec.GetCompany(), rec.GetDBType())
	if !customer.Get(no) {
		return apperrors.RelatedNotFound(gtables.CustomerLedgerEntryTableName, field, no.String())
	}
	return nil
}
