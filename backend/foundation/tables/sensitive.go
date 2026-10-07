package tables

// Sensitive fields (YAML `sensitive: true`, e.g. User.password_hash) never leave the server
// through the generic table API and cannot be used to query it: a filter, sort or search on
// one would reveal its value piece by piece without ever returning it.

// IsSensitive reports whether field is a sensitive field of table.
func IsSensitive(table Table, field string) bool {
	for _, f := range table.GetFields() {
		if f.Name == field {
			return f.Sensitive
		}
	}
	return false
}

// IsQueryableColumn reports whether field may be named in a filter, sort or search: a real
// column of the table (field names end up in SQL text) that is not sensitive.
func IsQueryableColumn(table Table, field string) bool {
	return table.HasColumn(field) && !IsSensitive(table, field)
}

// PublicMap is table.ToMap() without the sensitive fields: use it for every record sent
// to a client.
func PublicMap(table Table) map[string]interface{} {
	m := table.ToMap()
	for _, f := range table.GetFields() {
		if f.Sensitive {
			delete(m, f.Name)
		}
	}
	return m
}
