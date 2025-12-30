package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	apitypes "github.com/hansjlachmann/openerp/src/api/types"
	"github.com/hansjlachmann/openerp/src/business-logic/tables"
	"github.com/hansjlachmann/openerp/src/foundation/filters"
	"github.com/hansjlachmann/openerp/src/foundation/i18n"
	"github.com/hansjlachmann/openerp/src/foundation/session"
	"github.com/hansjlachmann/openerp/src/foundation/types"
)

// TablesHandler handles table-related API requests
type TablesHandler struct {
	db *sql.DB
}

// NewTablesHandler creates a new tables handler
func NewTablesHandler(db *sql.DB) *TablesHandler {
	return &TablesHandler{db: db}
}

// GetRecordIDs returns only the IDs from a table (lightweight for navigation)
// GET /api/tables/:table/ids
func (h *TablesHandler) GetRecordIDs(c *fiber.Ctx) error {
	tableName := c.Params("table")
	sess := session.GetCurrent()

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("No active session"))
	}

	company := sess.GetCompany()

	// Parse query parameters
	sortBy := c.Query("sort_by", "")

	// Get IDs based on table name
	var ids []string
	var err error

	switch tableName {
	case "Customer":
		ids, err = h.getCustomerIDs(company, sortBy)
	case "Payment_terms":
		ids, err = h.getPaymentTermsIDs(company, sortBy)
	case "Customer_ledger_entry":
		ids, err = h.getCustomerLedgerEntryIDs(company, sortBy)
	default:
		return c.Status(404).JSON(apitypes.NewErrorResponse(fmt.Sprintf("Table '%s' not found", tableName)))
	}

	if err != nil {
		return c.Status(500).JSON(apitypes.NewErrorResponse(err.Error()))
	}

	response := apitypes.NewSuccessResponse(map[string]interface{}{
		"ids": ids,
	})
	return c.JSON(response)
}

// ListRecords returns a list of records from a table
// GET /api/tables/:table/list
func (h *TablesHandler) ListRecords(c *fiber.Ctx) error {
	tableName := c.Params("table")
	sess := session.GetCurrent()

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("No active session"))
	}

	company := sess.GetCompany()
	language := sess.GetLanguage()

	// Parse query parameters
	sortBy := c.Query("sort_by", "")
	sortOrder := c.Query("sort_order", "asc")

	// Parse fields parameter (JSON array of field names)
	var requestedFields []string
	fieldsParam := c.Query("fields", "")
	if fieldsParam != "" {
		if err := json.Unmarshal([]byte(fieldsParam), &requestedFields); err != nil {
			return c.Status(400).JSON(apitypes.NewErrorResponse("Invalid fields parameter"))
		}
	}

	// Parse filters parameter (JSON array of filter expressions)
	var requestedFilters []filters.FilterExpression
	filtersParam := c.Query("filters", "")
	if filtersParam != "" {
		var apiFilters []struct {
			Field      string `json:"field"`
			Expression string `json:"expression"`
		}
		if err := json.Unmarshal([]byte(filtersParam), &apiFilters); err != nil {
			return c.Status(400).JSON(apitypes.NewErrorResponse("Invalid filters parameter"))
		}

		// Convert API filters to filter expressions
		for _, f := range apiFilters {
			requestedFilters = append(requestedFilters, filters.FilterExpression{
				Field:      f.Field,
				Expression: f.Expression,
			})
		}
	}

	// Build query based on table name
	var records interface{}
	var err error

	switch tableName {
	case "Customer":
		records, err = h.listCustomers(company, sortBy, sortOrder, requestedFields, requestedFilters)
	case "Payment_terms":
		records, err = h.listPaymentTerms(company, sortBy, sortOrder)
	case "Customer_ledger_entry":
		records, err = h.listCustomerLedgerEntries(company, sortBy, sortOrder)
	default:
		return c.Status(404).JSON(apitypes.NewErrorResponse(fmt.Sprintf("Table '%s' not found", tableName)))
	}

	if err != nil {
		return c.Status(500).JSON(apitypes.NewErrorResponse(err.Error()))
	}

	// Get captions
	ts := i18n.GetInstance()
	captions := &apitypes.CaptionData{
		Table:  ts.TableCaption(tableName, language),
		Fields: make(map[string]string),
	}

	// Add field captions based on table
	h.addFieldCaptions(tableName, language, captions)

	// Return paginated response
	listResponse := &apitypes.ListResponse{
		Records:  records,
		Total:    getRecordCount(records),
		Page:     1,
		PageSize: getRecordCount(records),
	}

	response := apitypes.NewSuccessResponseWithCaptions(listResponse, captions)
	return c.JSON(response)
}

// GetRecord returns a single record by ID
// GET /api/tables/:table/card/:id
func (h *TablesHandler) GetRecord(c *fiber.Ctx) error {
	tableName := c.Params("table")
	id := c.Params("id")
	sess := session.GetCurrent()

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("No active session"))
	}

	company := sess.GetCompany()
	language := sess.GetLanguage()

	var record interface{}
	var err error

	switch tableName {
	case "Customer":
		var customer tables.Customer
		customer.Init(h.db, company)
		if !customer.Get(types.NewCode(id)) {
			return c.Status(404).JSON(apitypes.NewErrorResponse("Record not found"))
		}
		// Calculate FlowFields before converting to map
		customer.CalcFields("balance_lcy", "sales_lcy", "no_of_ledger_entries")
		record = customerToMap(&customer)

	case "Payment_terms":
		var pt tables.PaymentTerms
		pt.Init(h.db, company)
		if !pt.Get(types.NewCode(id)) {
			return c.Status(404).JSON(apitypes.NewErrorResponse("Record not found"))
		}
		record = paymentTermsToMap(&pt)

	default:
		return c.Status(404).JSON(apitypes.NewErrorResponse(fmt.Sprintf("Table '%s' not found", tableName)))
	}

	if err != nil {
		return c.Status(500).JSON(apitypes.NewErrorResponse(err.Error()))
	}

	// Get captions
	ts := i18n.GetInstance()
	captions := &apitypes.CaptionData{
		Table:  ts.TableCaption(tableName, language),
		Fields: make(map[string]string),
	}
	h.addFieldCaptions(tableName, language, captions)

	response := apitypes.NewSuccessResponseWithCaptions(record, captions)
	return c.JSON(response)
}

// InsertRecord inserts a new record
// POST /api/tables/:table/insert
func (h *TablesHandler) InsertRecord(c *fiber.Ctx) error {
	tableName := c.Params("table")
	sess := session.GetCurrent()

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("No active session"))
	}

	company := sess.GetCompany()

	// Parse request body
	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Invalid request body"))
	}

	switch tableName {
	case "Customer":
		customer := mapToCustomer(data)
		customer.Init(h.db, company)
		if !customer.Insert(true) {
			return c.Status(500).JSON(apitypes.NewErrorResponse("Failed to insert customer"))
		}
		response := apitypes.NewSuccessResponse(customerToMap(customer))
		return c.JSON(response)

	case "Payment_terms":
		pt := mapToPaymentTerms(data)
		pt.Init(h.db, company)
		if !pt.Insert(true) {
			return c.Status(500).JSON(apitypes.NewErrorResponse("Failed to insert payment terms"))
		}
		response := apitypes.NewSuccessResponse(paymentTermsToMap(pt))
		return c.JSON(response)

	default:
		return c.Status(404).JSON(apitypes.NewErrorResponse(fmt.Sprintf("Table '%s' not found", tableName)))
	}
}

// ModifyRecord updates an existing record
// PUT /api/tables/:table/modify/:id
func (h *TablesHandler) ModifyRecord(c *fiber.Ctx) error {
	tableName := c.Params("table")
	id := c.Params("id")
	sess := session.GetCurrent()

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("No active session"))
	}

	company := sess.GetCompany()

	// Parse request body
	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Invalid request body"))
	}

	switch tableName {
	case "Customer":
		var customer tables.Customer
		customer.Init(h.db, company)
		if !customer.Get(types.NewCode(id)) {
			return c.Status(404).JSON(apitypes.NewErrorResponse("Record not found"))
		}
		updateCustomerFromMap(&customer, data)
		if !customer.Modify(true) {
			return c.Status(500).JSON(apitypes.NewErrorResponse("Failed to modify customer"))
		}
		response := apitypes.NewSuccessResponse(customerToMap(&customer))
		return c.JSON(response)

	case "Payment_terms":
		var pt tables.PaymentTerms
		pt.Init(h.db, company)
		if !pt.Get(types.NewCode(id)) {
			return c.Status(404).JSON(apitypes.NewErrorResponse("Record not found"))
		}
		updatePaymentTermsFromMap(&pt, data)
		if !pt.Modify(true) {
			return c.Status(500).JSON(apitypes.NewErrorResponse("Failed to modify payment terms"))
		}
		response := apitypes.NewSuccessResponse(paymentTermsToMap(&pt))
		return c.JSON(response)

	default:
		return c.Status(404).JSON(apitypes.NewErrorResponse(fmt.Sprintf("Table '%s' not found", tableName)))
	}
}

// DeleteRecord deletes a record
// DELETE /api/tables/:table/delete/:id
func (h *TablesHandler) DeleteRecord(c *fiber.Ctx) error {
	tableName := c.Params("table")
	id := c.Params("id")
	sess := session.GetCurrent()

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("No active session"))
	}

	company := sess.GetCompany()

	switch tableName {
	case "Customer":
		var customer tables.Customer
		customer.Init(h.db, company)
		if !customer.Get(types.NewCode(id)) {
			return c.Status(404).JSON(apitypes.NewErrorResponse("Record not found"))
		}
		if !customer.Delete(true) {
			return c.Status(500).JSON(apitypes.NewErrorResponse("Failed to delete customer"))
		}
		response := apitypes.NewSuccessResponse(nil)
		return c.JSON(response)

	case "Payment_terms":
		var pt tables.PaymentTerms
		pt.Init(h.db, company)
		if !pt.Get(types.NewCode(id)) {
			return c.Status(404).JSON(apitypes.NewErrorResponse("Record not found"))
		}
		if !pt.Delete(true) {
			return c.Status(500).JSON(apitypes.NewErrorResponse("Failed to delete payment terms"))
		}
		response := apitypes.NewSuccessResponse(nil)
		return c.JSON(response)

	default:
		return c.Status(404).JSON(apitypes.NewErrorResponse(fmt.Sprintf("Table '%s' not found", tableName)))
	}
}

// ValidateField validates a field value
// POST /api/tables/:table/validate
func (h *TablesHandler) ValidateField(c *fiber.Ctx) error {
	tableName := c.Params("table")
	sess := session.GetCurrent()

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("No active session"))
	}

	company := sess.GetCompany()

	var req apitypes.ValidationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse("Invalid request body"))
	}

	// Validate field based on table
	switch tableName {
	case "Customer":
		var customer tables.Customer
		customer.Init(h.db, company)
		if err := customer.ValidateField(req.Field, req.Value); err != nil {
			return c.JSON(apitypes.NewErrorResponse(err.Error()))
		}
		return c.JSON(apitypes.NewSuccessResponse(map[string]interface{}{
			"valid": true,
		}))

	default:
		return c.Status(404).JSON(apitypes.NewErrorResponse(fmt.Sprintf("Table '%s' not found", tableName)))
	}
}

// Helper functions

// filterFlowFields returns only the FlowFields that are in the requested fields list
func filterFlowFields(requestedFields []string, availableFlowFields []string) []string {
	var result []string
	requestedMap := make(map[string]bool)

	for _, field := range requestedFields {
		requestedMap[field] = true
	}

	for _, flowField := range availableFlowFields {
		if requestedMap[flowField] {
			result = append(result, flowField)
		}
	}

	return result
}

func (h *TablesHandler) listCustomers(company, sortBy, sortOrder string, requestedFields []string, requestedFilters []filters.FilterExpression) ([]map[string]interface{}, error) {
	var customer tables.Customer
	customer.Init(h.db, company)

	// Apply filters using SetFilter
	for _, filter := range requestedFilters {
		customer.SetFilter(filter.Field, filter.Expression)
	}

	// Apply sorting
	if sortBy != "" {
		customer.SetCurrentKey(sortBy)
	}

	var customers []map[string]interface{}

	if customer.FindSet() {
		for {
			// Only calculate FlowFields that are requested
			if len(requestedFields) > 0 {
				flowFieldsToCalc := filterFlowFields(requestedFields, []string{"balance_lcy", "sales_lcy", "no_of_ledger_entries"})
				if len(flowFieldsToCalc) > 0 {
					customer.CalcFields(flowFieldsToCalc...)
				}
			} else {
				// If no fields specified, calculate all FlowFields (backward compatibility)
				customer.CalcFields("balance_lcy", "sales_lcy", "no_of_ledger_entries")
			}

			customers = append(customers, customerToMap(&customer))
			if !customer.Next() {
				break
			}
		}
	}

	return customers, nil
}

func (h *TablesHandler) listPaymentTerms(company, sortBy, sortOrder string) ([]map[string]interface{}, error) {
	var pt tables.PaymentTerms
	pt.Init(h.db, company)

	if sortBy != "" {
		pt.SetCurrentKey(sortBy)
	}

	var paymentTerms []map[string]interface{}

	if pt.FindSet() {
		for {
			paymentTerms = append(paymentTerms, paymentTermsToMap(&pt))
			if !pt.Next() {
				break
			}
		}
	}

	return paymentTerms, nil
}

func (h *TablesHandler) listCustomerLedgerEntries(company, sortBy, sortOrder string) ([]map[string]interface{}, error) {
	var cle tables.CustomerLedgerEntry
	cle.Init(h.db, company)

	if sortBy != "" {
		cle.SetCurrentKey(sortBy)
	}

	var entries []map[string]interface{}

	if cle.FindSet() {
		for {
			entries = append(entries, customerLedgerEntryToMap(&cle))
			if !cle.Next() {
				break
			}
		}
	}

	return entries, nil
}

// ID-only helper functions for navigation

func (h *TablesHandler) getCustomerIDs(company, sortBy string) ([]string, error) {
	var customer tables.Customer
	customer.Init(h.db, company)

	if sortBy != "" {
		customer.SetCurrentKey(sortBy)
	}

	var ids []string

	if customer.FindSet() {
		for {
			ids = append(ids, customer.No.String())
			if !customer.Next() {
				break
			}
		}
	}

	return ids, nil
}

func (h *TablesHandler) getPaymentTermsIDs(company, sortBy string) ([]string, error) {
	var pt tables.PaymentTerms
	pt.Init(h.db, company)

	if sortBy != "" {
		pt.SetCurrentKey(sortBy)
	}

	var ids []string

	if pt.FindSet() {
		for {
			ids = append(ids, pt.Code.String())
			if !pt.Next() {
				break
			}
		}
	}

	return ids, nil
}

func (h *TablesHandler) getCustomerLedgerEntryIDs(company, sortBy string) ([]string, error) {
	var cle tables.CustomerLedgerEntry
	cle.Init(h.db, company)

	if sortBy != "" {
		cle.SetCurrentKey(sortBy)
	}

	var ids []string

	if cle.FindSet() {
		for {
			ids = append(ids, fmt.Sprintf("%d", cle.Entry_no))
			if !cle.Next() {
				break
			}
		}
	}

	return ids, nil
}

func (h *TablesHandler) addFieldCaptions(tableName, language string, captions *apitypes.CaptionData) {
	ts := i18n.GetInstance()

	switch tableName {
	case "Customer":
		fields := []string{"no", "name", "address", "post_code", "city", "phone_number", "email",
			"payment_terms_code", "credit_limit", "balance_lcy", "sales_lcy", "no_of_ledger_entries",
			"last_order_date", "created_at", "status"}
		for _, field := range fields {
			captions.Fields[field] = ts.FieldCaption(tableName, field, language)
		}

	case "Payment_terms":
		fields := []string{"code", "description", "due_date_calculation", "discount_date_calculation", "discount_percent"}
		for _, field := range fields {
			captions.Fields[field] = ts.FieldCaption(tableName, field, language)
		}

	case "Customer_ledger_entry":
		fields := []string{"entry_no", "customer_no", "posting_date", "document_type", "document_no",
			"description", "amount", "remaining_amount"}
		for _, field := range fields {
			captions.Fields[field] = ts.FieldCaption(tableName, field, language)
		}
	}
}

// Conversion functions: Table struct <-> map[string]interface{}

func customerToMap(c *tables.Customer) map[string]interface{} {
	return map[string]interface{}{
		"no":                    c.No.String(),
		"name":                  c.Name.String(),
		"address":               c.Address.String(),
		"post_code":             c.Post_code.String(),
		"city":                  c.City.String(),
		"phone_number":          c.Phonenumber.String(),
		"payment_terms_code":    c.Payment_terms_code.String(),
		"credit_limit":          c.Credit_limit.String(),
		"balance_lcy":           c.Balance_lcy.String(),
		"sales_lcy":             c.Sales_lcy.String(),
		"no_of_ledger_entries":  c.No_of_ledger_entries,
		"last_order_date":       c.Last_order_date.String(),
		"created_at":            c.Created_at.String(),
		"status":                int(c.Status),
	}
}

func mapToCustomer(data map[string]interface{}) *tables.Customer {
	customer := &tables.Customer{}

	if v, ok := data["no"].(string); ok {
		customer.No = types.NewCode(v)
	}
	if v, ok := data["name"].(string); ok {
		customer.Name = types.NewText(v)
	}
	if v, ok := data["address"].(string); ok {
		customer.Address = types.NewText(v)
	}
	if v, ok := data["post_code"].(string); ok {
		customer.Post_code = types.NewCode(v)
	}
	if v, ok := data["city"].(string); ok {
		customer.City = types.NewText(v)
	}
	if v, ok := data["phone_number"].(string); ok {
		customer.Phonenumber = types.NewText(v)
	}
	if v, ok := data["payment_terms_code"].(string); ok {
		customer.Payment_terms_code = types.NewCode(v)
	}
	if v, ok := data["status"].(float64); ok {
		customer.Status = tables.CustomerStatus(int(v))
	}

	return customer
}

func updateCustomerFromMap(customer *tables.Customer, data map[string]interface{}) {
	if v, ok := data["name"].(string); ok {
		customer.Name = types.NewText(v)
	}
	if v, ok := data["address"].(string); ok {
		customer.Address = types.NewText(v)
	}
	if v, ok := data["post_code"].(string); ok {
		customer.Post_code = types.NewCode(v)
	}
	if v, ok := data["city"].(string); ok {
		customer.City = types.NewText(v)
	}
	if v, ok := data["phone_number"].(string); ok {
		customer.Phonenumber = types.NewText(v)
	}
	if v, ok := data["payment_terms_code"].(string); ok {
		customer.Payment_terms_code = types.NewCode(v)
	}
	if v, ok := data["status"].(float64); ok {
		customer.Status = tables.CustomerStatus(int(v))
	}
}

func paymentTermsToMap(pt *tables.PaymentTerms) map[string]interface{} {
	return map[string]interface{}{
		"code":        pt.Code.String(),
		"description": pt.Description.String(),
		"active":      pt.Active,
	}
}

func mapToPaymentTerms(data map[string]interface{}) *tables.PaymentTerms {
	pt := &tables.PaymentTerms{}

	if v, ok := data["code"].(string); ok {
		pt.Code = types.NewCode(v)
	}
	if v, ok := data["description"].(string); ok {
		pt.Description = types.NewText(v)
	}
	if v, ok := data["active"].(bool); ok {
		pt.Active = v
	}

	return pt
}

func updatePaymentTermsFromMap(pt *tables.PaymentTerms, data map[string]interface{}) {
	if v, ok := data["description"].(string); ok {
		pt.Description = types.NewText(v)
	}
	if v, ok := data["active"].(bool); ok {
		pt.Active = v
	}
}

func customerLedgerEntryToMap(cle *tables.CustomerLedgerEntry) map[string]interface{} {
	return map[string]interface{}{
		"entry_no":         cle.Entry_no,
		"customer_no":      cle.Customer_no.String(),
		"posting_date":     cle.Posting_date.String(),
		"document_type":    int(cle.Document_type),
		"document_no":      cle.Document_no.String(),
		"description":      cle.Description.String(),
		"amount":           cle.Amount.String(),
		"remaining_amount": cle.Remaining_amount.String(),
	}
}

func getRecordCount(records interface{}) int {
	switch v := records.(type) {
	case []map[string]interface{}:
		return len(v)
	default:
		return 0
	}
}

// normalizeTableName converts "Customer" to "customer", "Payment Terms" to "payment_terms"
func normalizeTableName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "_"))
}
