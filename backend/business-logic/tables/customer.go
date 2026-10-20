package tables

import (
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/system"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// customerLogic is the business logic of table 18 Customer
type customerLogic struct{}

func init() { gtables.RegisterCustomerLogic(customerLogic{}) }

// OnInsert: a new customer is open
func (customerLogic) OnInsert(rec *gtables.Customer) error {
	rec.Status = gtables.Customer_Status.Open
	return nil
}

// OnValidate_Payment_terms_code: the payment terms must exist and be active
func (customerLogic) OnValidate_Payment_terms_code(rec *gtables.Customer) error {
	if rec.Payment_terms_code.IsEmpty() {
		return nil
	}
	var terms gtables.PaymentTerms
	terms.InitWithDBType(rec.GetDB(), rec.GetCompany(), rec.GetDBType())
	if !terms.Get(rec.Payment_terms_code) {
		return apperrors.RelatedNotFound(gtables.CustomerTableName, "payment_terms_code", rec.Payment_terms_code.String())
	}
	if !terms.Active {
		return apperrors.RelatedInactive(gtables.CustomerTableName, "payment_terms_code", rec.Payment_terms_code.String())
	}
	return nil
}

// OnValidate_Name: a name has at least 3 characters
func (customerLogic) OnValidate_Name(rec *gtables.Customer) error {
	if n := system.StrLen(string(rec.Name)); n > 0 && n < 3 {
		return apperrors.FieldTooShort(gtables.CustomerTableName, "name", 3)
	}
	return nil
}
