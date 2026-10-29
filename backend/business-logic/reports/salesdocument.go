package reports

import (
	"github.com/hansjlachmann/openerp/backend/business-logic/formataddress"
	"github.com/hansjlachmann/openerp/backend/business-logic/salespost"
	"github.com/hansjlachmann/openerp/backend/business-logic/salesprinted"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// salesDocument is the printout of a sales document of one type — report 204 Sales - Quote
// (salesquote.go), 205 Order Confirmation (orderconfirmation.go): the company (with its logo),
// the bill-to and ship-to addresses, the document's terms, its lines and totals, in copies; a
// print that is not a preview counts in the document's No. Printed (codeunit 313). Run from a
// document it prints that document. Each report has its layout and texts.
type salesDocument struct {
	documentType gtables.SalesHeaderDocument_type
	companyInfo  *gtables.CompanyInformation
	salesHeader  *gtables.SalesHeader
	noOfLoops    int
	printed      int // documents printed
}

func (r *salesDocument) Definition() *Definition {
	return &Definition{
		Options: []Option{
			{Name: "no_of_copies", Type: "number", Default: "0"},
			{Name: "print_company_address", Type: "bool", Default: "true"},
		},
		ContextTable: gtables.SalesHeaderTableName,
		OnInitReport: r.onInitReport,
		OnPreReport:  r.onPreReport,
		DataItems: []*DataItem{{
			Name:                "SalesHeader",
			Table:               gtables.SalesHeaderTableName,
			Key:                 []string{"document_type", "no"},
			RequestFilterFields: []string{"no", "sell_to_customer_no", "bill_to_customer_no", "no_printed"},
			OnPreDataItem:       Trigger(r.salesHeaderOnPreDataItem),
			OnAfterGetRecord:    Trigger(r.salesHeaderOnAfterGetRecord),
			OnPostDataItem:      Trigger(r.salesHeaderOnPostDataItem),
			DataItems: []*DataItem{{
				Name:             "CopyLoop",
				Table:            IntegerTable,
				OnPreDataItem:    Trigger(r.copyLoopOnPreDataItem),
				OnAfterGetRecord: Trigger(r.copyLoopOnAfterGetRecord),
				OnPostDataItem:   Trigger(r.copyLoopOnPostDataItem),
				DataItems: []*DataItem{{
					Name:             "SalesLine",
					Table:            gtables.SalesLineTableName,
					Key:              []string{"document_type", "document_no", "line_no"},
					OnPreDataItem:    Trigger(r.salesLineOnPreDataItem),
					OnAfterGetRecord: Trigger(r.salesLineOnAfterGetRecord),
				}},
			}},
		}},
	}
}

// onInitReport: run from a document (the card or the list), the request page starts filtered
// to it
func (r *salesDocument) onInitReport(run *Run) error {
	salesHeader, ok := run.Context().(*gtables.SalesHeader)
	if ok && salesHeader.No != "" {
		run.SetRequestFilter("SalesHeader", "no", salesHeader.No.String())
	}
	return nil
}

func (r *salesDocument) onPreReport(run *Run) error {
	r.companyInfo = gtables.GetCompanyInformation(run)
	if run.OptionBool("print_company_address") {
		companyAddress, err := formataddress.Company(r.companyInfo)
		if err != nil {
			return err
		}
		run.SetValue("company_address", companyAddress)
		run.SetValue("company_picture", r.companyInfo.Picture)
		run.SetValue("company_phone_no", r.companyInfo.Phone_no.String())
		run.SetValue("company_e_mail", r.companyInfo.E_mail.String())
		run.SetValue("company_vat_registration_no", r.companyInfo.Vat_registration_no.String())
	}
	return nil
}

func (r *salesDocument) salesHeaderOnPreDataItem(rec *gtables.SalesHeader, run *Run) error {
	rec.FilterGroup(2)
	rec.SetRange("document_type", r.documentType)
	rec.FilterGroup(0)
	return nil
}

func (r *salesDocument) salesHeaderOnAfterGetRecord(rec *gtables.SalesHeader, run *Run) error {
	r.salesHeader = rec
	billToAddress, err := formataddress.SalesHeaderBillTo(rec)
	if err != nil {
		return err
	}
	shipToAddress, err := formataddress.SalesHeaderShipTo(rec)
	if err != nil {
		return err
	}
	run.SetValue("bill_to_address", billToAddress)
	run.SetValue("ship_to_address", shipToAddress)

	salesPurchPerson := gtables.NewSalespersonPurchaser(rec)
	if rec.Salesperson_code == "" {
		salesPurchPerson.Init()
	} else {
		if !salesPurchPerson.Get(rec.Salesperson_code) {
			salesPurchPerson.Init()
		}
	}
	run.SetValue("salesperson", salesPurchPerson.Name.String())
	shipmentMethod := gtables.NewShipmentMethod(rec)
	if rec.Shipment_method_code == "" {
		shipmentMethod.Init()
	} else {
		if !shipmentMethod.Get(rec.Shipment_method_code) {
			shipmentMethod.Init()
		}
	}
	run.SetValue("ship_via", shipmentMethod.Description.String())
	paymentTerms := gtables.NewPaymentTerms(rec)
	if rec.Payment_terms_code == "" {
		paymentTerms.Init()
	} else {
		if !paymentTerms.Get(rec.Payment_terms_code) {
			paymentTerms.Init()
		}
	}
	run.SetValue("terms", paymentTerms.Description.String())

	totals, err := salespost.SumSalesLines(rec, salespost.General)
	if err != nil {
		return err
	}
	totalSalesLine := totals.TotalSalesLine
	run.SetValue("subtotal", totalSalesLine.Amount.Add(totalSalesLine.Inv_discount_amount))
	run.SetValue("invoice_discount", totalSalesLine.Inv_discount_amount.Neg())
	run.SetValue("tax_amount", totals.TaxAmount)
	run.SetValue("total", totalSalesLine.Amount_including_tax)
	if rec.Currency_code == "" {
		glSetup := gtables.GetGeneralLedgerSetup(rec)
		run.SetValue("currency", glSetup.Lcy_code.String())
	} else {
		run.SetValue("currency", rec.Currency_code.String())
	}

	noOfCopies := run.OptionInt("no_of_copies")
	if noOfCopies < 0 {
		noOfCopies = -noOfCopies
	}
	r.noOfLoops = 1 + noOfCopies
	r.printed++
	run.SetFileName(run.Text("FILE_NAME", rec.No))
	return nil
}

// salesHeaderOnPostDataItem: several documents go into one file named as the report
func (r *salesDocument) salesHeaderOnPostDataItem(rec *gtables.SalesHeader, run *Run) error {
	if r.printed > 1 {
		run.SetFileName("")
	}
	return nil
}

func (r *salesDocument) copyLoopOnPreDataItem(rec *Integer, run *Run) error {
	rec.SetRange("number", 1, r.noOfLoops)
	return nil
}

func (r *salesDocument) copyLoopOnAfterGetRecord(rec *Integer, run *Run) error {
	run.SetPageNo(1)
	if rec.Number > 1 {
		run.SetValue("copy_text", run.Text("COPY"))
	} else {
		run.SetValue("copy_text", "")
	}
	return nil
}

// copyLoopOnPostDataItem: the document is printed (not when only previewed)
func (r *salesDocument) copyLoopOnPostDataItem(rec *Integer, run *Run) error {
	if !run.Preview() {
		return salesprinted.Run(r.salesHeader)
	}
	return nil
}

func (r *salesDocument) salesLineOnPreDataItem(rec *gtables.SalesLine, run *Run) error {
	rec.SetRange("document_type", gtables.SalesLineDocument_type(r.salesHeader.Document_type))
	rec.SetRange("document_no", r.salesHeader.No)
	return nil
}

func (r *salesDocument) salesLineOnAfterGetRecord(rec *gtables.SalesLine, run *Run) error {
	run.SetValue("amount_excl_inv_disc", rec.Amount.Add(rec.Inv_discount_amount))
	run.SetValue("description_2", rec.Description_2.String())
	return nil
}
