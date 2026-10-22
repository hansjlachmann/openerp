package tables

import (
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// reminderLevelLogic is the business logic of table 293 Reminder_Level.
// Add a trigger when it is needed: the available triggers are listed at the
// top of generated/tables/reminderlevel_gen.go. The field rules of the definition
// (required, length, range) are checked after OnInsert and OnModify.
type reminderLevelLogic struct{}

func init() { gtables.RegisterReminderLevelLogic(reminderLevelLogic{}) }

// OnInsert runs before a new record is inserted
func (reminderLevelLogic) OnInsert(rec *gtables.ReminderLevel) error {
	return nil
}

// OnModify runs before a record is modified
func (reminderLevelLogic) OnModify(rec *gtables.ReminderLevel) error {
	return nil
}

// OnDelete deletes the level's beginning and ending texts
func (reminderLevelLogic) OnDelete(rec *gtables.ReminderLevel) error {
	var text gtables.ReminderText
	text.InitWithDBType(rec.GetDB(), rec.GetCompany(), rec.GetDBType())
	text.SetRange("reminder_terms_code", rec.Reminder_terms_code)
	text.SetRange("reminder_level", rec.No)
	text.DeleteAll()
	return nil
}

// OnRename runs before a primary key field is changed (rec.OldValue has the old value)
func (reminderLevelLogic) OnRename(rec *gtables.ReminderLevel) error {
	return nil
}

// NewReminderLevel gives a new level the next No. of its reminder terms (NewRecord)
func NewReminderLevel(rec *gtables.ReminderLevel) {
	var level gtables.ReminderLevel
	level.InitWithDBType(rec.GetDB(), rec.GetCompany(), rec.GetDBType())
	level.SetRange("reminder_terms_code", rec.Reminder_terms_code)
	level.FindLast()
	rec.No = level.No + 1
}
