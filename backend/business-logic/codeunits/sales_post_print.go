package codeunits

import (
	"encoding/base64"
	"errors"

	"github.com/hansjlachmann/openerp/backend/business-logic/reports"
	fcodeunits "github.com/hansjlachmann/openerp/backend/foundation/codeunits"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// SalesPostPrint - Codeunit 82 "Sales-Post + Print": Post and Print (Shift+F11) of a sales
// document — posts it as Sales-Post (Yes/No) does, then prints the posted invoice or credit memo
// (report 206, 207), or the shipment when the order was only shipped (208), to the browser's print
// dialog.
const SalesPostPrintID = 82
const SalesPostPrintName = "sales-post-print"

func init() {
	Register(SalesPostPrintID, SalesPostPrintName, NewSalesPostPrint)
}

type SalesPostPrint struct {
	db      database.Executor
	company string
	dbType  database.DBType
}

// NewSalesPostPrint creates a new instance of the codeunit
func NewSalesPostPrint(db database.Executor, company string, dbType database.DBType) fcodeunits.Codeunit {
	return &SalesPostPrint{db: db, company: company, dbType: dbType}
}

func (c *SalesPostPrint) ID() int                    { return SalesPostPrintID }
func (c *SalesPostPrint) Name() string               { return SalesPostPrintName }
func (c *SalesPostPrint) SourceTable() string        { return gtables.SalesHeaderTableName }
func (c *SalesPostPrint) UsesProgress() bool         { return true }
func (c *SalesPostPrint) GetDB() database.Executor   { return c.db }
func (c *SalesPostPrint) GetCompany() string         { return c.company }
func (c *SalesPostPrint) GetDBType() database.DBType { return c.dbType }

// Run posts the document and prints what was posted
func (c *SalesPostPrint) Run(record interface{}) (fcodeunits.Result, error) {
	result, err := c.code(record)
	return result, err
}

func (c *SalesPostPrint) code(record interface{}) (fcodeunits.Result, error) {
	salesPostYesNo := &SalesPostYesNo{db: c.db, company: c.company, dbType: c.dbType}
	rec, ok := record.(*gtables.SalesHeader)
	if !ok {
		return fcodeunits.Result{}, errors.New("codeunit 82 runs on a sales header")
	}
	result, salesPost, err := salesPostYesNo.post(rec)
	if err != nil || salesPost == nil {
		return result, err
	}
	src := ftables.DBSource(c.db, c.company, c.dbType)
	var reportID int
	var context interface{}
	switch {
	case salesPost.InvoiceNo != "":
		reportID = reports.SalesInvoiceID
		salesInvHeader := gtables.NewSalesInvoiceHeader(src)
		salesInvHeader.No = salesPost.InvoiceNo
		context = salesInvHeader
	case salesPost.CrMemoNo != "":
		reportID = reports.SalesCrMemoID
		salesCrMemoHeader := gtables.NewSalesCrMemoHeader(src)
		salesCrMemoHeader.No = salesPost.CrMemoNo
		context = salesCrMemoHeader
	case salesPost.ShipmentNo != "":
		reportID = reports.SalesShipmentID
		salesShptHeader := gtables.NewSalesShipmentHeader(src)
		salesShptHeader.No = salesPost.ShipmentNo
		context = salesShptHeader
	default:
		return result, nil
	}
	printout, err := reports.Execute(reportID, c.db, c.company, c.dbType, reports.RunOptions{Output: reports.OutputPrint, Context: context})
	if err != nil {
		return fcodeunits.Result{}, err
	}
	if result.Data == nil {
		result.Data = map[string]interface{}{}
	}
	result.Data["pdf"] = base64.StdEncoding.EncodeToString(printout.PDF)
	result.Data["filename"] = printout.FileName
	result.Data["output"] = printout.Output
	return result, nil
}
