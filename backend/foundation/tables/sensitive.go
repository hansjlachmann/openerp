package tables

import "fmt"

// Sensitive fields (YAML `sensitive: true`, e.g. User.password_hash) never leave the server
// through the generic table API and cannot be used to query it: a filter, sort or search on
// one would reveal its value piece by piece without ever returning it.
//
// Masked fields (YAML `masked: true`, BC ExtendedDatatype Masked, e.g. SMTP_Setup.password)
// are write-only secrets: the API sends MaskedValue instead of a value that is set ("" when
// empty) and accepts writes — MaskedValue sent back means "unchanged". They cannot be
// queried either.

// MaskedValue stands for the value of a masked field that has one (eight bullets).
const MaskedValue = "••••••••"

// IsSensitive reports whether field is a sensitive field of table.
func IsSensitive(table Table, field string) bool {
	for _, f := range table.GetFields() {
		if f.Name == field {
			return f.Sensitive
		}
	}
	return false
}

// IsMasked reports whether field is a masked field of table.
func IsMasked(table Table, field string) bool {
	for _, f := range table.GetFields() {
		if f.Name == field {
			return f.Masked
		}
	}
	return false
}

// IsQueryableColumn reports whether field may be named in a filter, sort or search: a real
// column of the table (field names end up in SQL text) that is neither sensitive nor masked.
func IsQueryableColumn(table Table, field string) bool {
	return table.HasColumn(field) && !IsSensitive(table, field) && !IsMasked(table, field)
}

// DropMaskedPlaceholders removes masked fields sent back as MaskedValue from data (a write):
// the stored secret stays as it is.
func DropMaskedPlaceholders(table Table, data map[string]interface{}) {
	for _, f := range table.GetFields() {
		if f.Masked && data[f.Name] == MaskedValue {
			delete(data, f.Name)
		}
	}
}

// PublicMap is table.ToMap() without the sensitive fields and with masked fields as
// MaskedValue: use it for every record sent to a client.
func PublicMap(table Table) map[string]interface{} {
	m := table.ToMap()
	for _, f := range table.GetFields() {
		switch {
		case f.Sensitive:
			delete(m, f.Name)
		case f.Masked:
			if v, ok := m[f.Name]; ok && fmt.Sprint(v) != "" {
				m[f.Name] = MaskedValue
			} else {
				m[f.Name] = ""
			}
		}
	}
	return m
}
