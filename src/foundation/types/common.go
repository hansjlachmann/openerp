package types

// FieldInfo represents field metadata for Object Designer
type FieldInfo struct {
	Name         string
	Type         string
	IsPrimaryKey bool
	FieldOrder   int
}

// ObjectType represents the type of Business Central object
type ObjectType int

const (
	ObjectTypeTable ObjectType = iota
	ObjectTypePage
	ObjectTypeReport
	ObjectTypeCodeunit
	ObjectTypeQuery
	ObjectTypeXMLPort
)

// String returns the string representation of ObjectType
func (ot ObjectType) String() string {
	return [...]string{
		"Table",
		"Page",
		"Report",
		"Codeunit",
		"Query",
		"XMLPort",
	}[ot]
}

// Object numbering ranges (Business Central standard)
const (
	// Microsoft reserved: 1-99,999,999
	RangeMicrosoftStart = 1
	RangeMicrosoftEnd   = 99999999

	// Customer objects: 50,000-99,999 (customization range)
	RangeCustomerStart = 50000
	RangeCustomerEnd   = 99999

	// Add-on range 1: 1,000,000-69,999,999
	RangeAddonStart = 1000000
	RangeAddonEnd   = 69999999

	// Add-on range 2: 70,000,000-74,999,999
	RangeAddon2Start = 70000000
	RangeAddon2End   = 74999999
)
