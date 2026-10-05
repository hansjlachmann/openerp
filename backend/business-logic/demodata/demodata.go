// Package demodata fills a company with demo data (BC/NAV Contoso-style): countries,
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

	"gopkg.in/yaml.v3"

	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:embed data/*.yaml
var dataFS embed.FS

// Size selects how much data is created.
type Size string

const (
	Small Size = "SMALL" // the customers in customers.yaml (screenshots, demos, E2E)
	Large Size = "LARGE" // ~10,000 customers (paging, search and FlowField load tests)
)

// largeCustomerCount is the total number of customers in the LARGE set.
const largeCustomerCount = 10000

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
	Countries     int
	PaymentTerms  int
	Customers     int
	LedgerEntries int
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
		Code string `yaml:"code"`
		Name string `yaml:"name"`
	} `yaml:"countries"`
	PaymentTerms []struct {
		Code        string `yaml:"code"`
		Description string `yaml:"description"`
		DueDays     int    `yaml:"due_days"`
	} `yaml:"payment_terms"`
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
	if err := yaml.Unmarshal(raw, out); err != nil {
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
	if err := loadYAML("setup.yaml", &setup); err != nil {
		return counts, err
	}
	if err := loadYAML("customers.yaml", &small); err != nil {
		return counts, err
	}

	rng := newRand()
	customers := small.Customers
	if opts.Size == Large {
		var gen generatorFile
		if err := loadYAML("generator.yaml", &gen); err != nil {
			return counts, err
		}
		customers = append(append([]customerData{}, customers...), generateCustomers(rng, gen, setup, largeCustomerCount-len(customers))...)
	}

	dueDays := make(map[string]int, len(setup.PaymentTerms))
	for _, pt := range setup.PaymentTerms {
		dueDays[strings.ToUpper(pt.Code)] = pt.DueDays
	}
	entries := generateEntries(rng, customers, dueDays, today, opts.Size)

	// One transaction: a failed run leaves the company as it was
	exec := db
	var tx *sql.Tx
	if sqlDB, ok := db.(*sql.DB); ok {
		var err error
		if tx, err = sqlDB.Begin(); err != nil {
			return counts, err
		}
		exec = tx
	}
	rollback := func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}

	var existing tables.Customer
	existing.InitWithDBType(exec, company, dbType)
	if !existing.IsEmpty() {
		rollback()
		return counts, ErrCompanyNotEmpty
	}

	p := newProgress(opts.Progress, len(setup.Countries)+len(setup.PaymentTerms), len(customers), len(entries))

	if err := insertSetup(exec, company, dbType, setup, &counts, p); err != nil {
		rollback()
		return Counts{}, err
	}
	if err := insertCustomers(exec, company, dbType, customers, entries, today, &counts, p); err != nil {
		rollback()
		return Counts{}, err
	}
	if err := insertEntries(exec, company, dbType, entries, &counts, p); err != nil {
		rollback()
		return Counts{}, err
	}

	if tx != nil {
		if err := tx.Commit(); err != nil {
			return Counts{}, err
		}
	}
	p.done()
	return counts, nil
}

func insertSetup(exec database.Executor, company string, dbType database.DBType, setup setupData, counts *Counts, p *progress) error {
	p.phase(PhaseSetup)
	for _, c := range setup.Countries {
		var rec tables.CountryRegion
		rec.InitWithDBType(exec, company, dbType)
		if !rec.Get(c.Code) {
			rec.Code = types.NewCode(c.Code)
			rec.Name = types.NewText(c.Name)
			if !rec.Insert(true) {
				return &InsertError{Table: gtables.CountryRegionTableName, Key: c.Code, Err: insertErr(rec.TriggerError())}
			}
			counts.Countries++
		}
		p.step()
	}
	for _, pt := range setup.PaymentTerms {
		var rec tables.PaymentTerms
		rec.InitWithDBType(exec, company, dbType)
		if !rec.Get(pt.Code) {
			rec.Code = types.NewCode(pt.Code)
			rec.Description = types.NewText(pt.Description)
			rec.Active = true
			if !rec.Insert(true) {
				return &InsertError{Table: tables.PaymentTermsTableName, Key: pt.Code, Err: insertErr(rec.TriggerError())}
			}
			counts.PaymentTerms++
		}
		p.step()
	}
	return nil
}

func insertCustomers(exec database.Executor, company string, dbType database.DBType, customers []customerData, entries []*ledgerEntry, today time.Time, counts *Counts, p *progress) error {
	p.phase(PhaseCustomers)
	lastOrder := make(map[string]time.Time)
	for _, e := range entries {
		if e.docType == gtables.CustomerLedgerEntry_Document_type.Invoice && e.date.After(lastOrder[e.customerNo]) {
			lastOrder[e.customerNo] = e.date
		}
	}
	created := types.NewDateTimeFromTime(today.Add(-366 * 24 * time.Hour))
	for _, c := range customers {
		var rec tables.Customer
		rec.InitWithDBType(exec, company, dbType)
		rec.No = types.NewCode(c.No)
		rec.Name = types.NewText(c.Name)
		rec.Address = types.NewText(c.Address)
		rec.Post_code = types.NewCode(c.PostCode)
		rec.City = types.NewText(c.City)
		rec.Country_region_code = types.NewCode(c.Country)
		rec.Phonenumber = types.NewText(c.Phone)
		rec.Payment_terms_code = types.NewCode(c.PaymentTerms)
		rec.Credit_limit = types.NewDecimal(c.CreditLimit)
		rec.Created_at = created
		if d, ok := lastOrder[c.No]; ok {
			rec.Last_order_date = types.NewDateFromTime(d)
		}
		if !rec.Insert(true) {
			return &InsertError{Table: tables.CustomerTableName, Key: c.No, Err: insertErr(rec.TriggerError())}
		}
		// OnInsert sets every new customer to Open; block afterwards, as a user would
		if c.Blocked {
			rec.Status = tables.Customer_Status.Blocked
			if !rec.Modify(true) {
				return &InsertError{Table: tables.CustomerTableName, Key: c.No, Err: insertErr(rec.TriggerError())}
			}
		}
		counts.Customers++
		p.step()
	}
	return nil
}

func insertEntries(exec database.Executor, company string, dbType database.DBType, entries []*ledgerEntry, counts *Counts, p *progress) error {
	p.phase(PhaseEntries)
	var last tables.CustomerLedgerEntry
	last.InitWithDBType(exec, company, dbType)
	firstNo := 1
	if last.FindLast() {
		firstNo = last.Entry_no + 1
	}
	for i, e := range entries {
		e.entryNo = firstNo + i
	}
	for _, e := range entries {
		var rec tables.CustomerLedgerEntry
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
		if !rec.Insert(true) {
			return &InsertError{Table: tables.CustomerLedgerEntryTableName, Key: fmt.Sprint(e.entryNo), Err: insertErr(rec.TriggerError())}
		}
		counts.LedgerEntries++
		p.step()
	}
	return nil
}

// insertErr gives a failed Insert a cause: the trigger error, or a database error
// (the generated Insert logs those and only returns false).
func insertErr(triggerErr error) error {
	if triggerErr != nil {
		return triggerErr
	}
	return errors.New("database insert failed")
}

// ---------------------------------------------------------------------------
// Generation
// ---------------------------------------------------------------------------

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
		n := 6 + rng.Intn(15)
		if size == Large {
			n = 3 + rng.Intn(10)
		}
		for i := 0; i < n; i++ {
			date := day(today, -rng.Intn(365))
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
