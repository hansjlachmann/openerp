package tables

import (
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// itemDiscountGroupLogic is the business logic of table 341 Item Discount Group (none: the table's relations, ranges and
// required fields are its rules)
type itemDiscountGroupLogic struct{}

func init() { gtables.RegisterItemDiscountGroupLogic(itemDiscountGroupLogic{}) }
