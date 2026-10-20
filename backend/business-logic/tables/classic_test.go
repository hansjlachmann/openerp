package tables

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// TransferFields copies by field number and type, also between tables
func TestTransferFields(t *testing.T) {
	from := &gtables.Customer{}
	from.No, from.Name, from.City = types.NewCode("C1"), types.NewText("Acme"), types.NewText("Oslo")
	from.Status = gtables.Customer_Status.Blocked

	to := &gtables.Customer{}
	to.No = types.NewCode("C2")
	to.TransferFields(from, false) // keep the key
	if to.No != "C2" || to.Name != "Acme" || to.City != "Oslo" || to.Status != gtables.Customer_Status.Blocked {
		t.Errorf("same table: %+v", to.ToMap())
	}
	to.TransferFields(from)
	if to.No != "C1" {
		t.Errorf("with the key: no = %s", to.No)
	}

	// Country/Region 1 = Code, 2 = Name: the same numbers and types as Customer No. and Name
	var country gtables.CountryRegion
	country.TransferFields(from)
	if country.Code != "C1" || country.Name != "Acme" {
		t.Errorf("other table: %+v", country.ToMap())
	}
	// A value of another type is not set
	if country.SetFieldValue(2, types.NewCode("X")) || country.Name != "Acme" {
		t.Error("a Code value was set into a Text field")
	}
}

func TestTestFieldAndFieldError(t *testing.T) {
	var c gtables.Customer
	err := c.TestField("name")
	var appErr *apperrors.AppError
	if !asAppError(err, &appErr) || appErr.Code != apperrors.ErrRequiredField || appErr.Field != "name" {
		t.Errorf("empty name: %v", err)
	}
	c.Name = types.NewText("Acme")
	if err := c.TestField("Name"); err != nil {
		t.Errorf("name with a value: %v", err)
	}
	c.Status = gtables.Customer_Status.Open
	if err := c.TestField("status", gtables.Customer_Status.Open); err != nil {
		t.Errorf("status Open: %v", err)
	}
	err = c.TestField("status", gtables.Customer_Status.Blocked)
	if !asAppError(err, &appErr) || appErr.Code != apperrors.ErrFieldMustBe || appErr.Message("en-US") != "Status must be Blocked" {
		t.Errorf("status Blocked: %v", err)
	}
	if err := c.TestField("credit_limit", types.NewDecimal(0)); err != nil {
		t.Errorf("decimal 0: %v", err)
	}
	if err := c.TestField("bogus"); err == nil {
		t.Error("unknown field accepted")
	}
	err = c.FieldError("City", "is not allowed for this customer")
	if !asAppError(err, &appErr) || appErr.Message("en-US") != "City is not allowed for this customer" {
		t.Errorf("FieldError: %v", err)
	}
}

func asAppError(err error, target **apperrors.AppError) bool {
	e, ok := err.(*apperrors.AppError)
	*target = e
	return ok
}

func TestCalcSumsChangeCompanyInit(t *testing.T) {
	db := newTestDB(t)
	for _, company := range []string{"c1", "c2"} {
		if err := (&gtables.CustomerLedgerEntry{}).CreateTableWithDBType(db, company, database.DBTypeSQLite); err != nil {
			t.Fatal(err)
		}
	}
	if err := (&gtables.Company{}).CreateTableWithDBType(db, "", database.DBTypeSQLite); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO "Company" (name) VALUES ('c1'), ('c2')`); err != nil {
		t.Fatal(err)
	}
	add := func(company string, no int, customer, amount string) {
		e := &gtables.CustomerLedgerEntry{}
		e.InitWithDBType(db, company, database.DBTypeSQLite)
		e.Entry_no, e.Customer_no, e.Amount_lcy = no, types.NewCode(customer), types.MustDecimal(amount)
		if !e.Insert(false) {
			t.Fatalf("insert %d", no)
		}
	}
	add("c1", 1, "C1", "100.50")
	add("c1", 2, "C1", "-20.25")
	add("c1", 3, "C2", "1000")
	add("c2", 1, "C1", "7")

	var e gtables.CustomerLedgerEntry
	e.InitWithDBType(db, "c1", database.DBTypeSQLite)
	e.SetRange("customer_no", "C1")
	if err := e.CalcSums("amount_lcy", "entry_no"); err != nil {
		t.Fatal(err)
	}
	if e.Amount_lcy.String() != "80.25" || e.Entry_no != 3 {
		t.Errorf("sums: amount %s, entry_no %d", e.Amount_lcy, e.Entry_no)
	}
	if err := e.CalcSums("description"); err == nil {
		t.Error("CalcSums on a text field accepted")
	}

	// ChangeCompany: the same filters in another company
	if e.CompanyName() != "c1" || e.ChangeCompany("nope") || e.CompanyName() != "c1" {
		t.Error("ChangeCompany to a company that does not exist")
	}
	if !e.ChangeCompany("c2") || e.CompanyName() != "c2" || e.Count() != 1 {
		t.Errorf("ChangeCompany c2: %s, %d entries", e.CompanyName(), e.Count())
	}

	// Init keeps the primary key
	e.Entry_no, e.Description = 5, types.NewText("x")
	e.Init()
	if e.Entry_no != 5 || e.Description != "" {
		t.Errorf("Init: entry %d, description %q", e.Entry_no, e.Description)
	}
}

func TestCaptionsAndValidate(t *testing.T) {
	var c gtables.Customer
	if c.TableCaption() != "Customer" || c.FieldCaption("name") != "Name" || !strings.Contains(c.TableCaption("nb-NO"), "Kunde") {
		t.Errorf("captions %q %q %q", c.TableCaption(), c.FieldCaption("name"), c.TableCaption("nb-NO"))
	}
	if err := c.Validate("name", "ab"); err == nil {
		t.Error("Validate must run OnValidate_Name")
	}
}

// recordingDB records the SQL a record sends (no database behind it)
type recordingDB struct{ queries []string }

func (r *recordingDB) Exec(q string, _ ...interface{}) (sql.Result, error) {
	r.queries = append(r.queries, q)
	return nil, errors.New("no database")
}
func (r *recordingDB) Query(q string, _ ...interface{}) (*sql.Rows, error) {
	r.queries = append(r.queries, q)
	return nil, errors.New("no database")
}
func (r *recordingDB) QueryRow(q string, _ ...interface{}) *sql.Row {
	r.queries = append(r.queries, q)
	return &sql.Row{}
}
func (r *recordingDB) Prepare(q string) (*sql.Stmt, error) { return nil, errors.New("no database") }

func TestLockTableReadsForUpdate(t *testing.T) {
	db := &recordingDB{}
	var c gtables.Customer
	c.InitWithDBType(db, "c1", database.DBTypePostgres)
	c.FindSet()
	c.LockTable()
	c.FindSet()
	if len(db.queries) != 2 || strings.HasSuffix(db.queries[0], "FOR UPDATE") || !strings.HasSuffix(db.queries[1], " FOR UPDATE") {
		t.Errorf("queries %q", db.queries)
	}
	// SQLite has no FOR UPDATE
	lite := &recordingDB{}
	c.InitWithDBType(lite, "c1", database.DBTypeSQLite)
	c.LockTable()
	c.FindSet()
	if len(lite.queries) != 1 || strings.Contains(lite.queries[0], "FOR UPDATE") {
		t.Errorf("SQLite queries %q", lite.queries)
	}
}
