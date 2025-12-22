package types

import "database/sql"

// Trigger interfaces for table lifecycle events
// Similar to Business Central's OnInsert, OnModify, OnDelete, OnRename triggers

// BeforeInsert is called before a record is inserted
type BeforeInsert interface {
	OnInsert() error
}

// BeforeModify is called before a record is modified
type BeforeModify interface {
	OnModify() error
}

// BeforeDelete is called before a record is deleted
// Receives database connection and company name for validation
type BeforeDelete interface {
	OnDelete(db *sql.DB, company string) error
}

// BeforeRename is called before a record's primary key is changed
type BeforeRename interface {
	OnRename() error
}

// AfterInsert is called after a record is inserted
type AfterInsert interface {
	AfterInsert(db *sql.DB) error
}

// AfterModify is called after a record is modified
type AfterModify interface {
	AfterModify(db *sql.DB) error
}

// AfterDelete is called after a record is deleted
type AfterDelete interface {
	AfterDelete(db *sql.DB) error
}

// Validator interface for record validation
type Validator interface {
	Validate() error
}
