package demodata

import (
	"time"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/sift"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// insertResCapacity gives each demo resource its capacity per working day (Monday to Friday)
// from a year ago to three months ahead: people 7.5 hours, machines 8 hours, a day-rate
// resource 1 day
func insertResCapacity(exec database.Executor, company string, dbType database.DBType, masterData []tableRecords, today time.Time, counts *Counts) error {
	var entries []*gtables.ResCapacityEntry
	entryNo := 0
	start := today.AddDate(-1, 0, 1)
	end := today.AddDate(0, 3, 0)
	for _, t := range masterData {
		if t.Table != gtables.ResourceTableName {
			continue
		}
		for _, r := range t.Records {
			capacity := types.MustDecimal("7.5")
			if r["type"] == 1 {
				capacity = types.NewDecimalFromInt(8)
			}
			if r["unit_of_measure_code"] == "DAY" {
				capacity = types.NewDecimalFromInt(1)
			}
			for date := start; !date.After(end); date = date.AddDate(0, 0, 1) {
				if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
					continue
				}
				entryNo++
				e := &gtables.ResCapacityEntry{}
				e.InitWithDBType(exec, company, dbType)
				e.Entry_no = entryNo
				e.Resource_no = types.NewCode(r["no"].(string))
				e.Resource_group_no = types.NewCode(r["resource_group_no"].(string))
				e.Date = types.NewDateFromTime(date)
				e.Capacity = capacity
				entries = append(entries, e)
			}
		}
	}
	if len(entries) == 0 {
		return nil
	}
	// The SIFT totals are built once after the load instead of updated per entry
	err := sift.Suspend(exec, dbType, company, gtables.ResCapacityEntryTableName)
	if err != nil {
		return err
	}
	err = entries[0].InsertAll(entries, false)
	if err != nil {
		return err
	}
	counts.ResCapacityEntries += len(entries)
	return (&gtables.ResCapacityEntry{}).SyncKeys(exec, company, dbType)
}
