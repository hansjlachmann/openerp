package tables

import (
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// noSeriesLogic is the business logic of table 308 No_Series.
// Add a trigger when it is needed: the available triggers are listed at the
// top of generated/tables/noseries_gen.go. The field rules of the definition
// (required, length, range) are checked after OnInsert and OnModify.
type noSeriesLogic struct{}

func init() { gtables.RegisterNoSeriesLogic(noSeriesLogic{}) }

// OnInsert runs before a new record is inserted
func (noSeriesLogic) OnInsert(rec *gtables.NoSeries) error {
	return nil
}

// OnModify runs before a record is modified
func (noSeriesLogic) OnModify(rec *gtables.NoSeries) error {
	return nil
}

// OnDelete deletes the series' lines and its relationships (both as the series and as a
// related series)
func (noSeriesLogic) OnDelete(rec *gtables.NoSeries) error {
	var line gtables.NoSeriesLine
	line.InitWithDBType(rec.GetDB(), rec.GetCompany(), rec.GetDBType())
	line.SetRange("series_code", rec.Code)
	line.DeleteAll()
	var rel gtables.NoSeriesRelationship
	rel.InitWithDBType(rec.GetDB(), rec.GetCompany(), rec.GetDBType())
	rel.SetRange("code", rec.Code)
	rel.DeleteAll()
	rel.SetRange("code")
	rel.SetRange("series_code", rec.Code)
	rel.DeleteAll()
	return nil
}

// OnRename runs before a primary key field is changed (rec.OldValue has the old value)
func (noSeriesLogic) OnRename(rec *gtables.NoSeries) error {
	return nil
}
