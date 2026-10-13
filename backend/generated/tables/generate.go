// Package tables is the generated code of the tables: one record type per table
// definition (backend/definitions/tables) and the table registry. Never edit it: change the
// definition and run go generate here. The business logic is in business-logic/tables.
package tables

//go:generate go run ../../../tools/tablegen/main.go -defs ../../definitions/tables -out . -logic ../../business-logic/tables -app ../../definitions/application.yaml
