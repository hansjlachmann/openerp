package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
	apitypes "github.com/hansjlachmann/openerp/backend/api/types"
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/i18n"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
)

// CompanyInitializer can create company-scoped tables for a newly inserted company.
type CompanyInitializer interface {
	InitializeCompanyTablesWithDBType(db *sql.DB, companyName string, dbType database.DBType) error
}

// TablesHandler handles table-related API requests
type TablesHandler struct {
	db          *sql.DB
	dbType      database.DBType
	companyInit CompanyInitializer
}

// NewTablesHandler creates a new tables handler (defaults to SQLite)
func NewTablesHandler(db *sql.DB) *TablesHandler {
	return NewTablesHandlerWithDBType(db, database.DBTypeSQLite)
}

// NewTablesHandlerWithDBType creates a new tables handler with explicit database type
func NewTablesHandlerWithDBType(db *sql.DB, dbType database.DBType) *TablesHandler {
	return &TablesHandler{db: db, dbType: dbType}
}

// NewTablesHandlerFull creates a tables handler with company initializer support
func NewTablesHandlerFull(db *sql.DB, dbType database.DBType, companyInit CompanyInitializer) *TablesHandler {
	return &TablesHandler{db: db, dbType: dbType, companyInit: companyInit}
}

// getTable creates a table instance by name using the registry
func (h *TablesHandler) getTable(tableName, company string) (ftables.Table, error) {
	factory, ok := tables.GetTableFactory(tableName)
	if !ok {
		return nil, apperrors.TableNotFound(tableName)
	}
	table := factory()
	table.InitWithDBType(h.db, company, h.dbType)
	return table, nil
}

// ensureSetupRecord makes sure a setup table's single blank-primary-key record
// exists, creating it if absent (BC-style: the setup record is always there).
func (h *TablesHandler) ensureSetupRecord(tableName, company string) {
	probe, err := h.getTable(tableName, company)
	if err != nil || !probe.IsSetupTable() {
		return
	}
	if probe.Get("") {
		return // already exists
	}
	// Insert the single blank-primary-key record (fresh instance, no triggers).
	blank, err := h.getTable(tableName, company)
	if err != nil {
		return
	}
	_ = blank.Insert(false)
}

// parseRecordKey parses a URL record ID into the appropriate type for table.Get()
// For single PK tables: returns the string as-is
// For composite PK tables: splits comma-separated values and returns map[string]interface{}
// fullPrimaryKey returns the current record's complete primary key in the form Get expects
// (a string for a single-field key, a field->value map for a composite key — the same form
// parseRecordKey builds from a URL id), plus a comma-joined display value. GetPrimaryKeyValue
// only returns the first key field, so it cannot identify a composite-key record.
func fullPrimaryKey(table ftables.Table) (interface{}, string) {
	values := table.ToMap()
	var pkFields []string
	for _, f := range table.GetFields() {
		if f.PrimaryKey {
			pkFields = append(pkFields, f.Name)
		}
	}
	if len(pkFields) <= 1 {
		return table.GetPrimaryKeyValue(), table.GetPrimaryKeyValue()
	}
	pkMap := make(map[string]interface{}, len(pkFields))
	parts := make([]string, len(pkFields))
	for i, field := range pkFields {
		parts[i] = fmt.Sprint(values[field])
		pkMap[field] = parts[i]
	}
	return pkMap, strings.Join(parts, ",")
}

func parseRecordKey(id string, table ftables.Table) interface{} {
	// URL-decode the id (Fiber does not auto-decode route params)
	if decoded, err := url.PathUnescape(id); err == nil {
		id = decoded
	}

	fields := table.GetFields()
	var pkFields []string
	for _, f := range fields {
		if f.PrimaryKey {
			pkFields = append(pkFields, f.Name)
		}
	}

	if len(pkFields) <= 1 {
		return id
	}

	// Composite PK: split by comma and build map
	parts := strings.Split(id, ",")
	if len(parts) != len(pkFields) {
		return id // Mismatch — fall back to string
	}

	pkMap := make(map[string]interface{})
	for i, field := range pkFields {
		pkMap[field] = parts[i]
	}
	return pkMap
}

// LookupData represents structured lookup data for a field
type LookupData struct {
	Columns       []LookupColumn           `json:"columns,omitempty"`        // Column definitions (advanced mode)
	Rows          []map[string]interface{} `json:"rows"`                     // Row data
	Simple        map[string]string        `json:"simple,omitempty"`         // Simple key->display map (simple mode)
	SearchTimeout int                      `json:"search_timeout,omitempty"` // Auto-clear search timeout in ms (0 = default 1500)
}

// LookupColumn represents a column in the lookup dropdown
type LookupColumn struct {
	Source string `json:"source"`
	Width  int    `json:"width"`
}

// getLookupValues fetches lookup values for table relation fields
func (h *TablesHandler) getLookupValues(table ftables.Table, company string) map[string]*LookupData {
	lookups := make(map[string]*LookupData)

	for fieldName, relInfo := range table.GetTableRelationFields() {
		// Get the related table
		relFactory, ok := tables.GetTableFactory(relInfo.Table)
		if !ok {
			continue // Skip if related table not found
		}

		relTable := relFactory()
		relTable.InitWithDBType(h.db, company, h.dbType)

		lookup := &LookupData{
			Rows:          []map[string]interface{}{},
			Simple:        make(map[string]string),
			SearchTimeout: relInfo.SearchTimeout,
		}

		// Add column definitions if lookup_columns is specified (advanced mode)
		if len(relInfo.LookupColumns) > 0 {
			for _, col := range relInfo.LookupColumns {
				lookup.Columns = append(lookup.Columns, LookupColumn{
					Source: col.Source,
					Width:  col.Width,
				})
			}
		}

		// Fetch all records from related table
		if relTable.FindSet() {
			for {
				keyValue := relTable.GetPrimaryKeyValue()
				recordMap := relTable.ToMap()

				// For advanced mode: add full row data
				if len(relInfo.LookupColumns) > 0 {
					row := make(map[string]interface{})
					row["_key"] = keyValue // Always include the key
					for _, col := range relInfo.LookupColumns {
						if val, ok := recordMap[col.Source]; ok {
							row[col.Source] = val
						}
					}
					lookup.Rows = append(lookup.Rows, row)
				}

				// For simple mode: add key->display mapping
				displayValue := keyValue
				if relInfo.DisplayField != "" {
					if dv, ok := recordMap[relInfo.DisplayField]; ok && dv != nil {
						displayValue = fmt.Sprintf("%v", dv)
					}
				}
				lookup.Simple[keyValue] = displayValue

				if !relTable.Next() {
					break
				}
			}
		}

		lookups[fieldName] = lookup
	}

	return lookups
}

// getLookupValuesAsInterface converts lookup data to interface{} map for JSON serialization
func (h *TablesHandler) getLookupValuesAsInterface(table ftables.Table, company string) map[string]interface{} {
	lookups := h.getLookupValues(table, company)
	result := make(map[string]interface{})
	for k, v := range lookups {
		result[k] = v
	}
	return result
}

// GetOptions returns only the option field metadata (no records)
// GET /api/tables/:table/options
func (h *TablesHandler) GetOptions(c *fiber.Ctx) error {
	tableName := c.Params("table")
	sess := getSession(c)

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	company := sess.GetCompany()
	language := sess.GetLanguage()

	// Create table instance (doesn't fetch any data)
	table, err := h.getTable(tableName, company)
	if err != nil {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.TableNotFound(tableName).Message(language)))
	}

	// Build options map
	options := make(map[string]map[string]string)
	for fieldName, optionValues := range table.GetOptionFields() {
		optionMap := make(map[string]string)
		for i, opt := range optionValues {
			optionMap[fmt.Sprintf("%d", i)] = opt
		}
		options[fieldName] = optionMap
	}

	// Build lookups map for table relation fields
	lookups := h.getLookupValues(table, company)

	response := apitypes.NewSuccessResponse(map[string]interface{}{
		"options": options,
		"lookups": lookups,
	})
	return c.JSON(response)
}

// GetRecordIDs returns only the IDs from a table (lightweight for navigation)
// GET /api/tables/:table/ids
func (h *TablesHandler) GetRecordIDs(c *fiber.Ctx) error {
	tableName := c.Params("table")
	sess := getSession(c)

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	company := sess.GetCompany()
	language := sess.GetLanguage()

	// Create table instance
	table, err := h.getTable(tableName, company)
	if err != nil {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.TableNotFound(tableName).Message(language)))
	}

	// Parse query parameters
	sortBy := c.Query("sort_by", "")
	if sortBy != "" {
		// Field names end up in SQL text: only accept real columns of this table
		if !table.HasColumn(sortBy) {
			return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidSortField().Message(language)))
		}
		table.SetCurrentKey(sortBy)
	}

	// Collect IDs
	var ids []string
	if table.FindSet() {
		ids = append(ids, table.GetPrimaryKeyValue())
		for table.Next() {
			ids = append(ids, table.GetPrimaryKeyValue())
		}
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
	sess := getSession(c)

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	company := sess.GetCompany()
	language := sess.GetLanguage()

	// Create table instance
	table, err := h.getTable(tableName, company)
	if err != nil {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.TableNotFound(tableName).Message(language)))
	}

	// Parse query parameters
	sortBy := c.Query("sort_by", "")
	if sortBy != "" {
		// Field names end up in SQL text: only accept real columns of this table
		if !table.HasColumn(sortBy) {
			return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidSortField().Message(language)))
		}
		table.SetCurrentKey(sortBy)
	}

	// Setup tables always have their single record present (BC-style)
	if table.IsSetupTable() {
		h.ensureSetupRecord(tableName, company)
	}

	// Parse fields parameter (JSON array of field names) - for future FlowField optimization
	var requestedFields []string
	fieldsParam := c.Query("fields", "")
	if fieldsParam != "" {
		if err := json.Unmarshal([]byte(fieldsParam), &requestedFields); err != nil {
			return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidFields().Message(language)))
		}
	}

	// Parse filters parameter (JSON array of filter expressions)
	filtersParam := c.Query("filters", "")
	if filtersParam != "" {
		var apiFilters []struct {
			Field      string `json:"field"`
			Expression string `json:"expression"`
		}
		if err := json.Unmarshal([]byte(filtersParam), &apiFilters); err != nil {
			return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidFilters().Message(language)))
		}

		// Apply BC-style filters (field names end up in SQL text: only accept real columns)
		for _, f := range apiFilters {
			if !table.HasColumn(f.Field) {
				return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidFilters().Message(language)))
			}
			table.SetFilter(f.Field, f.Expression)
		}
	}

	// Parse pagination window (opt-in). When page_size > 0 the query is limited
	// server-side and total reflects the full filtered count. Without page_size the
	// entire result set is returned, preserving the frontend's client-side
	// search/sort over the full data set.
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	pageSize := c.QueryInt("page_size", 0)
	if pageSize < 0 {
		pageSize = 0
	}
	const maxPageSize = 1000
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	total := 0
	if pageSize > 0 {
		total = table.Count() // full filtered count, before applying the window
		table.SetPage(pageSize, (page-1)*pageSize)
	}

	// Collect records (non-nil so an empty result serializes as [] rather than null,
	// e.g. when a requested page is past the end of the data)
	records := make([]map[string]interface{}, 0)
	flowFields := table.GetFlowFields()

	if table.FindSet() {
		// Calculate FlowFields if not explicitly excluded
		if len(requestedFields) == 0 || containsAny(requestedFields, flowFields) {
			table.CalcFields(flowFields...)
		}
		records = append(records, table.ToMap())

		for table.Next() {
			if len(requestedFields) == 0 || containsAny(requestedFields, flowFields) {
				table.CalcFields(flowFields...)
			}
			records = append(records, table.ToMap())
		}
	}

	// Get captions
	ts := i18n.GetInstance()
	captions := &apitypes.CaptionData{
		Table:      ts.TableCaption(tableName, language),
		Fields:     make(map[string]string),
		FieldTypes: make(map[string]string),
		Options:    make(map[string]map[string]string),
		Lookups:    h.getLookupValuesAsInterface(table, company),
	}

	// Add field captions and types from metadata
	for _, field := range table.GetFields() {
		captions.Fields[field.Name] = ts.FieldCaption(tableName, field.Name, language)
		captions.FieldTypes[field.Name] = string(field.Type)
	}

	// Add option field values
	for fieldName, options := range table.GetOptionFields() {
		optionMap := make(map[string]string)
		for i, opt := range options {
			optionMap[fmt.Sprintf("%d", i)] = opt
		}
		captions.Options[fieldName] = optionMap
	}

	// Pagination metadata: real values when a page_size was requested, otherwise
	// the legacy single-page shape (all records reported as page 1).
	respPage, respPageSize, respTotal := 1, len(records), len(records)
	if pageSize > 0 {
		respPage, respPageSize, respTotal = page, pageSize, total
	}

	response := apitypes.NewSuccessResponseWithCaptions(map[string]interface{}{
		"records":   records,
		"total":     respTotal,
		"page":      respPage,
		"page_size": respPageSize,
	}, captions)
	return c.JSON(response)
}

// GetRecord returns a single record by ID
// GET /api/tables/:table/card/:id
func (h *TablesHandler) GetRecord(c *fiber.Ctx) error {
	tableName := c.Params("table")
	id := c.Params("id")
	sess := getSession(c)

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	company := sess.GetCompany()
	language := sess.GetLanguage()
	tableCaption := i18n.GetInstance().TableCaption(tableName, language)

	// Create table instance
	table, err := h.getTable(tableName, company)
	if err != nil {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.TableNotFound(tableName).Message(language)))
	}

	// Get record by primary key (supports composite keys via comma-separated values)
	if !table.Get(parseRecordKey(id, table)) {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.RecordNotFound(tableCaption, id).Message(language)))
	}

	// Calculate FlowFields
	table.CalcFields(table.GetFlowFields()...)

	// Get captions
	ts := i18n.GetInstance()
	captions := &apitypes.CaptionData{
		Table:      ts.TableCaption(tableName, language),
		Fields:     make(map[string]string),
		FieldTypes: make(map[string]string),
		Options:    make(map[string]map[string]string),
		Lookups:    h.getLookupValuesAsInterface(table, company),
	}

	for _, field := range table.GetFields() {
		captions.Fields[field.Name] = ts.FieldCaption(tableName, field.Name, language)
		captions.FieldTypes[field.Name] = string(field.Type)
	}

	// Add option field values
	for fieldName, options := range table.GetOptionFields() {
		optionMap := make(map[string]string)
		for i, opt := range options {
			optionMap[fmt.Sprintf("%d", i)] = opt
		}
		captions.Options[fieldName] = optionMap
	}

	response := apitypes.NewSuccessResponseWithCaptions(table.ToMap(), captions)
	return c.JSON(response)
}

// InsertRecord inserts a new record
// POST /api/tables/:table/insert
func (h *TablesHandler) InsertRecord(c *fiber.Ctx) error {
	tableName := c.Params("table")
	sess := getSession(c)

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	company := sess.GetCompany()
	language := sess.GetLanguage()
	tableCaption := i18n.GetInstance().TableCaption(tableName, language)

	// Create table instance
	table, err := h.getTable(tableName, company)
	if err != nil {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.TableNotFound(tableName).Message(language)))
	}

	// Parse request body
	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidRequestBody().Message(language)))
	}

	// Validate and set each changed field (runs OnValidate triggers for table relations, etc.)
	if err := validateChangedFields(table, tableName, data); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(err.Error()))
	}

	// Special handling for User table password
	if tableName == "User" {
		if password, ok := data["password"].(string); ok && password != "" {
			if userTable, ok := table.(*tables.User); ok {
				if err := userTable.SetPassword(password); err != nil {
					return c.Status(400).JSON(apitypes.NewErrorResponse(err.Error()))
				}
			}
		}
	}

	// Check for empty primary key
	pkValue := table.GetPrimaryKeyValue()
	if pkValue == "" {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.EmptyPrimaryKey(tableCaption).Message(language)))
	}

	// Check if record already exists (all primary key fields, for composite keys too)
	pkLookup, pkDisplay := fullPrimaryKey(table)
	existingTable, _ := h.getTable(tableName, company)
	if existingTable.Get(pkLookup) {
		return c.Status(409).JSON(apitypes.NewErrorResponse(apperrors.DuplicateRecord(tableCaption, pkDisplay).Message(language)))
	}

	// Insert record
	if !table.Insert(true) {
		// A failed OnInsert trigger is a business rule the user broke: show its message
		if trigErr := table.TriggerError(); trigErr != nil {
			return c.Status(400).JSON(apitypes.NewErrorResponse(trigErr.Error()))
		}
		return c.Status(500).JSON(apitypes.NewErrorResponse(apperrors.InsertFailed(tableCaption).Message(language)))
	}

	// Initialize company-scoped tables when a new Company is created
	if tableName == "Company" && h.companyInit != nil {
		companyName := table.GetPrimaryKeyValue()
		if err := h.companyInit.InitializeCompanyTablesWithDBType(h.db, companyName, h.dbType); err != nil {
			fmt.Printf("Warning: Failed to initialize tables for company '%s': %v\n", companyName, err)
		}
	}

	response := apitypes.NewSuccessResponse(table.ToMap())
	return c.JSON(response)
}

// ModifyRecord updates an existing record
// PUT /api/tables/:table/modify/:id
func (h *TablesHandler) ModifyRecord(c *fiber.Ctx) error {
	tableName := c.Params("table")
	id := c.Params("id")
	sess := getSession(c)

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	company := sess.GetCompany()
	language := sess.GetLanguage()
	tableCaption := i18n.GetInstance().TableCaption(tableName, language)

	// Create table instance
	table, err := h.getTable(tableName, company)
	if err != nil {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.TableNotFound(tableName).Message(language)))
	}

	// An empty ID is only valid for a setup table (its single record has a blank primary key)
	if id == "" && !table.IsSetupTable() {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.EmptyPrimaryKey(tableCaption).Message(language)))
	}

	// Get existing record (supports composite keys via comma-separated values)
	if !table.Get(parseRecordKey(id, table)) {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.RecordNotFound(tableCaption, id).Message(language)))
	}

	// Parse request body
	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidRequestBody().Message(language)))
	}

	// Validate and set each changed field (runs OnValidate triggers for table relations, etc.)
	if err := validateChangedFields(table, tableName, data); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(err.Error()))
	}

	// Special handling for User table password
	if tableName == "User" {
		if password, ok := data["password"].(string); ok && password != "" {
			if userTable, ok := table.(*tables.User); ok {
				if err := userTable.SetPassword(password); err != nil {
					return c.Status(400).JSON(apitypes.NewErrorResponse(err.Error()))
				}
			}
		}
	}

	// Modify record
	if !table.Modify(true) {
		// A failed OnModify trigger is a business rule the user broke: show its message
		if trigErr := table.TriggerError(); trigErr != nil {
			return c.Status(400).JSON(apitypes.NewErrorResponse(trigErr.Error()))
		}
		return c.Status(500).JSON(apitypes.NewErrorResponse(apperrors.ModifyFailed(tableCaption, id).Message(language)))
	}

	// Calculate FlowFields for response
	table.CalcFields(table.GetFlowFields()...)

	response := apitypes.NewSuccessResponse(table.ToMap())
	return c.JSON(response)
}

// DeleteRecord deletes a record
// DELETE /api/tables/:table/delete/:id
func (h *TablesHandler) DeleteRecord(c *fiber.Ctx) error {
	tableName := c.Params("table")
	id := c.Params("id")
	sess := getSession(c)

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	company := sess.GetCompany()
	language := sess.GetLanguage()
	tableCaption := i18n.GetInstance().TableCaption(tableName, language)

	// Create table instance
	table, err := h.getTable(tableName, company)
	if err != nil {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.TableNotFound(tableName).Message(language)))
	}

	// An empty ID is only valid for a setup table (its single record has a blank primary key)
	if id == "" && !table.IsSetupTable() {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.EmptyPrimaryKey(tableCaption).Message(language)))
	}

	// Get existing record (supports composite keys via comma-separated values)
	if !table.Get(parseRecordKey(id, table)) {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.RecordNotFound(tableCaption, id).Message(language)))
	}

	// Capture company name before delete for table cleanup
	var deletedCompanyName string
	if tableName == "Company" {
		deletedCompanyName = table.GetPrimaryKeyValue()
	}

	// Delete record
	if !table.Delete(true) {
		// A failed OnDelete trigger is a business rule the user broke: show its message
		if trigErr := table.TriggerError(); trigErr != nil {
			return c.Status(400).JSON(apitypes.NewErrorResponse(trigErr.Error()))
		}
		return c.Status(500).JSON(apitypes.NewErrorResponse(apperrors.DeleteFailed(tableCaption, id).Message(language)))
	}

	// Drop company-scoped tables after a Company is deleted
	if deletedCompanyName != "" && h.dbType == database.DBTypePostgres {
		h.dropCompanyTables(deletedCompanyName)
	} else if deletedCompanyName != "" {
		h.dropCompanyTablesSQLite(deletedCompanyName)
	}

	response := apitypes.NewSuccessResponse(nil)
	return c.JSON(response)
}

// validateChangedFields runs ValidateField (BC/NAV VALIDATE) for each field in data whose
// value differs from the table's current value, in table field order. Unchanged fields are
// skipped so a stale payload value cannot overwrite a sibling that another field's
// OnValidate trigger filled in. Unknown fields are still validated so they are reported.
func validateChangedFields(table ftables.Table, tableName string, data map[string]interface{}) error {
	current := table.ToMap()

	skip := make(map[string]bool)
	for _, ff := range table.GetFlowFields() {
		skip[ff] = true // FlowFields are calculated, not validated
	}
	if tableName == "User" {
		skip["password"] = true // virtual field, handled separately by the caller
	}

	ordered := make([]string, 0, len(data))
	seen := make(map[string]bool, len(data))
	for _, f := range table.GetFields() {
		if _, ok := data[f.Name]; ok {
			ordered = append(ordered, f.Name)
			seen[f.Name] = true
		}
	}
	extra := make([]string, 0)
	for name := range data {
		if !seen[name] {
			extra = append(extra, name)
		}
	}
	sort.Strings(extra)
	ordered = append(ordered, extra...)

	for _, fieldName := range ordered {
		value := data[fieldName]
		// Skip nil values - frontend may send null for unchanged fields
		if value == nil || skip[fieldName] {
			continue
		}
		if cur, ok := current[fieldName]; ok && fmt.Sprint(cur) == fmt.Sprint(value) {
			continue
		}
		if err := table.ValidateField(fieldName, value); err != nil {
			return err
		}
	}
	return nil
}

// InitRecord returns a new, not yet inserted record with its defaults applied
// (BC/NAV OnNewRecord). Nothing is persisted.
// POST /api/tables/:table/init
func (h *TablesHandler) InitRecord(c *fiber.Ctx) error {
	tableName := c.Params("table")
	sess := getSession(c)

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	company := sess.GetCompany()
	language := sess.GetLanguage()

	table, err := h.getTable(tableName, company)
	if err != nil {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.TableNotFound(tableName).Message(language)))
	}

	table.InitRecord()

	return c.JSON(apitypes.NewSuccessResponse(table.ToMap()))
}

// ValidateField validates a single field value (BC/NAV VALIDATE).
// When the in-progress record is supplied, the table is hydrated from it first so the
// field's OnValidate trigger can see (and fill in) sibling fields; the resulting record
// is returned in data.
// POST /api/tables/:table/validate
func (h *TablesHandler) ValidateField(c *fiber.Ctx) error {
	tableName := c.Params("table")
	sess := getSession(c)

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	company := sess.GetCompany()
	language := sess.GetLanguage()

	// Parse request body
	var req struct {
		Field  string                 `json:"field"`
		Value  interface{}            `json:"value"`
		Record map[string]interface{} `json:"record"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidRequestBody().Message(language)))
	}

	// Create table instance
	table, err := h.getTable(tableName, company)
	if err != nil {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.TableNotFound(tableName).Message(language)))
	}

	// Hydrate from the in-progress record (plain assignment, no triggers)
	if req.Record != nil {
		table.FromMap(req.Record)
	}

	// Validate field (runs OnValidate trigger)
	if err := table.ValidateField(req.Field, req.Value); err != nil {
		return c.JSON(apitypes.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	// Check table relation: verify the value exists in the related table
	if valueStr, ok := req.Value.(string); ok && valueStr != "" {
		relFields := table.GetTableRelationFields()
		if relInfo, hasRelation := relFields[req.Field]; hasRelation {
			relTable, relErr := h.getTable(relInfo.Table, company)
			if relErr == nil {
				if !relTable.Get(valueStr) {
					relCaption := i18n.GetInstance().TableCaption(relInfo.Table, language)
					return c.JSON(apitypes.APIResponse{
						Success: false,
						Error:   fmt.Sprintf("%s '%s' does not exist", relCaption, valueStr),
					})
				}
			}
		}
	}

	return c.JSON(apitypes.NewSuccessResponse(table.ToMap()))
}

// dropCompanyTables drops all "companyName$*" tables from PostgreSQL
func (h *TablesHandler) dropCompanyTables(companyName string) {
	rows, err := h.db.Query(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name LIKE $1
	`, companyName+"$%")
	if err != nil {
		fmt.Printf("Warning: Failed to find company tables for '%s': %v\n", companyName, err)
		return
	}
	defer rows.Close()

	var tablesToDrop []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			tablesToDrop = append(tablesToDrop, name)
		}
	}

	for _, name := range tablesToDrop {
		if _, err := h.db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS "%s"`, name)); err != nil {
			fmt.Printf("Warning: Failed to drop table '%s': %v\n", name, err)
		}
	}
}

// dropCompanyTablesSQLite drops all "companyName$*" tables from SQLite
func (h *TablesHandler) dropCompanyTablesSQLite(companyName string) {
	rows, err := h.db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name LIKE ?`, companyName+"$%")
	if err != nil {
		fmt.Printf("Warning: Failed to find company tables for '%s': %v\n", companyName, err)
		return
	}
	defer rows.Close()

	var tablesToDrop []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			tablesToDrop = append(tablesToDrop, name)
		}
	}

	for _, name := range tablesToDrop {
		if _, err := h.db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS "%s"`, name)); err != nil {
			fmt.Printf("Warning: Failed to drop table '%s': %v\n", name, err)
		}
	}
}

// containsAny checks if slice contains any of the items
func containsAny(slice []string, items []string) bool {
	for _, s := range slice {
		for _, item := range items {
			if s == item {
				return true
			}
		}
	}
	return false
}
