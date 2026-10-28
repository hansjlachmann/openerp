package reports

import "fmt"

// IntegerTable: a data item of this table loops over numbers (the classic Integer table) —
// copies, pages, the lines of an array. Its record is *Integer; OnPreDataItem sets the range,
// rec.SetRange("number", 1, noOfCopies); without a range the data item has no records.
const IntegerTable = "Integer"

// Integer is the record of an Integer data item: Number is the current number
type Integer struct {
	Number int
	from   int
	to     int
	ranged bool
}

// SetRange sets the numbers the data item loops over (one value: only that number; no
// values: none)
func (i *Integer) SetRange(field string, values ...interface{}) {
	if field != "number" {
		panic(fmt.Sprintf("Integer has no field %s", field))
	}
	switch len(values) {
	case 0:
		i.ranged = false
	case 1:
		i.from, i.to, i.ranged = toInt(values[0]), toInt(values[0]), true
	default:
		i.from, i.to, i.ranged = toInt(values[0]), toInt(values[1]), true
	}
}

// ToMap: the record's fields (a layout prints field number)
func (i *Integer) ToMap() map[string]interface{} {
	return map[string]interface{}{"number": i.Number}
}

// toInt is a whole number of any integer type
func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	}
	panic(fmt.Sprintf("Integer.SetRange: %v is not a whole number", v))
}
