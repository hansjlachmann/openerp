// Package demodata fills a company with demo data: countries,
// payment terms, customers and customer ledger entries. The data lives in the YAML
// files under data/; records are written through the table API (Insert with triggers),
// so they obey the same rules as data entered by a user.
//
// Create runs in one transaction (all or nothing) and refuses a company that already
// has customers. Generated values come from a fixed random seed and dates are relative
// to the run date, so every run produces the same data.
package demodata

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/sift"
	"github.com/hansjlachmann/openerp/backend/foundation/system"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
	"gopkg.in/yaml.v3"

	// The table triggers (business logic) run for the records written here
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
)

//go:embed data/*.yaml
var dataFS embed.FS

// Size selects how much data is created.
type Size string

const (
	Small Size = "SMALL" // the customers in customers.yaml (screenshots, demos, E2E)
	Large Size = "LARGE" // ~10,000 customers (paging, search and FlowField load tests)
	Heavy Size = "HEAVY" // 200 customers with ~1,000 entries each over two years (SIFT volume)
)

// largeCustomerCount is the total number of customers in the LARGE set.
const largeCustomerCount = 10000

// heavyCustomerCount is the total number of customers in the HEAVY set; each gets about
// 1,000 ledger entries (≈500 invoices, most of them paid) spread over two years.
const heavyCustomerCount = 200

// seed makes every run generate the same data.
const seed = 20261005

// newRand returns the generator every run starts from.
func newRand() *rand.Rand { return rand.New(rand.NewSource(seed)) }

// ParseSize parses a Job Queue parameter: blank means SMALL.
func ParseSize(parameter string) (Size, error) {
	switch strings.ToUpper(strings.TrimSpace(parameter)) {
	case "", string(Small):
		return Small, nil
	case string(Large):
		return Large, nil
	case string(Heavy):
		return Heavy, nil
	}
	return "", &SizeError{Parameter: parameter}
}

// Phase identifies the step reported to Progress.
type Phase string

const (
	PhaseSetup     Phase = "setup"     // countries and payment terms
	PhaseCustomers Phase = "customers" // customers
	PhaseEntries   Phase = "entries"   // customer ledger entries
)

// Options control a Create run.
type Options struct {
	Size Size
	// Progress (optional) is called when the overall percentage changes.
	Progress func(percent int, phase Phase)
	// Today (optional) anchors the generated dates; zero means the current date.
	Today time.Time
}

// Counts reports what Create inserted.
type Counts struct {
	Countries           int
	PaymentTerms        int
	MasterData          int // records of the master data tables (setup.yaml master_data)
	CustomerData        int // the customers' comments, ship-to addresses, bank accounts (customer_data)
	Vendors             int
	VendorLedgerEntries int
	VendorData          int // the vendors' comments, order addresses, bank accounts (vendor_data)
	Customers           int
	LedgerEntries       int
	GLEntries           int // demo G/L entries (monthly sales and costs against the bank)
	// AccountingPeriods: monthly periods of the fiscal years from two years ago to this year
	// and the first period of the next year; the oldest year is closed
	AccountingPeriods int
}

// ErrCompanyNotEmpty is returned when the company already has customers.
var ErrCompanyNotEmpty = errors.New("company already has customers")

// SizeError is returned by ParseSize for an unknown size.
type SizeError struct{ Parameter string }

func (e *SizeError) Error() string { return fmt.Sprintf("unknown demo data size %q", e.Parameter) }

// InsertError is returned when a record cannot be inserted; nothing is kept.
type InsertError struct {
	Table string
	Key   string
	Err   error
}

func (e *InsertError) Error() string {
	return fmt.Sprintf("could not create %s %s: %v", e.Table, e.Key, e.Err)
}

func (e *InsertError) Unwrap() error { return e.Err }

// ---------------------------------------------------------------------------
// YAML data
// ---------------------------------------------------------------------------

type setupData struct {
	Countries []struct {
		Code          string `yaml:"code"`
		Name          string `yaml:"name"`
		EUCountryCode string `yaml:"eu_country_code"`
	} `yaml:"countries"`
	PaymentTerms []struct {
		Code        string `yaml:"code"`
		Description string `yaml:"description"`
		DueDays     int    `yaml:"due_days"`
	} `yaml:"payment_terms"`
	// MasterData: records of master data tables (API name), field name → value; the
	// tables' defaults fill the fields not given
	MasterData []tableRecords `yaml:"master_data"`
	// CustomerData: records that belong to the customers (comments, ship-to addresses, bank
	// accounts), inserted after them
	CustomerData []tableRecords `yaml:"customer_data"`
	// VendorData: the vendors' comments, order addresses and bank accounts, inserted after them
	VendorData []tableRecords `yaml:"vendor_data"`
}

// tableRecords are records of a table (API name), field name → value
type tableRecords struct {
	Table   string                   `yaml:"table"`
	Records []map[string]interface{} `yaml:"records"`
}

// recordCount is the number of records of the tables
func recordCount(data []tableRecords) int {
	n := 0
	for _, t := range data {
		n += len(t.Records)
	}
	return n
}

type customerData struct {
	No           string  `yaml:"no"`
	Name         string  `yaml:"name"`
	Address      string  `yaml:"address"`
	PostCode     string  `yaml:"post_code"`
	City         string  `yaml:"city"`
	Country      string  `yaml:"country"`
	Phone        string  `yaml:"phone"`
	PaymentTerms string  `yaml:"payment_terms"`
	CreditLimit  float64 `yaml:"credit_limit"`
	Blocked      bool    `yaml:"blocked"`
}

type customersFile struct {
	Customers []customerData `yaml:"customers"`
}

type generatorCountry struct {
	Country          string   `yaml:"country"`
	First            []string `yaml:"first"`
	Second           []string `yaml:"second"`
	LegalForms       []string `yaml:"legal_forms"`
	Streets          []string `yaml:"streets"`
	HouseNumberFirst bool     `yaml:"house_number_first"`
	Cities           []struct {
		City      string   `yaml:"city"`
		PostCodes []string `yaml:"post_codes"`
	} `yaml:"cities"`
	Phone string `yaml:"phone"`
}

type generatorFile struct {
	Countries []generatorCountry `yaml:"countries"`
}

func loadYAML(name string, out interface{}) error {
	raw, err := dataFS.ReadFile("data/" + name)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(raw, out)
	if err != nil {
		return fmt.Errorf("demo data %s: %w", name, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

// Create fills the company with demo data of the given size.
func Create(db database.Executor, company string, dbType database.DBType, opts Options) (Counts, error) {
	var counts Counts
	if opts.Size == "" {
		opts.Size = Small
	}
	today := opts.Today
	if today.IsZero() {
		today = time.Now()
	}
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)

	var setup setupData
	var small customersFile
	err := loadYAML("setup.yaml", &setup)
	if err != nil {
		return counts, err
	}
	err = loadYAML("customers.yaml", &small)
	if err != nil {
		return counts, err
	}
	var vendors vendorsFile
	err = loadYAML("vendors.yaml", &vendors)
	if err != nil {
		return counts, err
	}

	rng := newRand()
	customers := small.Customers
	if opts.Size == Large || opts.Size == Heavy {
		var gen generatorFile
		err = loadYAML("generator.yaml", &gen)
		if err != nil {
			return counts, err
		}
		total := largeCustomerCount
		if opts.Size == Heavy {
			total = heavyCustomerCount
		}
		customers = append(append([]customerData{}, customers...), generateCustomers(rng, gen, setup, total-len(customers))...)
	}

	dueDays := make(map[string]int, len(setup.PaymentTerms))
	for _, pt := range setup.PaymentTerms {
		dueDays[strings.ToUpper(pt.Code)] = pt.DueDays
	}
	entries := generateEntries(rng, customers, dueDays, today, opts.Size)
	vendorEntries := generateVendorEntries(vendors.Vendors, dueDays, today)

	// One transaction: a failed run leaves the company as it was
	exec := db
	var tx *sql.Tx
	sqlDB, ok := db.(*sql.DB)
	if ok {
		var err error
		tx, err = sqlDB.Begin()
		if err != nil {
			return counts, err
		}
		exec = tx
	}
	rollback := func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}

	var existing gtables.Customer
	existing.InitWithDBType(exec, company, dbType)
	if !existing.IsEmpty() {
		rollback()
		return counts, ErrCompanyNotEmpty
	}

	p := newProgress(opts.Progress, len(setup.Countries)+len(setup.PaymentTerms)+recordCount(setup.MasterData)+recordCount(setup.CustomerData)+recordCount(setup.VendorData), len(customers), len(entries))

	err = insertSetup(exec, company, dbType, setup, &counts, p)
	if err != nil {
		rollback()
		return Counts{}, err
	}
	err = insertCustomers(exec, company, dbType, customers, entries, today, &counts, p)
	if err != nil {
		rollback()
		return Counts{}, err
	}
	err = insertRecords(exec, company, dbType, setup.CustomerData, &counts.CustomerData, p)
	if err != nil {
		rollback()
		return Counts{}, err
	}
	err = insertAccountingPeriods(exec, company, dbType, today, &counts)
	if err != nil {
		rollback()
		return Counts{}, err
	}
	// The entries' SIFT totals are built once after the load instead of updated per entry
	err = sift.Suspend(exec, dbType, company, gtables.CustomerLedgerEntryTableName)
	if err != nil {
		rollback()
		return Counts{}, err
	}
	err = insertEntries(exec, company, dbType, entries, &counts, p)
	if err != nil {
		rollback()
		return Counts{}, err
	}
	err = (&gtables.CustomerLedgerEntry{}).SyncKeys(exec, company, dbType)
	if err != nil {
		rollback()
		return Counts{}, err
	}
	err = insertVendors(exec, company, dbType, vendors.Vendors, &counts)
	if err != nil {
		rollback()
		return Counts{}, err
	}
	err = insertVendorEntries(exec, company, dbType, vendorEntries, &counts)
	if err != nil {
		rollback()
		return Counts{}, err
	}
	err = insertRecords(exec, company, dbType, setup.VendorData, &counts.VendorData, p)
	if err != nil {
		rollback()
		return Counts{}, err
	}
	err = useSeriesNumbers(exec, company, dbType, customers, entries, vendorSeriesUse(vendors.Vendors, vendorEntries))
	if err != nil {
		rollback()
		return Counts{}, err
	}
	err = insertGLEntries(exec, company, dbType, generateGLEntries(today, opts.Size, customerNos(customers), func() *gtables.GLEntry {
		e := &gtables.GLEntry{}
		e.InitWithDBType(exec, company, dbType)
		return e
	}), &counts)
	if err != nil {
		rollback()
		return Counts{}, err
	}

	if tx != nil {
		err := tx.Commit()
		if err != nil {
			return Counts{}, err
		}
	}
	p.done()
	return counts, nil
}

func insertSetup(exec database.Executor, company string, dbType database.DBType, setup setupData, counts *Counts, p *progress) error {
	p.phase(PhaseSetup)
	for _, c := range setup.Countries {
		var rec gtables.CountryRegion
		rec.InitWithDBType(exec, company, dbType)
		if !rec.Get(c.Code) {
			rec.Code = types.NewCode(c.Code)
			rec.Name = types.NewText(c.Name)
			rec.Eu_country_code = types.NewCode(c.EUCountryCode)
			err := rec.InsertOrError(true)
			if err != nil {
				return &InsertError{Table: gtables.CountryRegionTableName, Key: c.Code, Err: err}
			}
			counts.Countries++
		}
		p.step()
	}
	for _, pt := range setup.PaymentTerms {
		var rec gtables.PaymentTerms
		rec.InitWithDBType(exec, company, dbType)
		if !rec.Get(pt.Code) {
			rec.Code = types.NewCode(pt.Code)
			rec.Description = types.NewText(pt.Description)
			rec.Due_date_calculation = types.NewCode(fmt.Sprintf("%dD", pt.DueDays))
			rec.Active = true
			err := rec.InsertOrError(true)
			if err != nil {
				return &InsertError{Table: gtables.PaymentTermsTableName, Key: pt.Code, Err: err}
			}
			counts.PaymentTerms++
		}
		p.step()
	}
	err := insertRecords(exec, company, dbType, setup.MasterData, &counts.MasterData, p)
	if err != nil {
		return err
	}
	// The chart of accounts as the Indent Chart of Accounts function leaves it
	return tables.IndentChartOfAccounts(exec, company, dbType, nil)
}

// insertRecords inserts the records of tables that do not exist yet (a setup record gets the
// values) and counts them
func insertRecords(exec database.Executor, company string, dbType database.DBType, data []tableRecords, count *int, p *progress) error {
	for _, t := range data {
		factory, ok := gtables.GetTableFactory(t.Table)
		if !ok {
			return fmt.Errorf("demo data: table %q does not exist", t.Table)
		}
		for _, record := range t.Records {
			rec := factory()
			rec.InitWithDBType(exec, company, dbType)
			rec.FromMap(record)
			key := primaryKey(rec)
			switch {
			case !rec.Get(key):
				rec.InitWithDBType(exec, company, dbType)
				rec.FromMap(record)
				err := rec.InsertOrError(true)
				if err != nil {
					return &InsertError{Table: t.Table, Key: rec.GetPrimaryKeyValue(), Err: err}
				}
				(*count)++
			case rec.IsSetupTable():
				// The setup record exists as soon as its page was opened: it gets the values
				rec.FromMap(record)
				err := rec.ModifyOrError(true)
				if err != nil {
					return &InsertError{Table: t.Table, Key: "", Err: err}
				}
				(*count)++
			}
			p.step()
		}
	}
	return nil
}

// insertAccountingPeriods creates the monthly accounting periods of the fiscal years from two
// years ago to this year and the first period of the next year, as Create Fiscal Year does,
// and closes the oldest year as Fiscal Year-Close does
func insertAccountingPeriods(exec database.Executor, company string, dbType database.DBType, today time.Time, counts *Counts) error {
	accountingPeriod := gtables.NewAccountingPeriod(ftables.DBSource(exec, company, dbType))
	if !accountingPeriod.IsEmpty() {
		return nil
	}
	firstYear := today.Year() - 2
	for i := 0; i <= 3*12; i++ {
		startingDate := types.NewDate(firstYear+i/12, time.Month(i%12+1), 1)
		accountingPeriod.Init()
		err := accountingPeriod.Validate("starting_date", startingDate)
		if err != nil {
			return err
		}
		accountingPeriod.New_fiscal_year = i%12 == 0
		// the closed year and the first period of the year after it are date locked
		accountingPeriod.Closed = i < 12
		accountingPeriod.Date_locked = i <= 12
		err = accountingPeriod.InsertOrError(false)
		if err != nil {
			return &InsertError{Table: gtables.AccountingPeriodTableName, Key: startingDate.String(), Err: err}
		}
		counts.AccountingPeriods++
	}
	return nil
}

func insertCustomers(exec database.Executor, company string, dbType database.DBType, customers []customerData, entries []*ledgerEntry, today time.Time, counts *Counts, p *progress) error {
	p.phase(PhaseCustomers)
	salespeople := []string{"AH", "JB", "PL"}
	for i, c := range customers {
		var rec gtables.Customer
		rec.InitWithDBType(exec, company, dbType)
		rec.InitRecord()
		rec.No = types.NewCode(c.No)
		rec.Name = types.NewText(c.Name)
		rec.Search_name = types.NewCode(system.CopyStr(c.Name, 1, 50))
		rec.Address = types.NewText(c.Address)
		rec.Post_code = types.NewCode(c.PostCode)
		rec.City = types.NewText(c.City)
		rec.Country_code = types.NewCode(c.Country)
		rec.Phone_no = types.NewText(c.Phone)
		rec.Payment_terms_code = types.NewCode(c.PaymentTerms)
		rec.Credit_limit_lcy = types.NewDecimal(c.CreditLimit)
		rec.Blocked = c.Blocked
		rec.Invoice_disc_code = rec.No
		rec.Salesperson_code = types.NewCode(salespeople[i%len(salespeople)])
		rec.Payment_method_code = "BANK"
		rec.Fin_charge_terms_code = "STD"
		rec.Price_group_code = "RETAIL"
		rec.Print_statements = true
		switch {
		case c.CreditLimit >= 250000:
			rec.Cust_item_disc_gr = "LARGE"
		case c.CreditLimit >= 100000:
			rec.Cust_item_disc_gr = "WHOLESALE"
		default:
			rec.Cust_item_disc_gr = "RETAIL"
		}
		// Posting groups, tax, reminders and shipping by country: domestic (Norway) or abroad
		if c.Country == "NO" {
			rec.Customer_posting_group, rec.Gen_bus_posting_group, rec.Tax_bus_posting_group = "DOMESTIC", "DOMESTIC", "DOMESTIC"
			rec.Tax_area_code, rec.Reminder_terms_code = "DOMESTIC", "DOMESTIC"
			rec.Shipping_agent_code, rec.Shipment_method_code, rec.Location_code = "POSTEN", "POST", "MAIN"
			rec.Territory_code = types.NewCode(norwegianTerritory(c.PostCode))
			rec.Tax_liable = true
		} else {
			rec.Customer_posting_group, rec.Gen_bus_posting_group, rec.Tax_bus_posting_group = "FOREIGN", "EXPORT", "EXPORT"
			rec.Tax_area_code, rec.Reminder_terms_code = "ABROAD", "FOREIGN"
			rec.Shipping_agent_code, rec.Shipment_method_code, rec.Location_code = "DHL", "TRUCK", "MAIN"
			rec.Territory_code = "NORDIC"
			if c.Country == "GB" {
				rec.Territory_code = "UK"
			}
		}
		err := rec.InsertOrError(true)
		if err != nil {
			return &InsertError{Table: gtables.CustomerTableName, Key: c.No, Err: err}
		}
		counts.Customers++
		p.step()
	}
	return nil
}

func insertEntries(exec database.Executor, company string, dbType database.DBType, entries []*ledgerEntry, counts *Counts, p *progress) error {
	p.phase(PhaseEntries)
	var last gtables.CustomerLedgerEntry
	last.InitWithDBType(exec, company, dbType)
	firstNo := 1
	if last.FindLast() {
		firstNo = last.Entry_no + 1
	}
	for i, e := range entries {
		e.entryNo = firstNo + i
	}
	// Multi-row inserts (InsertAll) in chunks: one round trip per chunk instead of one per
	// entry (HEAVY: 190,000 entries); every entry still runs its OnInsert trigger
	const chunk = 1000
	batch := make([]*gtables.CustomerLedgerEntry, 0, chunk)
	flush := func(first int) error {
		err := last.InsertAll(batch, true)
		if err != nil {
			var batchErr *ftables.BatchInsertError
			key, cause := "", err
			if errors.As(err, &batchErr) {
				key, cause = fmt.Sprint(entries[first+batchErr.Index].entryNo), batchErr.Err
			}
			return &InsertError{Table: gtables.CustomerLedgerEntryTableName, Key: key, Err: cause}
		}
		counts.LedgerEntries += len(batch)
		for range batch {
			p.step()
		}
		batch = batch[:0]
		return nil
	}
	for i, e := range entries {
		rec := &gtables.CustomerLedgerEntry{}
		rec.InitWithDBType(exec, company, dbType)
		rec.Entry_no = e.entryNo
		rec.Customer_no = types.NewCode(e.customerNo)
		rec.Sell_to_customer_no = types.NewCode(e.customerNo)
		rec.Posting_date = types.NewDateFromTime(e.date)
		rec.Document_date = types.NewDateFromTime(e.date)
		rec.Document_type = e.docType
		rec.Document_no = types.NewCode(e.docNo)
		rec.Description = types.NewText(e.description)
		rec.Amount = types.NewDecimal(e.amount)
		rec.Amount_lcy = types.NewDecimal(e.amount)
		rec.Original_amount_lcy = types.NewDecimal(e.amount)
		tables.SetDebitCredit(rec)
		rec.Remaining_amount = types.NewDecimal(e.remaining)
		rec.Remaining_amt_lcy = types.NewDecimal(e.remaining)
		rec.Sales_lcy = types.NewDecimal(e.sales)
		rec.Open = e.open
		rec.Positive = e.amount > 0
		if !e.due.IsZero() {
			rec.Due_date = types.NewDateFromTime(e.due)
		}
		if e.closedBy != nil {
			rec.Closed_by_entry_no = e.closedBy.entryNo
			rec.Closed_at_date = types.NewDateFromTime(e.closedAt)
		}
		if e.docType == gtables.CustomerLedgerEntry_Document_type.Payment {
			rec.Bal_account_type = gtables.CustomerLedgerEntry_Bal_account_type.BankAccount
		}
		batch = append(batch, rec)
		if len(batch) == chunk {
			err := flush(i + 1 - chunk)
			if err != nil {
				return err
			}
		}
	}
	if len(batch) > 0 {
		return flush(len(entries) - len(batch))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Generation
// ---------------------------------------------------------------------------

// norwegianTerritory is the demo territory of a Norwegian post code: east (0–3), west (4–6),
// north (7–9)
func norwegianTerritory(postCode string) string {
	switch {
	case postCode == "":
		return ""
	case postCode[0] <= '3':
		return "EAST"
	case postCode[0] <= '6':
		return "WEST"
	}
	return "NORTH"
}

// generateCustomers builds n customers from the generator building blocks, cycling
// through the countries. Numbers continue after the SMALL set (C00201, C00202, ...).
func generateCustomers(rng *rand.Rand, gen generatorFile, setup setupData, n int) []customerData {
	pick := func(list []string) string { return list[rng.Intn(len(list))] }
	terms := make([]string, len(setup.PaymentTerms))
	for i, pt := range setup.PaymentTerms {
		terms[i] = pt.Code
	}
	out := make([]customerData, 0, n)
	for i := 0; i < n; i++ {
		g := gen.Countries[i%len(gen.Countries)]
		city := g.Cities[rng.Intn(len(g.Cities))]
		street, number := pick(g.Streets), 1+rng.Intn(120)
		address := fmt.Sprintf("%s %d", street, number)
		if g.HouseNumberFirst {
			address = fmt.Sprintf("%d %s", number, street)
		}
		out = append(out, customerData{
			No:           fmt.Sprintf("C%05d", 201+i),
			Name:         pick(g.First) + pick(g.Second) + pick(g.LegalForms),
			Address:      address,
			PostCode:     pick(city.PostCodes),
			City:         city.City,
			Country:      g.Country,
			Phone:        fillDigits(rng, g.Phone),
			PaymentTerms: pick(terms),
			CreditLimit:  float64(rng.Intn(41) * 10000),
			Blocked:      rng.Float64() < 0.02,
		})
	}
	return out
}

// fillDigits replaces each # in pattern with a random digit.
func fillDigits(rng *rand.Rand, pattern string) string {
	var b strings.Builder
	for _, r := range pattern {
		if r == '#' {
			b.WriteByte(byte('0' + rng.Intn(10)))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

type ledgerEntry struct {
	customerNo  string
	date        time.Time
	docType     gtables.CustomerLedgerEntryDocument_type
	docNo       string
	description string
	amount      float64
	remaining   float64
	sales       float64
	open        bool
	due         time.Time
	closedBy    *ledgerEntry
	closedAt    time.Time
	entryNo     int
}

// generateEntries builds a year of invoices per customer, with payments for most
// invoices past their due date and the odd credit memo. Entries are returned in
// posting date order, the order they get their entry numbers in.
func generateEntries(rng *rand.Rand, customers []customerData, dueDays map[string]int, today time.Time, size Size) []*ledgerEntry {
	var entries []*ledgerEntry
	day := func(t time.Time, days int) time.Time { return t.AddDate(0, 0, days) }
	clamp := func(t, min time.Time) time.Time {
		if t.Before(min) {
			t = min
		}
		if t.After(today) {
			t = today
		}
		return t
	}

	for _, c := range customers {
		n, days := 6+rng.Intn(15), 365
		switch size {
		case Large:
			n = 3 + rng.Intn(10)
		case Heavy:
			n, days = 450+rng.Intn(100), 730
		}
		for i := 0; i < n; i++ {
			date := day(today, -rng.Intn(days))
			amount := round2(500 + rng.Float64()*24500)
			inv := &ledgerEntry{
				customerNo: c.No,
				date:       date,
				docType:    gtables.CustomerLedgerEntry_Document_type.Invoice,
				amount:     amount,
				remaining:  amount,
				sales:      amount,
				open:       true,
				due:        day(date, dueDays[strings.ToUpper(c.PaymentTerms)]),
			}
			entries = append(entries, inv)

			if rng.Float64() < 0.06 {
				credit := -round2(amount * (0.1 + rng.Float64()*0.2))
				entries = append(entries, &ledgerEntry{
					customerNo: c.No,
					date:       clamp(day(date, 1+rng.Intn(10)), date),
					docType:    gtables.CustomerLedgerEntry_Document_type.CreditMemo,
					amount:     credit,
					sales:      credit,
					due:        date,
				})
				inv.remaining = round2(inv.remaining + credit)
			}

			if inv.due.Before(day(today, -5)) && rng.Float64() < 0.88 {
				payDate := clamp(day(inv.due, rng.Intn(16)-5), date)
				pay := &ledgerEntry{
					customerNo: c.No,
					date:       payDate,
					docType:    gtables.CustomerLedgerEntry_Document_type.Payment,
					amount:     -inv.remaining,
					closedBy:   inv,
					closedAt:   payDate,
				}
				entries = append(entries, pay)
				inv.remaining = 0
				inv.open = false
				inv.closedBy = pay
				inv.closedAt = payDate
			}
		}
	}

	sort.SliceStable(entries, func(i, j int) bool { return entries[i].date.Before(entries[j].date) })

	var invoices, credits, payments int
	for _, e := range entries {
		switch e.docType {
		case gtables.CustomerLedgerEntry_Document_type.Invoice:
			invoices++
			e.docNo = fmt.Sprintf("SI%06d", invoices)
			e.description = "Invoice " + e.docNo
		case gtables.CustomerLedgerEntry_Document_type.CreditMemo:
			credits++
			e.docNo = fmt.Sprintf("SCM%06d", credits)
			e.description = "Credit Memo " + e.docNo
		default:
			payments++
			e.docNo = fmt.Sprintf("PAY%06d", payments)
		}
	}
	// Payment descriptions name the invoice they settle (numbered above)
	for _, e := range entries {
		if e.docType == gtables.CustomerLedgerEntry_Document_type.Payment {
			e.description = "Payment " + e.closedBy.docNo
		}
	}
	return entries
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// ---------------------------------------------------------------------------
// Progress
// ---------------------------------------------------------------------------

// progress turns record counts into an overall percentage: setup 0-5%,
// customers 5-30%, ledger entries 30-100%.
type progress struct {
	report      func(int, Phase)
	current     Phase
	start, span float64
	total, n    int
	last        int
	sizes       map[Phase]int
}

func newProgress(report func(int, Phase), setup, customers, entries int) *progress {
	return &progress{report: report, last: -1, sizes: map[Phase]int{PhaseSetup: setup, PhaseCustomers: customers, PhaseEntries: entries}}
}

func (p *progress) phase(ph Phase) {
	ranges := map[Phase][2]float64{PhaseSetup: {0, 5}, PhaseCustomers: {5, 25}, PhaseEntries: {30, 70}}
	p.current, p.start, p.span = ph, ranges[ph][0], ranges[ph][1]
	p.total, p.n = p.sizes[ph], 0
	p.emit(int(p.start))
}

func (p *progress) step() {
	p.n++
	if p.total > 0 {
		p.emit(int(p.start + p.span*float64(p.n)/float64(p.total)))
	}
}

func (p *progress) done() { p.emit(100) }

func (p *progress) emit(pct int) {
	if p.report == nil || pct == p.last {
		return
	}
	p.last = pct
	p.report(pct, p.current)
}

// useSeriesNumbers marks the numbers the demo customers and posted documents have as used in
// their number series (CUST, S-INV+, S-CR+; more: the vendors' VEND, P-INV+), so the next
// number continues after them
func useSeriesNumbers(exec database.Executor, company string, dbType database.DBType, customers []customerData, entries []*ledgerEntry, more map[string]ledgerUse) error {
	last := map[string]ledgerUse{}
	for series, u := range more {
		last[series] = u
	}
	use := func(series, no string, date time.Time) {
		u := last[series]
		if no > u.no {
			last[series] = ledgerUse{no, date}
		}
	}
	for _, c := range customers {
		use("CUST", c.No, time.Time{})
	}
	for _, e := range entries {
		switch e.docType {
		case gtables.CustomerLedgerEntry_Document_type.Invoice:
			use("S-INV+", e.docNo, e.date)
		case gtables.CustomerLedgerEntry_Document_type.CreditMemo:
			use("S-CR+", e.docNo, e.date)
		}
	}
	for series, u := range last {
		var line gtables.NoSeriesLine
		line.InitWithDBType(exec, company, dbType)
		if !line.GetByPK(types.Code(series), 10000) {
			continue
		}
		line.Last_no_used = types.Code(u.no)
		if !u.date.IsZero() {
			line.Last_date_used = types.NewDateFromTime(u.date)
		}
		err := line.ModifyOrError(false)
		if err != nil {
			return &InsertError{Table: gtables.NoSeriesLineTableName, Key: series, Err: err}
		}
	}
	return nil
}

type ledgerUse struct {
	no   string
	date time.Time
}

// primaryKey is a record's key for Get: the value of its key field, or field → value for a
// composite key (e.g. currency and starting date of an exchange rate)
func primaryKey(rec ftables.Table) interface{} {
	values := rec.ToMap()
	key := map[string]interface{}{}
	for _, f := range rec.GetFields() {
		if f.PrimaryKey {
			key[f.Name] = values[f.Name]
		}
	}
	if len(key) == 1 {
		return rec.GetPrimaryKeyValue()
	}
	return key
}

// customerNos are the numbers of the demo customers (balancing accounts of the G/L entries)
func customerNos(customers []customerData) []string {
	nos := make([]string, 0, len(customers))
	for _, c := range customers {
		nos = append(nos, c.No)
	}
	return nos
}
