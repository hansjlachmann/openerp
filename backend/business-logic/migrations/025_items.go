package migrations

import (
	fmigrations "github.com/hansjlachmann/openerp/backend/foundation/migrations"
)

func init() {
	Register(&Migration025Items{})
}

// Migration025Items goes with the items (tables 27 Item, 30 Item Translation, 94 Inventory
// Posting Group, 99 Item Vendor, 204 Unit of Measure, 313 Inventory Setup, 5401 Item Variant,
// 5402 Unit of Measure Translation, 5404 Item Unit of Measure). The READER role may read them,
// as all business tables; the tables come from the table sync.
type Migration025Items struct{}

func (m *Migration025Items) Version() int {
	return 25
}

func (m *Migration025Items) Name() string {
	return "items"
}

func (m *Migration025Items) Description() string {
	return "READER reads the items and their units, variants, translations, vendors and setup"
}

func (m *Migration025Items) Up(ctx *fmigrations.Context) error {
	return grantReaderRead(ctx, itemTables)
}

var itemTables = []string{
	"ITEM", "ITEM_TRANSLATION", "INVENTORY_POSTING_GROUP", "ITEM_VENDOR", "UNIT_OF_MEASURE", "INVENTORY_SETUP",
	"ITEM_VARIANT", "UNIT_OF_MEASURE_TRANSLATION", "ITEM_UNIT_OF_MEASURE",
}
