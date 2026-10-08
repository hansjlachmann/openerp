package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
	apitypes "github.com/hansjlachmann/openerp/backend/api/types"
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/company"
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
	// onCompanyChanged is called after a company was renamed (newName set) or deleted
	// (newName ""), so sessions working in it can follow
	onCompanyChanged func(c *fiber.Ctx, oldName, newName string)
}

// OnCompanyChanged sets the function called after a company was renamed or deleted.
func (h *TablesHandler) OnCompanyChanged(fn func(c *fiber.Ctx, oldName, newName string)) {
	h.onCompanyChanged = fn
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
	// LazyURL is set when the related table is too large to send along: the dropdown
	// loads its rows on demand from this URL (GET …/lookup/:field?search=), Rows and
	// Simple stay empty
	LazyURL string `json:"lazy_url,omitempty"`
	Total   int    `json:"total,omitempty"` // rows in the related table (lazy lookups)
}

// lookupInlineLimit: related tables with at most this many rows are sent along with every
// list/card response (instant dropdowns); larger ones are loaded on demand. Sending all
// 10,000 customers with each Customer Ledger Entries window cost ~0.4 s per request.
const lookupInlineLimit = 200

// lookupPageSize is how many rows an on-demand dropdown loads per search
const lookupPageSize = 50

// LookupColumn represents a column in the lookup dropdown
type LookupColumn struct {
	Source string `json:"source"`
	Width  int    `json:"width"`
}

// getLookupValues fetches lookup values for table relation fields
func (h *TablesHandler) getLookupValues(tableName string, table ftables.Table, company string) map[string]*LookupData {
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

		// Large related table: no rows, the dropdown loads them on demand
		if total := relTable.Count(); total > lookupInlineLimit {
			if len(lookup.Columns) == 0 {
				lookup.Columns = []LookupColumn{{Source: relInfo.Field, Width: 150}}
			}
			lookup.Rows = nil
			lookup.Simple = nil
			lookup.LazyURL = fmt.Sprintf("/api/tables/%s/lookup/%s", url.PathEscape(tableName), url.PathEscape(fieldName))
			lookup.Total = total
			lookups[fieldName] = lookup
			continue
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

// LookupRows returns rows for an on-demand dropdown (lookups with lazy_url): rows of the
// table related to :field whose key or lookup columns contain search (case-insensitive),
// ordered by key, at most lookupPageSize. key=<value> returns the row with that key, if any.
// GET /api/tables/:table/lookup/:field?search=&offset=&key=
func (h *TablesHandler) LookupRows(c *fiber.Ctx) error {
	tableName := c.Params("table")
	fieldName := c.Params("field")
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
	relInfo, ok := table.GetTableRelationFields()[fieldName]
	if !ok {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.InvalidFields().Message(language)))
	}
	relFactory, ok := tables.GetTableFactory(relInfo.Table)
	if !ok {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.TableNotFound(relInfo.Table).Message(language)))
	}
	relTable := relFactory()
	relTable.InitWithDBType(h.db, company, h.dbType)

	// Columns shown in the dropdown (stored columns only), the key first
	columns := []string{relInfo.Field}
	for _, col := range relInfo.LookupColumns {
		if col.Source != relInfo.Field && ftables.IsQueryableColumn(relTable, col.Source) {
			columns = append(columns, col.Source)
		}
	}

	if key := c.Query("key", ""); key != "" {
		relTable.SetFilter(relInfo.Field, strings.ToUpper(key))
	} else if search := strings.TrimSpace(c.Query("search", "")); search != "" {
		relTable.SetSearch(columns, search)
	}
	relTable.SetCurrentKey(relInfo.Field)
	total := relTable.Count()
	relTable.SetPage(lookupPageSize, max(c.QueryInt("offset", 0), 0))

	rows := make([]map[string]interface{}, 0)
	if relTable.FindSet() {
		for {
			record := relTable.ToMap()
			row := map[string]interface{}{"_key": fmt.Sprint(record[relInfo.Field])}
			for _, col := range columns {
				row[col] = record[col]
			}
			rows = append(rows, row)
			if !relTable.Next() {
				break
			}
		}
	}
	return c.JSON(apitypes.NewSuccessResponse(map[string]interface{}{"rows": rows, "total": total}))
}

// getLookupValuesAsInterface converts lookup data to interface{} map for JSON serialization
func (h *TablesHandler) getLookupValuesAsInterface(tableName string, table ftables.Table, company string) map[string]interface{} {
	lookups := h.getLookupValues(tableName, table, company)
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
	lookups := h.getLookupValues(tableName, table, company)

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
		// Field names end up in SQL text: only accept real (non-sensitive) columns of this table
		if !ftables.IsQueryableColumn(table, sortBy) {
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
		// Field names end up in SQL text: only accept real (non-sensitive) columns of this table
		if !ftables.IsQueryableColumn(table, sortBy) {
			return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidSortField().Message(language)))
		}
		table.SetCurrentKey(sortBy)
	}
	switch c.Query("sort_order", "asc") {
	case "asc":
	case "desc":
		table.SetAscending(false)
	default:
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidSortField().Message(language)))
	}

	// Setup tables always have their single record present (BC-style)
	if table.IsSetupTable() {
		h.ensureSetupRecord(tableName, company)
	}

	// FlowFilters (e.g. Date Filter) the FlowFields apply
	if err := applyFlowFilters(c, table); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(err.Error()))
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
			if !ftables.IsQueryableColumn(table, f.Field) {
				return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidFilters().Message(language)))
			}
			table.SetFilter(f.Field, f.Expression)
		}
	}

	// Search box: case-insensitive "contains" over the given columns (any of them)
	if search := strings.TrimSpace(c.Query("search", "")); search != "" {
		var searchFields []string
		if err := json.Unmarshal([]byte(c.Query("search_fields", "[]")), &searchFields); err != nil {
			return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidFilters().Message(language)))
		}
		for _, f := range searchFields {
			if !ftables.IsQueryableColumn(table, f) {
				return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidFilters().Message(language)))
			}
		}
		table.SetSearch(searchFields, search)
	}

	// Pagination window (opt-in): offset/limit (list page windows) or page/page_size.
	// Then total is the full filtered count. Without either the entire result set is
	// returned (small tables, callers that need every row).
	const maxPageSize = 1000
	offset := max(c.QueryInt("offset", 0), 0)
	limit := min(max(c.QueryInt("limit", 0), 0), maxPageSize)
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	pageSize := min(max(c.QueryInt("page_size", 0), 0), maxPageSize)
	if limit == 0 && pageSize > 0 {
		limit, offset = pageSize, (page-1)*pageSize
	}

	total := 0
	if limit > 0 {
		total = table.Count() // full filtered count, before applying the window
		table.SetPage(limit, offset)
	}

	// Collect records (non-nil so an empty result serializes as [] rather than null,
	// e.g. when a requested page is past the end of the data)
	records := make([]map[string]interface{}, 0)
	if table.FindSet() {
		records = append(records, ftables.PublicMap(table))
		for table.Next() {
			records = append(records, ftables.PublicMap(table))
		}
	}

	// FlowFields for all rows at once (one grouped query per FlowField, not one query
	// per row and FlowField); only those requested when the client names its fields
	if calc := flowFieldsToCalc(table.GetFlowFields(), requestedFields); len(calc) > 0 {
		table.CalcFieldsForRecords(records, calc...)
	}

	// Get captions
	ts := i18n.GetInstance()
	captions := &apitypes.CaptionData{
		Table:      ts.TableCaption(tableName, language),
		Fields:     make(map[string]string),
		FieldTypes: make(map[string]string),
		Options:    make(map[string]map[string]string),
		Lookups:    h.getLookupValuesAsInterface(tableName, table, company),
	}

	// Add field captions and types from metadata
	for _, field := range table.GetFields() {
		if field.Sensitive {
			continue
		}
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

	// Pagination metadata: real values when a window was requested, otherwise the
	// legacy single-page shape (all records reported as page 1).
	respPage, respPageSize, respTotal, respOffset := 1, len(records), len(records), 0
	if limit > 0 {
		respPageSize, respTotal, respOffset = limit, total, offset
		if pageSize > 0 {
			respPage = page
		}
	}

	response := apitypes.NewSuccessResponseWithCaptions(map[string]interface{}{
		"records":   records,
		"total":     respTotal,
		"page":      respPage,
		"page_size": respPageSize,
		"offset":    respOffset,
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

	// FlowFilters (e.g. Date Filter) the FlowFields apply
	if err := applyFlowFilters(c, table); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(err.Error()))
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
		Lookups:    h.getLookupValuesAsInterface(tableName, table, company),
	}

	for _, field := range table.GetFields() {
		if field.Sensitive {
			continue
		}
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

	response := apitypes.NewSuccessResponseWithCaptions(ftables.PublicMap(table), captions)
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

	// Sensitive fields (e.g. password_hash) cannot be set through the API; a masked field
	// sent back as its placeholder keeps its stored value
	ftables.DropMaskedPlaceholders(table, data)
	if err := rejectSensitiveFields(table, data); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(err.Error()))
	}

	// A changed table relation field must point to an existing record
	if err := h.checkRelations(table, company, language, data); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(err.Error()))
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
			return c.Status(400).JSON(apitypes.NewErrorResponse(errorMessage(trigErr, language)))
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

	response := apitypes.NewSuccessResponse(ftables.PublicMap(table))
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
	_, oldKey := fullPrimaryKey(table)

	// Parse request body
	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidRequestBody().Message(language)))
	}

	// Sensitive fields (e.g. password_hash) cannot be set through the API; a masked field
	// sent back as its placeholder keeps its stored value
	ftables.DropMaskedPlaceholders(table, data)
	if err := rejectSensitiveFields(table, data); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(err.Error()))
	}

	// A changed table relation field must point to an existing record
	if err := h.checkRelations(table, company, language, data); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(err.Error()))
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

	// A rename to a key that another record already has (BC/NAV: "The record already exists")
	if newLookup, newKey := fullPrimaryKey(table); newKey != oldKey {
		if existing, err := h.getTable(tableName, company); err == nil && existing.Get(newLookup) {
			return c.Status(409).JSON(apitypes.NewErrorResponse(apperrors.DuplicateRecord(tableCaption, newKey).Message(language)))
		}
	}

	// Modify record — in a transaction, so a renamed key and the fields of other tables that
	// refer to it (BC/NAV Rename, done by the generated Modify) change together or not at all
	if ok, err := h.modifyInTransaction(table); !ok {
		if err != nil {
			return c.Status(500).JSON(apitypes.NewErrorResponse(err.Error()))
		}
		// A failed OnModify trigger is a business rule the user broke: show its message
		if trigErr := table.TriggerError(); trigErr != nil {
			return c.Status(400).JSON(apitypes.NewErrorResponse(errorMessage(trigErr, language)))
		}
		return c.Status(500).JSON(apitypes.NewErrorResponse(apperrors.ModifyFailed(tableCaption, id).Message(language)))
	}

	// A renamed company: sessions working in it must follow (cookie, session cache)
	if tableName == "Company" && h.onCompanyChanged != nil {
		if oldName, newName := fmt.Sprint(parseRecordKey(id, table)), table.GetPrimaryKeyValue(); oldName != newName {
			h.onCompanyChanged(c, oldName, newName)
		}
	}

	// Calculate FlowFields for response
	table.CalcFields(table.GetFlowFields()...)

	response := apitypes.NewSuccessResponse(ftables.PublicMap(table))
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

	// Delete record (a Company with all its tables and SIFT objects, in one transaction)
	deleted, txErr := h.deleteInTransaction(tableName, table)
	if txErr != nil {
		return c.Status(500).JSON(apitypes.NewErrorResponse(txErr.Error()))
	}
	if !deleted {
		// A failed OnDelete trigger is a business rule the user broke: show its message
		if trigErr := table.TriggerError(); trigErr != nil {
			return c.Status(400).JSON(apitypes.NewErrorResponse(errorMessage(trigErr, language)))
		}
		return c.Status(500).JSON(apitypes.NewErrorResponse(apperrors.DeleteFailed(tableCaption, id).Message(language)))
	}

	if tableName == "Company" && h.onCompanyChanged != nil {
		h.onCompanyChanged(c, table.GetPrimaryKeyValue(), "")
	}

	response := apitypes.NewSuccessResponse(nil)
	return c.JSON(response)
}

// rejectSensitiveFields returns an error when data sets a sensitive field (YAML sensitive:
// true). Such fields are set by the server only (User.password_hash from the virtual
// "password" field). A null value (an unchanged field) is allowed.
func rejectSensitiveFields(table ftables.Table, data map[string]interface{}) error {
	for name, value := range data {
		if value != nil && ftables.IsSensitive(table, name) {
			return fmt.Errorf("field %s cannot be set", name)
		}
	}
	return nil
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

	return c.JSON(apitypes.NewSuccessResponse(ftables.PublicMap(table)))
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

	// Sensitive fields can neither be validated (set) nor hydrated through the API
	if ftables.IsSensitive(table, req.Field) || rejectSensitiveFields(table, req.Record) != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidRequestBody().Message(language)))
	}

	// Hydrate from the in-progress record (plain assignment, no triggers)
	if req.Record != nil {
		ftables.DropMaskedPlaceholders(table, req.Record)
		table.FromMap(req.Record)
	}

	// A masked field sent back as its placeholder is unchanged: nothing to validate
	if ftables.IsMasked(table, req.Field) && req.Value == ftables.MaskedValue {
		return c.JSON(apitypes.NewSuccessResponse(ftables.PublicMap(table)))
	}

	// Validate field (runs OnValidate trigger)
	if err := table.ValidateField(req.Field, req.Value); err != nil {
		return c.JSON(apitypes.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	// Check table relation: verify the value exists in the related table
	if err := h.checkRelation(table, company, language, req.Field, req.Value); err != nil {
		return c.JSON(apitypes.APIResponse{Success: false, Error: err.Error()})
	}

	return c.JSON(apitypes.NewSuccessResponse(ftables.PublicMap(table)))
}

// modifyInTransaction runs table.Modify(true) in a database transaction (committed when it
// succeeds, rolled back otherwise) and then points the record back at the handler's
// connection. err is set only when the transaction itself fails.
func (h *TablesHandler) modifyInTransaction(table ftables.Table) (bool, error) {
	setter, ok := table.(interface{ SetDB(database.Executor) })
	if !ok {
		return table.Modify(true), nil
	}
	tx, err := h.db.Begin()
	if err != nil {
		return false, err
	}
	setter.SetDB(tx)
	defer setter.SetDB(h.db)
	if !table.Modify(true) {
		_ = tx.Rollback()
		return false, nil
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// deleteInTransaction deletes the record like table.Delete(true); deleting a Company also
// drops all its tables and SIFT objects, in the same transaction, so a failure keeps the
// company and its data. err is set only when the transaction itself fails.
func (h *TablesHandler) deleteInTransaction(tableName string, table ftables.Table) (bool, error) {
	setter, ok := table.(interface{ SetDB(database.Executor) })
	if tableName != "Company" || !ok {
		return table.Delete(true), nil
	}
	companyName := table.GetPrimaryKeyValue()
	tx, err := h.db.Begin()
	if err != nil {
		return false, err
	}
	setter.SetDB(tx)
	defer setter.SetDB(h.db)
	if !table.Delete(true) {
		_ = tx.Rollback()
		return false, nil
	}
	if err := company.DropObjects(tx, h.dbType, companyName); err != nil {
		_ = tx.Rollback()
		return false, fmt.Errorf("delete company %s: %w", companyName, err)
	}
	return true, tx.Commit()
}

// errorMessage is an error as shown to the user: AppErrors (e.g. from table triggers) in
// the user's language, others as they are.
func errorMessage(err error, language string) string {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr.Message(language)
	}
	return err.Error()
}

// checkRelation returns an error when value (a non-empty key) does not exist in the
// table related to field. Fields without a table relation always pass.
func (h *TablesHandler) checkRelation(table ftables.Table, company, language, field string, value interface{}) error {
	valueStr, ok := value.(string)
	if !ok || valueStr == "" {
		return nil
	}
	relInfo, hasRelation := table.GetTableRelationFields()[field]
	if !hasRelation {
		return nil
	}
	relTable, err := h.getTable(relInfo.Table, company)
	if err != nil {
		return nil // related table not registered: nothing to check against
	}
	if !relTable.Get(valueStr) {
		relCaption := i18n.GetInstance().TableCaption(relInfo.Table, language)
		return fmt.Errorf("%s '%s' does not exist", relCaption, valueStr)
	}
	return nil
}

// checkRelations checks every table relation field in data whose value differs from the
// record's current value (insert/modify). The page's dropdowns are not the only guard:
// on-demand dropdowns (large related tables) accept a typed key and rely on this check.
func (h *TablesHandler) checkRelations(table ftables.Table, company, language string, data map[string]interface{}) error {
	current := table.ToMap()
	for field := range table.GetTableRelationFields() {
		value, ok := data[field]
		if !ok || fmt.Sprint(current[field]) == fmt.Sprint(value) {
			continue
		}
		if err := h.checkRelation(table, company, language, field, value); err != nil {
			return err
		}
	}
	return nil
}

// applyFlowFilters sets the FlowFilter fields (NAV FieldClass FlowFilter, e.g. Date Filter)
// from the flow_filters query parameter, JSON [{field, expression}]. The expression is
// validated per the field's kind (dates in ISO); unknown fields and invalid expressions
// are errors.
func applyFlowFilters(c *fiber.Ctx, table ftables.Table) error {
	param := c.Query("flow_filters", "")
	if param == "" {
		return nil
	}
	var filters []struct {
		Field      string `json:"field"`
		Expression string `json:"expression"`
	}
	if err := json.Unmarshal([]byte(param), &filters); err != nil {
		return fmt.Errorf("invalid flow_filters parameter")
	}
	for _, f := range filters {
		if err := table.SetFlowFilter(f.Field, f.Expression); err != nil {
			return fmt.Errorf("%s: %v", f.Field, err)
		}
	}
	return nil
}

// flowFieldsToCalc returns the FlowFields a list request needs: all of them when the
// client sends no field list, otherwise only the requested ones.
func flowFieldsToCalc(flowFields, requestedFields []string) []string {
	if len(requestedFields) == 0 {
		return flowFields
	}
	var calc []string
	for _, f := range flowFields {
		if slices.Contains(requestedFields, f) {
			calc = append(calc, f)
		}
	}
	return calc
}
