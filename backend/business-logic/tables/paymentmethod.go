package tables

import (
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// paymentMethodLogic is the business logic of table 289 Payment_Method.
// Add a trigger when it is needed: the available triggers are listed at the
// top of generated/tables/paymentmethod_gen.go. The field rules of the definition
// (required, length, range) are checked after OnInsert and OnModify.
type paymentMethodLogic struct{}

func init() { gtables.RegisterPaymentMethodLogic(paymentMethodLogic{}) }

// OnInsert runs before a new record is inserted
func (paymentMethodLogic) OnInsert(rec *gtables.PaymentMethod) error {
	return nil
}

// OnModify runs before a record is modified
func (paymentMethodLogic) OnModify(rec *gtables.PaymentMethod) error {
	return nil
}

// OnDelete runs before a record is deleted
func (paymentMethodLogic) OnDelete(rec *gtables.PaymentMethod) error {
	return nil
}

// OnRename runs before a primary key field is changed (rec.OldValue has the old value)
func (paymentMethodLogic) OnRename(rec *gtables.PaymentMethod) error {
	return nil
}

// OnValidate_Bal_account_type clears Bal. Account No.
func (paymentMethodLogic) OnValidate_Bal_account_type(rec *gtables.PaymentMethod) error {
	rec.Bal_account_no = ""
	return nil
}

// OnValidate_Bal_account_no: a G/L account must be a posting account, not blocked, that allows
// direct posting
func (paymentMethodLogic) OnValidate_Bal_account_no(rec *gtables.PaymentMethod) error {
	if rec.Bal_account_type != gtables.PaymentMethod_Bal_account_type.GLAccount || rec.Bal_account_no == "" {
		return nil
	}
	var glAcc gtables.GLAccount
	glAcc.InitWithDBType(rec.GetDB(), rec.GetCompany(), rec.GetDBType())
	if !glAcc.Get(rec.Bal_account_no) {
		return apperrors.RelatedNotFound(gtables.PaymentMethodTableName, "bal_account_no", rec.Bal_account_no.String())
	}
	if err := CheckGLAcc(&glAcc); err != nil {
		return err
	}
	return glAcc.TestField("direct_posting", true)
}
