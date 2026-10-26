package migrations

import (
	fmigrations "github.com/hansjlachmann/openerp/backend/foundation/migrations"
)

func init() {
	Register(&Migration026ResourcesBOM{})
}

// Migration026ResourcesBOM goes with the resources (tables 152 Resource Group, 156 Resource,
// 160 Res. Capacity Entry, 200 Work Type, 201 Resource Price, 202 Resource Cost, 314 Resources
// Setup, 335 Resource Price Change) and Rounding Method 42. The READER role may read them, as
// all business tables; the tables come from the table sync.
type Migration026ResourcesBOM struct{}

func (m *Migration026ResourcesBOM) Version() int {
	return 26
}

func (m *Migration026ResourcesBOM) Name() string {
	return "resources-bom"
}

func (m *Migration026ResourcesBOM) Description() string {
	return "READER reads the resources, their prices and costs, and the rounding methods"
}

func (m *Migration026ResourcesBOM) Up(ctx *fmigrations.Context) error {
	return grantReaderRead(ctx, resourceBOMTables)
}

var resourceBOMTables = []string{
	"RESOURCE_GROUP", "RESOURCE", "RES_CAPACITY_ENTRY", "WORK_TYPE", "RESOURCE_PRICE", "RESOURCE_COST",
	"RESOURCES_SETUP", "RESOURCE_PRICE_CHANGE", "ROUNDING_METHOD",
}
