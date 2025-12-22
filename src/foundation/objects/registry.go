package objects

import (
	"fmt"

	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// ObjectRegistry maintains a registry of all Business Central objects
// Similar to BC's object numbering system
type ObjectRegistry struct {
	tables    map[int]interface{}
	pages     map[int]interface{}
	reports   map[int]interface{}
	codeunits map[int]interface{}
}

// NewObjectRegistry creates a new object registry
func NewObjectRegistry() *ObjectRegistry {
	return &ObjectRegistry{
		tables:    make(map[int]interface{}),
		pages:     make(map[int]interface{}),
		reports:   make(map[int]interface{}),
		codeunits: make(map[int]interface{}),
	}
}

// RegisterTable registers a table with its object ID
func (or *ObjectRegistry) RegisterTable(id int, table interface{}) error {
	if err := validateObjectID(id); err != nil {
		return fmt.Errorf("invalid table ID %d: %w", id, err)
	}

	if _, exists := or.tables[id]; exists {
		return fmt.Errorf("table ID %d already registered", id)
	}

	or.tables[id] = table
	return nil
}

// RegisterPage registers a page with its object ID
func (or *ObjectRegistry) RegisterPage(id int, page interface{}) error {
	if err := validateObjectID(id); err != nil {
		return fmt.Errorf("invalid page ID %d: %w", id, err)
	}

	if _, exists := or.pages[id]; exists {
		return fmt.Errorf("page ID %d already registered", id)
	}

	or.pages[id] = page
	return nil
}

// RegisterCodeunit registers a codeunit with its object ID
func (or *ObjectRegistry) RegisterCodeunit(id int, codeunit interface{}) error {
	if err := validateObjectID(id); err != nil {
		return fmt.Errorf("invalid codeunit ID %d: %w", id, err)
	}

	if _, exists := or.codeunits[id]; exists {
		return fmt.Errorf("codeunit ID %d already registered", id)
	}

	or.codeunits[id] = codeunit
	return nil
}

// RegisterReport registers a report with its object ID
func (or *ObjectRegistry) RegisterReport(id int, report interface{}) error {
	if err := validateObjectID(id); err != nil {
		return fmt.Errorf("invalid report ID %d: %w", id, err)
	}

	if _, exists := or.reports[id]; exists {
		return fmt.Errorf("report ID %d already registered", id)
	}

	or.reports[id] = report
	return nil
}

// GetTable retrieves a table by ID
func (or *ObjectRegistry) GetTable(id int) (interface{}, bool) {
	table, ok := or.tables[id]
	return table, ok
}

// GetPage retrieves a page by ID
func (or *ObjectRegistry) GetPage(id int) (interface{}, bool) {
	page, ok := or.pages[id]
	return page, ok
}

// GetCodeunit retrieves a codeunit by ID
func (or *ObjectRegistry) GetCodeunit(id int) (interface{}, bool) {
	codeunit, ok := or.codeunits[id]
	return codeunit, ok
}

// GetReport retrieves a report by ID
func (or *ObjectRegistry) GetReport(id int) (interface{}, bool) {
	report, ok := or.reports[id]
	return report, ok
}

// ListTables returns all registered table IDs
func (or *ObjectRegistry) ListTables() []int {
	ids := make([]int, 0, len(or.tables))
	for id := range or.tables {
		ids = append(ids, id)
	}
	return ids
}

// ListPages returns all registered page IDs
func (or *ObjectRegistry) ListPages() []int {
	ids := make([]int, 0, len(or.pages))
	for id := range or.pages {
		ids = append(ids, id)
	}
	return ids
}

// ListCodeunits returns all registered codeunit IDs
func (or *ObjectRegistry) ListCodeunits() []int {
	ids := make([]int, 0, len(or.codeunits))
	for id := range or.codeunits {
		ids = append(ids, id)
	}
	return ids
}

// ListReports returns all registered report IDs
func (or *ObjectRegistry) ListReports() []int {
	ids := make([]int, 0, len(or.reports))
	for id := range or.reports {
		ids = append(ids, id)
	}
	return ids
}

// validateObjectID validates that an object ID is within valid BC ranges
func validateObjectID(id int) error {
	// Check if within any valid range
	if (id >= types.RangeMicrosoftStart && id <= types.RangeMicrosoftEnd) ||
		(id >= types.RangeCustomerStart && id <= types.RangeCustomerEnd) ||
		(id >= types.RangeAddonStart && id <= types.RangeAddonEnd) ||
		(id >= types.RangeAddon2Start && id <= types.RangeAddon2End) {
		return nil
	}

	return fmt.Errorf("object ID %d is outside valid ranges", id)
}

// GetObjectRange returns a description of which range an object ID belongs to
func GetObjectRange(id int) string {
	if id >= types.RangeMicrosoftStart && id <= types.RangeCustomerStart-1 {
		return "Microsoft Base Application"
	}
	if id >= types.RangeCustomerStart && id <= types.RangeCustomerEnd {
		return "Customer Customization"
	}
	if id >= types.RangeAddonStart && id <= types.RangeAddonEnd {
		return "Add-on Range 1"
	}
	if id >= types.RangeAddon2Start && id <= types.RangeAddon2End {
		return "Add-on Range 2"
	}
	return "Invalid Range"
}
