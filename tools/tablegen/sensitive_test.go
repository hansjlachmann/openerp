package main

import (
	"strings"
	"testing"
)

func TestValidateSensitiveFields(t *testing.T) {
	user := &TableDef{}
	user.Table.Name = "User"
	user.Table.Fields = []Field{
		{Name: "user_id", PrimaryKey: true},
		{Name: "user_name"},
		{Name: "password_hash", Sensitive: true},
	}
	member := func(rel TableRelation) *TableDef {
		d := &TableDef{}
		d.Table.Name = "User Member"
		d.Table.Fields = []Field{{Name: "user_id", PrimaryKey: true, TableRelation: &rel}}
		return d
	}
	byStruct := map[string]*TableDef{"User": user}

	if err := validateSensitiveFields(user, byStruct); err != nil {
		t.Errorf("User: %v", err)
	}
	ok := member(TableRelation{Table: "User", Field: "user_id", LookupColumns: []LookupColumn{{Source: "user_id"}, {Source: "user_name"}}})
	if err := validateSensitiveFields(ok, byStruct); err != nil {
		t.Errorf("relation without sensitive columns: %v", err)
	}
	for _, rel := range []TableRelation{
		{Table: "User", Field: "user_id", LookupColumns: []LookupColumn{{Source: "password_hash"}}},
		{Table: "User", Field: "user_id", DisplayField: "password_hash"},
	} {
		if err := validateSensitiveFields(member(rel), byStruct); err == nil || !strings.Contains(err.Error(), "password_hash") {
			t.Errorf("relation showing password_hash: err = %v", err)
		}
	}

	key := &TableDef{}
	key.Table.Fields = []Field{{Name: "secret", PrimaryKey: true, Sensitive: true}}
	if err := validateSensitiveFields(key, byStruct); err == nil {
		t.Error("sensitive primary key accepted")
	}
}
