package tables

import (
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// customerPostingGroupLogic is the business logic of table 92 Customer_Posting_Group.
// Add a trigger when it is needed: the available triggers are listed at the
// top of generated/tables/customerpostinggroup_gen.go. The field rules of the definition
// (required, length, range) are checked after OnInsert and OnModify.
type customerPostingGroupLogic struct{}

func init() { gtables.RegisterCustomerPostingGroupLogic(customerPostingGroupLogic{}) }

// OnInsert runs before a new record is inserted
func (customerPostingGroupLogic) OnInsert(rec *gtables.CustomerPostingGroup) error {
	return nil
}

// OnModify runs before a record is modified
func (customerPostingGroupLogic) OnModify(rec *gtables.CustomerPostingGroup) error {
	return nil
}

// OnDelete runs before a record is deleted
func (customerPostingGroupLogic) OnDelete(rec *gtables.CustomerPostingGroup) error {
	return nil
}

// OnRename runs before a primary key field is changed (rec.OldValue has the old value)
func (customerPostingGroupLogic) OnRename(rec *gtables.CustomerPostingGroup) error {
	return nil
}

// OnValidate_Receivables_account: a posting account
func (customerPostingGroupLogic) OnValidate_Receivables_account(rec *gtables.CustomerPostingGroup) error {
	return checkCustPostingGroupGLAcc(rec, "receivables_account", rec.Receivables_account, false, false)
}

// OnValidate_Service_charge_acc: a posting account with a Gen. Prod. Posting Group and Direct Posting
func (customerPostingGroupLogic) OnValidate_Service_charge_acc(rec *gtables.CustomerPostingGroup) error {
	return checkCustPostingGroupGLAcc(rec, "service_charge_acc", rec.Service_charge_acc, true, true)
}

// OnValidate_Payment_disc_acc: a posting account
func (customerPostingGroupLogic) OnValidate_Payment_disc_acc(rec *gtables.CustomerPostingGroup) error {
	return checkCustPostingGroupGLAcc(rec, "payment_disc_acc", rec.Payment_disc_acc, false, false)
}

// OnValidate_Invoice_rounding_account: a posting account with a Gen. Prod. Posting Group
func (customerPostingGroupLogic) OnValidate_Invoice_rounding_account(rec *gtables.CustomerPostingGroup) error {
	return checkCustPostingGroupGLAcc(rec, "invoice_rounding_account", rec.Invoice_rounding_account, true, false)
}

// OnValidate_Additional_fee_account: a posting account with a Gen. Prod. Posting Group and Direct Posting
func (customerPostingGroupLogic) OnValidate_Additional_fee_account(rec *gtables.CustomerPostingGroup) error {
	return checkCustPostingGroupGLAcc(rec, "additional_fee_account", rec.Additional_fee_account, true, true)
}

// OnValidate_Interest_account: a posting account with a Gen. Prod. Posting Group
func (customerPostingGroupLogic) OnValidate_Interest_account(rec *gtables.CustomerPostingGroup) error {
	return checkCustPostingGroupGLAcc(rec, "interest_account", rec.Interest_account, true, false)
}

// checkCustPostingGroupGLAcc checks the G/L account entered in a field: it exists and is a
// posting account that is not blocked, with a Gen. Prod. Posting Group and Direct Posting when
// asked
func checkCustPostingGroupGLAcc(rec *gtables.CustomerPostingGroup, field string, accNo types.Code, checkProdPostingGroup, checkDirectPosting bool) error {
	if accNo == "" {
		return nil
	}
	var glAcc gtables.GLAccount
	glAcc.InitWithDBType(rec.GetDB(), rec.GetCompany(), rec.GetDBType())
	if !glAcc.Get(accNo) {
		return apperrors.RelatedNotFound(gtables.CustomerPostingGroupTableName, field, accNo.String())
	}
	if err := CheckGLAcc(&glAcc); err != nil {
		return err
	}
	if checkProdPostingGroup {
		if err := glAcc.TestField("gen_prod_posting_group"); err != nil {
			return err
		}
	}
	if checkDirectPosting {
		return glAcc.TestField("direct_posting", true)
	}
	return nil
}
