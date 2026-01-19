package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	apitypes "github.com/hansjlachmann/openerp/backend/api/types"
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/filters"
	"github.com/hansjlachmann/openerp/backend/foundation/i18n"
	"github.com/hansjlachmann/openerp/backend/foundation/session"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
)

// TablesHandler handles table-related API requests
type TablesHandler struct {
	db     *sql.DB
	dbType database.DBType
}

// NewTablesHandler creates a new tables handler (defaults to SQLite)
func NewTablesHandler(db *sql.DB) *TablesHandler {
	return NewTablesHandlerWithDBType(db, database.DBTypeSQLite)
}

// NewTablesHandlerWithDBType creates a new tables handler with explicit database type
func NewTablesHandlerWithDBType(db *sql.DB, dbType database.DBType) *TablesHandler {
	return &TablesHandler{db: db, dbType: dbType}
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
	sess := session.GetCurrent()

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
	sess := session.GetCurrent()

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
	sess := session.GetCurrent()

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
		table.SetCurrentKey(sortBy)
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

		// Apply BC-style filters
		for _, f := range apiFilters {
			table.SetFilter(f.Field, f.Expression)
		}
	}

	// Collect records
	var records []map[string]interface{}
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
		Table:   ts.TableCaption(tableName, language),
		Fields:  make(map[string]string),
		Options: make(map[string]map[string]string),
		Lookups: h.getLookupValuesAsInterface(table, company),
	}

	// Add field captions from metadata
	for _, field := range table.GetFields() {
		captions.Fields[field.Name] = ts.FieldCaption(tableName, field.Name, language)
	}

	// Add option field values
	for fieldName, options := range table.GetOptionFields() {
		optionMap := make(map[string]string)
		for i, opt := range options {
			optionMap[fmt.Sprintf("%d", i)] = opt
		}
		captions.Options[fieldName] = optionMap
	}

	response := apitypes.NewSuccessResponseWithCaptions(map[string]interface{}{
		"records":   records,
		"total":     len(records),
		"page":      1,
		"page_size": len(records),
	}, captions)
	return c.JSON(response)
}

// GetRecord returns a single record by ID
// GET /api/tables/:table/card/:id
func (h *TablesHandler) GetRecord(c *fiber.Ctx) error {
	tableName := c.Params("table")
	id := c.Params("id")
	sess := session.GetCurrent()

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

	// Get record by primary key
	if !table.Get(id) {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.RecordNotFound(tableCaption, id).Message(language)))
	}

	// Calculate FlowFields
	table.CalcFields(table.GetFlowFields()...)

	// Get captions
	ts := i18n.GetInstance()
	captions := &apitypes.CaptionData{
		Table:   ts.TableCaption(tableName, language),
		Fields:  make(map[string]string),
		Options: make(map[string]map[string]string),
		Lookups: h.getLookupValuesAsInterface(table, company),
	}

	for _, field := range table.GetFields() {
		captions.Fields[field.Name] = ts.FieldCaption(tableName, field.Name, language)
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
	sess := session.GetCurrent()

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

	// Get FlowFields to skip during validation (they're calculated, not stored)
	flowFields := make(map[string]bool)
	for _, ff := range table.GetFlowFields() {
		flowFields[ff] = true
	}

	// Validate and set each field (runs OnValidate triggers for table relations, etc.)
	for fieldName, value := range data {
		// Skip nil values - frontend may send null for unchanged fields
		if value == nil {
			continue
		}
		// Skip FlowFields - they're calculated, not validated
		if flowFields[fieldName] {
			continue
		}
		// Skip virtual password field for User table - handled separately below
		if tableName == "User" && fieldName == "password" {
			continue
		}
		if err := table.ValidateField(fieldName, value); err != nil {
			return c.Status(400).JSON(apitypes.NewErrorResponse(err.Error()))
		}
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

	// Check if record already exists
	existingTable, _ := h.getTable(tableName, company)
	if existingTable.Get(pkValue) {
		return c.Status(409).JSON(apitypes.NewErrorResponse(apperrors.DuplicateRecord(tableCaption, pkValue).Message(language)))
	}

	// Insert record
	if !table.Insert(true) {
		return c.Status(500).JSON(apitypes.NewErrorResponse(apperrors.InsertFailed(tableCaption).Message(language)))
	}

	response := apitypes.NewSuccessResponse(table.ToMap())
	return c.JSON(response)
}

// ModifyRecord updates an existing record
// PUT /api/tables/:table/modify/:id
func (h *TablesHandler) ModifyRecord(c *fiber.Ctx) error {
	tableName := c.Params("table")
	id := c.Params("id")
	sess := session.GetCurrent()

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	company := sess.GetCompany()
	language := sess.GetLanguage()
	tableCaption := i18n.GetInstance().TableCaption(tableName, language)

	// Check for empty ID
	if id == "" {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.EmptyPrimaryKey(tableCaption).Message(language)))
	}

	// Create table instance
	table, err := h.getTable(tableName, company)
	if err != nil {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.TableNotFound(tableName).Message(language)))
	}

	// Get existing record
	if !table.Get(id) {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.RecordNotFound(tableCaption, id).Message(language)))
	}

	// Parse request body
	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidRequestBody().Message(language)))
	}

	// Get FlowFields to skip during validation (they're calculated, not stored)
	flowFields := make(map[string]bool)
	for _, ff := range table.GetFlowFields() {
		flowFields[ff] = true
	}

	// Validate and update each field (runs OnValidate triggers for table relations, etc.)
	for fieldName, value := range data {
		// Skip nil values - frontend may send null for unchanged fields
		if value == nil {
			continue
		}
		// Skip FlowFields - they're calculated, not validated
		if flowFields[fieldName] {
			continue
		}
		// Skip virtual password field for User table - handled separately below
		if tableName == "User" && fieldName == "password" {
			continue
		}
		if err := table.ValidateField(fieldName, value); err != nil {
			return c.Status(400).JSON(apitypes.NewErrorResponse(err.Error()))
		}
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
	sess := session.GetCurrent()

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	company := sess.GetCompany()
	language := sess.GetLanguage()
	tableCaption := i18n.GetInstance().TableCaption(tableName, language)

	// Check for empty ID
	if id == "" {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.EmptyPrimaryKey(tableCaption).Message(language)))
	}

	// Create table instance
	table, err := h.getTable(tableName, company)
	if err != nil {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.TableNotFound(tableName).Message(language)))
	}

	// Get existing record
	if !table.Get(id) {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.RecordNotFound(tableCaption, id).Message(language)))
	}

	// Delete record
	if !table.Delete(true) {
		return c.Status(500).JSON(apitypes.NewErrorResponse(apperrors.DeleteFailed(tableCaption, id).Message(language)))
	}

	response := apitypes.NewSuccessResponse(nil)
	return c.JSON(response)
}

// ValidateField validates a single field value
// POST /api/tables/:table/validate
func (h *TablesHandler) ValidateField(c *fiber.Ctx) error {
	tableName := c.Params("table")
	sess := session.GetCurrent()

	if sess == nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	company := sess.GetCompany()
	language := sess.GetLanguage()

	// Parse request body
	var req struct {
		Field string      `json:"field"`
		Value interface{} `json:"value"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(apitypes.NewErrorResponse(apperrors.InvalidRequestBody().Message(language)))
	}

	// Create table instance
	table, err := h.getTable(tableName, company)
	if err != nil {
		return c.Status(404).JSON(apitypes.NewErrorResponse(apperrors.TableNotFound(tableName).Message(language)))
	}

	// Validate field
	if err := table.ValidateField(req.Field, req.Value); err != nil {
		return c.JSON(apitypes.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	return c.JSON(apitypes.APIResponse{
		Success: true,
	})
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

// Legacy field captions helper (kept for backward compatibility, uses metadata now)
func (h *TablesHandler) addFieldCaptions(tableName, language string, captions *apitypes.CaptionData) {
	ts := i18n.GetInstance()

	// Get field names from a temporary table instance
	factory, ok := tables.GetTableFactory(tableName)
	if !ok {
		return
	}

	table := factory()
	for _, field := range table.GetFields() {
		captions.Fields[field.Name] = ts.FieldCaption(tableName, field.Name, language)
	}
}

// applyFilters applies BC-style filters to a table (utility function)
func applyFilters(table ftables.Table, filterExpressions []filters.FilterExpression) {
	for _, f := range filterExpressions {
		table.SetFilter(f.Field, f.Expression)
	}
}

// getTableForTimestamps provides timestamp initialization for User tables
// This is a table-specific hook that can't be fully generic
func initializeUserTimestamps(table ftables.Table) {
	if userTable, ok := table.(*tables.User); ok {
		now := time.Now()
		userTable.Created_at = types.NewDateTimeFromTime(now)
		userTable.Last_login = types.NewDateTimeFromTime(now)
	}
}
