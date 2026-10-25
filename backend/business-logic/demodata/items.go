package demodata

import (
	"fmt"
	"math/rand"
	"time"

	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// postDemoItems posts a year of inventory movements of the demo items through the item posting
// (codeunit 22, with Automatic Cost Posting also their value in the G/L): an opening stock a
// year ago and every quarter a purchase and two sales, at the main warehouse
func postDemoItems(src ftables.Source, open func(src ftables.Source) ItemJnlPoster, today time.Time, masterData []tableRecords, counts *Counts) error {
	rng := rand.New(rand.NewSource(seed + 27))
	itemJnlPostLine := open(src)
	start := today.AddDate(-1, 0, 1)
	lineNo := 0
	post := func(date time.Time, entryType gtables.ItemJournalLineEntry_type, itemNo, documentNo string, qty int, unitAmount *types.Decimal) error {
		lineNo = lineNo + 10000
		line := gtables.NewItemJournalLine(src)
		line.InitRecord()
		line.Journal_template_name = "ITEM"
		line.Journal_batch_name = "DEFAULT"
		line.Line_no = lineNo
		line.Source_code = "ITEMJNL"
		line.Document_no = types.Code(documentNo)
		for _, step := range []struct {
			field string
			value interface{}
		}{
			{"posting_date", types.NewDateFromTime(date)},
			{"entry_type", entryType},
			{"item_no", types.Code(itemNo)},
			{"location_code", types.Code("MAIN")},
			{"quantity", types.NewDecimalFromInt(int64(qty))},
		} {
			err := line.Validate(step.field, step.value)
			if err != nil {
				return fmt.Errorf("demo item journal %s %s: %w", itemNo, step.field, err)
			}
		}
		if unitAmount != nil {
			err := line.Validate("unit_amount", *unitAmount)
			if err != nil {
				return fmt.Errorf("demo item journal %s unit amount: %w", itemNo, err)
			}
		}
		err := itemJnlPostLine.Run(line)
		if err != nil {
			return fmt.Errorf("demo item journal %s %s: %w", itemNo, documentNo, err)
		}
		counts.ItemLedgerEntries++
		return nil
	}

	for _, t := range masterData {
		if t.Table != gtables.ItemTableName {
			continue
		}
		for _, r := range t.Records {
			itemNo := fmt.Sprint(r["no"])
			cost, err := types.NewDecimalFromString(fmt.Sprint(r["last_direct_cost"]))
			if err != nil {
				return err
			}
			err = post(start, gtables.ItemJournalLine_Entry_type.PositiveAdjmt, itemNo, "OPENING", 40+rng.Intn(160), nil)
			if err != nil {
				return err
			}
			for q := 0; q < 4; q++ {
				quarter := start.AddDate(0, 3*q, 0)
				// purchases at the last direct cost ±10 %
				price := cost.Mul(types.NewDecimal(0.9 + rng.Float64()*0.2)).Round(2)
				err = post(quarter.AddDate(0, 0, 5), gtables.ItemJournalLine_Entry_type.Purchase, itemNo, fmt.Sprintf("PURCH%d", q+1), 20+rng.Intn(100), &price)
				if err != nil {
					return err
				}
				for s := 0; s < 2; s++ {
					err = post(quarter.AddDate(0, 0, 30+30*s), gtables.ItemJournalLine_Entry_type.Sale, itemNo, fmt.Sprintf("SALE%d-%d", q+1, s+1), 5+rng.Intn(45), nil)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
